package wasmpack

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Wrap takes the wasm_exec.js file from the go installation and wraps the packed wasm bytes in it, returning the
// resulting JavaScript code. If name is not empty, instead of running the wasm immediately it will be assigned to
// a global function with the provided name that can be called with an optional env object and args array.
func Wrap(name string, wasm []byte) ([]byte, error) {
	cmd := exec.Command("go", "env", "GOROOT")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}
	goRoot := strings.Replace(string(out), "\n", "", -1)
	goRoot = strings.Replace(goRoot, "\r", "", -1)
	goRoot = strings.TrimSpace(goRoot)
	if string(goRoot) == "" {
		return nil, fmt.Errorf("GOROOT is empty")
	}
	for _, dir := range []string{"lib", "misc"} {
		if code, err := os.ReadFile(filepath.Join(goRoot, dir, "wasm", "wasm_exec.js")); err == nil {
			pre := ""
			post := ""
			if name != "" {
				pre = "globalThis[\"" + name + "\"] = (env, args) => {\n" +
					"if(env && typeof env === 'object') go.env = env;\n" +
					"if(args && args.length > 0) go.argv.push(args);\n"
				post = "}\n"
			}
			code = bytes.Replace(code, []byte("globalThis.Go ="), []byte("\tif(globalThis[\""+name+"\"]) throw new Error('global function \""+name+"\" already exists');\n"+
				pre+
				"const go = new "), 1)
			code = bytes.Replace(code, []byte("})();"), []byte("WebAssembly.instantiate("+string(wasm)+", go.importObject).then(({instance}) => {\n"+
				"go.run(instance);\n"+
				"})\n"+
				post+
				"})();\n"), 1)
			return code, nil
		}
	}
	return nil, fmt.Errorf("wasm_exec.js not found in %s", goRoot)
}

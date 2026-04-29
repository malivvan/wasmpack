package wasmpack

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// readWasmExec locates and reads wasm_exec.js from the active Go installation.
func readWasmExec() ([]byte, error) {
	cmd := exec.Command("go", "env", "GOROOT")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}
	goRoot := strings.TrimSpace(string(out))
	if goRoot == "" {
		return nil, fmt.Errorf("GOROOT is empty")
	}
	for _, dir := range []string{"lib", "misc"} {
		if code, err := os.ReadFile(filepath.Join(goRoot, dir, "wasm", "wasm_exec.js")); err == nil {
			return code, nil
		}
	}
	return nil, fmt.Errorf("wasm_exec.js not found in %s", goRoot)
}

// WrapIIFE wraps the packed wasm bytes in wasm_exec.js as a self-executing IIFE.
// The WebAssembly module starts running as soon as the script is loaded.
// packed is the JavaScript expression returned by Pack.
func WrapIIFE(packed []byte) ([]byte, error) {
	code, err := readWasmExec()
	if err != nil {
		return nil, err
	}
	// Inline-instantiate the Go class anonymously: `globalThis.Go = class {` → `const go = new class {`
	code = bytes.Replace(code, []byte("globalThis.Go ="), []byte("const go = new "), 1)
	// Replace the IIFE closing with wasm loading and auto-execution.
	code = bytes.Replace(code, []byte("})();"), []byte(
		"WebAssembly.instantiate("+string(packed)+", go.importObject).then(({instance}) => {\n"+
			"\tgo.run(instance);\n"+
			"});\n"+
			"})();\n"), 1)
	return code, nil
}

// WrapESM wraps the packed wasm bytes in wasm_exec.js as an ES module.
// The module exports a single `run(env, args)` function that instantiates and
// starts the WebAssembly module, returning a Promise.
// packed is the JavaScript expression returned by Pack.
func WrapESM(packed []byte) ([]byte, error) {
	code, err := readWasmExec()
	if err != nil {
		return nil, err
	}
	// Replace the IIFE closing with the exported run function.
	// globalThis.Go is left as-is so the Go runtime is still registered;
	// the exported run function creates a fresh instance on each call.
	code = bytes.Replace(code, []byte("})();"), []byte(
		"})();\n\n"+
			"export function run(env, args) {\n"+
			"\tconst go = new globalThis.Go();\n"+
			"\tif (env && typeof env === 'object') Object.assign(go.env, env);\n"+
			"\tif (args && Array.isArray(args)) go.argv = go.argv.concat(args);\n"+
			"\treturn WebAssembly.instantiate(\n"+
			"\t\t"+string(packed)+",\n"+
			"\t\tgo.importObject\n"+
			"\t).then(({ instance }) => {\n"+
			"\t\treturn go.run(instance);\n"+
			"\t});\n"+
			"}\n"), 1)
	return code, nil
}

// WrapCJS wraps the packed wasm bytes in wasm_exec.js as a CommonJS module.
// The module exports a single `run(env, args)` function that instantiates and
// starts the WebAssembly module, returning a Promise.
// packed is the JavaScript expression returned by Pack.
func WrapCJS(packed []byte) ([]byte, error) {
	code, err := readWasmExec()
	if err != nil {
		return nil, err
	}
	// Replace the IIFE closing with the module.exports assignment.
	code = bytes.Replace(code, []byte("})();"), []byte(
		"})();\n\n"+
			"module.exports = function run(env, args) {\n"+
			"\tconst go = new globalThis.Go();\n"+
			"\tif (env && typeof env === 'object') Object.assign(go.env, env);\n"+
			"\tif (args && Array.isArray(args)) go.argv = go.argv.concat(args);\n"+
			"\treturn WebAssembly.instantiate(\n"+
			"\t\t"+string(packed)+",\n"+
			"\t\tgo.importObject\n"+
			"\t).then(({ instance }) => {\n"+
			"\t\treturn go.run(instance);\n"+
			"\t});\n"+
			"};\n"), 1)
	return code, nil
}

// WrapHTML wraps the packed wasm IIFE inside a minimal HTML page.
// The <script> tag uses defer so it executes after HTML is parsed.
// packed is the JavaScript expression returned by Pack.
func WrapHTML(packed []byte) ([]byte, error) {
	js, err := WrapIIFE(packed)
	if err != nil {
		return nil, err
	}
	html := "<!DOCTYPE html>\n<html>\n<head>\n<script defer>\n" + string(js) + "\n</script>\n</head>\n<body></body>\n</html>\n"
	return []byte(html), nil
}

// Wrap takes the wasm_exec.js file from the go installation and wraps the packed wasm bytes in it, returning the
// resulting JavaScript code. If name is not empty, instead of running the wasm immediately it will be assigned to
// a global function with the provided name that can be called with an optional env object and args array.
func Wrap(name string, wasm []byte) ([]byte, error) {
	code, err := readWasmExec()
	if err != nil {
		return nil, err
	}
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

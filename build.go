package wasmpack

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"
)

// Build compiles the Go code at the specified path into a WebAssembly (WASM) binary. The resulting WASM file is read
// into memory and returned as a byte slice. If optimization options are provided, the function will also optimize the
// generated WASM file using the wasm-opt tool. The temporary WASM file created during the build process is removed
// after reading its contents.
func Build(path string, garble string, wasmopt string) ([]byte, error) {
	out := filepath.Join(os.TempDir(), "wasmpack"+strconv.Itoa(time.Now().Nanosecond())+".wasm")
	name := "go"
	args := []string{"build", "-o", out, "-ldflags=-s -w", "-trimpath", path}
	if garble != "" {
		name = "garble"
		args = append(splitArgs(garble), args...)
	}
	cmd := exec.Command(name, args...)
	cmd.Env = append([]string{"GOOS=js", "GOARCH=wasm", "CGO_ENABLED=0"}, os.Environ()...)
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("error building wasm: %w: %s", err, string(out))
	}
	if wasmopt != "" {
		err = Opt(out, wasmopt)
		if err != nil {
			return nil, err
		}
	}
	wasm, err := os.ReadFile(out)
	if err != nil {
		return nil, fmt.Errorf("error reading wasm file: %w", err)
	}
	err = os.Remove(out)
	if err != nil {
		return nil, fmt.Errorf("error removing wasm file: %w", err)
	}
	return wasm, nil
}

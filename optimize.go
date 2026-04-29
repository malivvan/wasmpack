package wasmpack

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// WasmOpt runs wasm-opt on the wasm file at path with the given flag slice,
// writing the result back to the same path.
// Example flags: []string{"-O2"}, []string{"-O4", "--enable-simd"}.
func WasmOpt(path string, flags []string) error {
	args := append(flags, "-o", path, path)
	cmd := exec.Command("wasm-opt", args...)
	cmd.Env = os.Environ()
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("wasm-opt failed: %w", err)
	}
	return nil
}

// splitArgs splits a space-separated flag string into individual tokens,
// trimming surrounding whitespace from each token.
func splitArgs(s string) []string {
	var args []string
	for _, arg := range strings.Split(s, " ") {
		if t := strings.TrimSpace(arg); t != "" {
			args = append(args, t)
		}
	}
	return args
}

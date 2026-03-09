package wasmpack

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Opt optimizes the given wasm file using wasm-opt with the provided options. The optimized wasm file is written
// back to the same path. The opt string is expected to be a space-separated list of options for wasm-opt.
func Opt(path string, wasmopt string) error {
	args := append(splitArgs(wasmopt), "-o", path, path)
	cmd := exec.Command("wasm-opt", args...)
	cmd.Env = os.Environ()
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("error building wasm: %w: %s", err, string(path))
	}
	return nil
}

func splitArgs(s string) []string {
	var args []string
	for _, arg := range strings.Split(s, " ") {
		args = append(args, strings.TrimSpace(arg))
	}
	return args
}

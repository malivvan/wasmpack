package wasmpack

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"
)

// Build compiles the Go package at path into a WebAssembly binary and returns
// the raw bytes. The binary is written to a temp file, optionally optimised
// with wasm-opt, read back, and then removed.
//
// When useGarble is true the build is performed via garble using garbleCfg.
// When useTinygo is true the build is performed via tinygo using tinygoCfg.
// useGarble and useTinygo are mutually exclusive; useGarble takes precedence.
// When useWasmOpt is true the wasm binary is optimised via wasm-opt using wasmOptCfg.
func Build(path string, useGarble bool, garbleCfg GarbleConfig, useTinygo bool, tinygoCfg TinygoConfig, useWasmOpt bool, wasmOptCfg WasmOptConfig) ([]byte, error) {
	out := filepath.Join(os.TempDir(), "wasmpack"+strconv.Itoa(time.Now().Nanosecond())+".wasm")

	var name string
	var args []string

	switch {
	case useGarble:
		name = "garble"
		args = append(garbleCfg.ToArgs(), "build", "-o", out, "-ldflags=-s -w", "-trimpath", path)
	case useTinygo:
		name = "tinygo"
		args = append(tinygoCfg.ToArgs(), "build", "-o", out, path)
	default:
		name = "go"
		args = []string{"build", "-o", out, "-ldflags=-s -w", "-trimpath", path}
	}

	cmd := exec.Command(name, args...)
	cmd.Env = append([]string{"GOOS=js", "GOARCH=wasm", "CGO_ENABLED=0"}, os.Environ()...)
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("build failed: %w", err)
	}
	if useWasmOpt {
		if err := WasmOpt(out, wasmOptCfg.ToArgs()); err != nil {
			_ = os.Remove(out)
			return nil, err
		}
	}
	wasm, err := os.ReadFile(out)
	if err != nil {
		return nil, fmt.Errorf("reading compiled wasm: %w", err)
	}
	if err := os.Remove(out); err != nil {
		return nil, fmt.Errorf("removing temp wasm file: %w", err)
	}
	return wasm, nil
}

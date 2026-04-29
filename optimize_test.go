package wasmpack

import (
	"testing"
)

// ---------- splitArgs ----------

func TestSplitArgs_Empty(t *testing.T) {
	args := splitArgs("")
	if len(args) != 0 {
		t.Errorf("expected no args, got %v", args)
	}
}

func TestSplitArgs_WhitespaceOnly(t *testing.T) {
	args := splitArgs("   ")
	if len(args) != 0 {
		t.Errorf("expected no args from whitespace-only input, got %v", args)
	}
}

func TestSplitArgs_Single(t *testing.T) {
	args := splitArgs("-O2")
	if len(args) != 1 || args[0] != "-O2" {
		t.Errorf("expected [-O2], got %v", args)
	}
}

func TestSplitArgs_Multiple(t *testing.T) {
	args := splitArgs("-O2 --enable-simd -v")
	want := []string{"-O2", "--enable-simd", "-v"}
	if len(args) != len(want) {
		t.Fatalf("expected %v, got %v", want, args)
	}
	for i, w := range want {
		if args[i] != w {
			t.Errorf("arg[%d]: expected %q, got %q", i, w, args[i])
		}
	}
}

func TestSplitArgs_ExtraSpaces(t *testing.T) {
	args := splitArgs("  -a   -b  ")
	if len(args) != 2 {
		t.Fatalf("expected 2 args, got %v", args)
	}
	if args[0] != "-a" || args[1] != "-b" {
		t.Errorf("unexpected args: %v", args)
	}
}

// ---------- WasmOpt ----------

func TestWasmOpt_SkipIfNotInstalled(t *testing.T) {
	// WasmOpt requires an external binary; skip if unavailable.
	// We intentionally pass a non-existent path to test the error path.
	err := WasmOpt("/nonexistent/file.wasm", []string{"-O2"})
	if err == nil {
		t.Error("expected an error when calling wasm-opt on a non-existent file, got nil")
	}
}


package wasmpack

import (
	"os"
	"testing"
)

// TestBuild_Sample compiles the sample package as an integration test.
// Run with: go test -run TestBuild_Sample (skipped under -short).
func TestBuild_Sample(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration build test in short mode")
	}
	wasm, err := Build(
		"./sample",
		false, GarbleConfig{},
		false, TinygoConfig{},
		false, WasmOptConfig{},
	)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}
	if len(wasm) == 0 {
		t.Fatal("Build returned empty wasm")
	}
	// A valid wasm binary starts with the magic bytes 0x00 0x61 0x73 0x6d
	if len(wasm) < 4 || wasm[0] != 0x00 || wasm[1] != 0x61 || wasm[2] != 0x73 || wasm[3] != 0x6d {
		t.Errorf("output does not start with wasm magic bytes; got %x", wasm[:min(8, len(wasm))])
	}
}

// TestBuild_InvalidPackage verifies that Build returns an error for an invalid path.
func TestBuild_InvalidPackage(t *testing.T) {
	_, err := Build(
		"./nonexistent_package_xyz",
		false, GarbleConfig{},
		false, TinygoConfig{},
		false, WasmOptConfig{},
	)
	if err == nil {
		t.Error("expected error for non-existent package, got nil")
	}
}

// TestBuild_FromPrecompiledWasm verifies Pack+Wrap end-to-end using a
// pre-compiled .wasm file (if present from a prior build).
func TestBuild_PackAndWrapSample(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	wasm, err := Build(
		"./sample",
		false, GarbleConfig{},
		false, TinygoConfig{},
		false, WasmOptConfig{},
	)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	packed, err := Pack(wasm)
	if err != nil {
		t.Fatalf("Pack failed: %v", err)
	}

	// IIFE
	js, err := WrapIIFE(packed)
	if err != nil {
		t.Fatalf("WrapIIFE failed: %v", err)
	}
	if len(js) == 0 {
		t.Fatal("WrapIIFE returned empty JS")
	}

	// HTML
	html, err := WrapHTML(packed)
	if err != nil {
		t.Fatalf("WrapHTML failed: %v", err)
	}
	if len(html) == 0 {
		t.Fatal("WrapHTML returned empty HTML")
	}

	// Minify the IIFE output
	minified, err := Minify(js)
	if err != nil {
		t.Fatalf("Minify failed: %v", err)
	}
	if len(minified) >= len(js) {
		t.Logf("note: minified size (%d) not smaller than original (%d)", len(minified), len(js))
	}
}

// TestBuild_WriteAndReadBack ensures the full pack pipeline produces a valid
// output file when written to disk.
func TestBuild_WriteIIFE(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	wasm, err := Build(
		"./sample",
		false, GarbleConfig{},
		false, TinygoConfig{},
		false, WasmOptConfig{},
	)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}
	packed, err := Pack(wasm)
	if err != nil {
		t.Fatalf("Pack failed: %v", err)
	}
	js, err := WrapIIFE(packed)
	if err != nil {
		t.Fatalf("WrapIIFE failed: %v", err)
	}

	tmp, err := os.CreateTemp(t.TempDir(), "output-*.js")
	if err != nil {
		t.Fatalf("CreateTemp: %v", err)
	}
	defer tmp.Close()

	if _, err := tmp.Write(js); err != nil {
		t.Fatalf("writing output: %v", err)
	}

	info, err := tmp.Stat()
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if info.Size() == 0 {
		t.Fatal("output file is empty")
	}
	t.Logf("IIFE output: %s  (%d bytes)", tmp.Name(), info.Size())
}


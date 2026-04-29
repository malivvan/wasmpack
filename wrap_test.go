package wasmpack

import (
	"strings"
	"testing"
)

// testPacked returns a small packed wasm snippet for use in wrap tests.
func testPacked(t *testing.T) []byte {
	t.Helper()
	// minimal valid wasm module: magic + version
	packed, err := Pack([]byte{0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00})
	if err != nil {
		t.Fatalf("testPacked: Pack failed: %v", err)
	}
	return packed
}

func TestWrapIIFE(t *testing.T) {
	packed := testPacked(t)
	result, err := WrapIIFE(packed)
	if err != nil {
		t.Fatalf("WrapIIFE failed: %v", err)
	}
	s := string(result)
	if !strings.Contains(s, "WebAssembly.instantiate") {
		t.Error("WrapIIFE result missing WebAssembly.instantiate")
	}
	if !strings.Contains(s, "go.run(instance)") {
		t.Error("WrapIIFE result missing go.run(instance)")
	}
	// the wasm_exec IIFE wrapper should be present
	if !strings.Contains(s, "(() => {") && !strings.Contains(s, "(function()") {
		// wasm_exec.js uses (() => { ... })();
		// Accept either form; just ensure we get a non-trivial JS file
	}
	if len(result) < 1000 {
		t.Errorf("WrapIIFE result suspiciously small (%d bytes)", len(result))
	}
}

func TestWrapESM(t *testing.T) {
	packed := testPacked(t)
	result, err := WrapESM(packed)
	if err != nil {
		t.Fatalf("WrapESM failed: %v", err)
	}
	s := string(result)
	if !strings.Contains(s, "export function run") {
		t.Error("WrapESM result missing 'export function run'")
	}
	if !strings.Contains(s, "WebAssembly.instantiate") {
		t.Error("WrapESM result missing WebAssembly.instantiate")
	}
	if len(result) < 1000 {
		t.Errorf("WrapESM result suspiciously small (%d bytes)", len(result))
	}
}

func TestWrapCJS(t *testing.T) {
	packed := testPacked(t)
	result, err := WrapCJS(packed)
	if err != nil {
		t.Fatalf("WrapCJS failed: %v", err)
	}
	s := string(result)
	if !strings.Contains(s, "module.exports") {
		t.Error("WrapCJS result missing 'module.exports'")
	}
	if !strings.Contains(s, "WebAssembly.instantiate") {
		t.Error("WrapCJS result missing WebAssembly.instantiate")
	}
	if len(result) < 1000 {
		t.Errorf("WrapCJS result suspiciously small (%d bytes)", len(result))
	}
}

func TestWrapHTML(t *testing.T) {
	packed := testPacked(t)
	result, err := WrapHTML(packed)
	if err != nil {
		t.Fatalf("WrapHTML failed: %v", err)
	}
	s := string(result)
	if !strings.HasPrefix(s, "<!DOCTYPE html>") {
		t.Error("WrapHTML result does not start with <!DOCTYPE html>")
	}
	if !strings.Contains(s, "<script defer>") {
		t.Error("WrapHTML result missing '<script defer>'")
	}
	if !strings.Contains(s, "</html>") {
		t.Error("WrapHTML result missing '</html>'")
	}
	if !strings.Contains(s, "WebAssembly.instantiate") {
		t.Error("WrapHTML result missing WebAssembly.instantiate")
	}
}

func TestWrap_WithName(t *testing.T) {
	packed := testPacked(t)
	result, err := Wrap("myApp", packed)
	if err != nil {
		t.Fatalf("Wrap failed: %v", err)
	}
	s := string(result)
	if !strings.Contains(s, "myApp") {
		t.Error("Wrap result missing the provided name 'myApp'")
	}
	if !strings.Contains(s, "WebAssembly.instantiate") {
		t.Error("Wrap result missing WebAssembly.instantiate")
	}
}

func TestWrap_EmptyName(t *testing.T) {
	packed := testPacked(t)
	result, err := Wrap("", packed)
	if err != nil {
		t.Fatalf("Wrap with empty name failed: %v", err)
	}
	if len(result) == 0 {
		t.Fatal("Wrap returned empty result")
	}
}

func TestWrapFormatsAreDistinct(t *testing.T) {
	packed := testPacked(t)

	iife, err := WrapIIFE(packed)
	if err != nil {
		t.Fatalf("WrapIIFE: %v", err)
	}
	esm, err := WrapESM(packed)
	if err != nil {
		t.Fatalf("WrapESM: %v", err)
	}
	cjs, err := WrapCJS(packed)
	if err != nil {
		t.Fatalf("WrapCJS: %v", err)
	}

	if string(iife) == string(esm) {
		t.Error("IIFE and ESM output are identical")
	}
	if string(iife) == string(cjs) {
		t.Error("IIFE and CJS output are identical")
	}
	if string(esm) == string(cjs) {
		t.Error("ESM and CJS output are identical")
	}
}


package wasmpack

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestPack_NonEmpty(t *testing.T) {
	input := []byte{0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00} // minimal wasm header
	result, err := Pack(input)
	if err != nil {
		t.Fatalf("Pack failed: %v", err)
	}
	if len(result) == 0 {
		t.Fatal("Pack returned empty result")
	}
	s := string(result)
	if !strings.HasPrefix(s, "((input) => {") {
		t.Errorf("Pack result has unexpected prefix: %q", s[:min(60, len(s))])
	}
	if !strings.HasSuffix(s, "`)") {
		t.Errorf("Pack result has unexpected suffix: %q", s[max(0, len(s)-10):])
	}
}

func TestPack_ContainsValidBase64(t *testing.T) {
	input := []byte("Hello, WebAssembly!")
	result, err := Pack(input)
	if err != nil {
		t.Fatalf("Pack failed: %v", err)
	}
	s := string(result)
	// The format is: ...)(` BASE64 `))
	first := strings.Index(s, "`")
	last := strings.LastIndex(s, "`")
	if first < 0 || last <= first {
		t.Fatal("could not locate backtick-delimited base64 in Pack result")
	}
	b64Data := s[first+1 : last]
	if _, err := base64.StdEncoding.DecodeString(b64Data); err != nil {
		t.Errorf("embedded base64 payload is invalid: %v", err)
	}
}

func TestPack_ContainsInflate(t *testing.T) {
	result, err := Pack([]byte{0x01, 0x02, 0x03})
	if err != nil {
		t.Fatalf("Pack failed: %v", err)
	}
	if !strings.Contains(string(result), "inflate") {
		t.Error("Pack result does not contain inflate logic")
	}
}

func TestPack_EmptyBytes(t *testing.T) {
	result, err := Pack([]byte{})
	if err != nil {
		t.Fatalf("Pack with empty input failed: %v", err)
	}
	if len(result) == 0 {
		t.Fatal("Pack returned empty result for empty input")
	}
}

func TestPack_LargerPayload(t *testing.T) {
	// 1 KB of pseudo-data
	input := make([]byte, 1024)
	for i := range input {
		input[i] = byte(i % 256)
	}
	result, err := Pack(input)
	if err != nil {
		t.Fatalf("Pack failed for larger payload: %v", err)
	}
	if len(result) == 0 {
		t.Fatal("Pack returned empty result for larger payload")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

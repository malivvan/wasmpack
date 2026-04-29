package wasmpack

import (
	"bytes"
	"strings"
	"testing"
)

const simpleJS = `
var x = 1;
// single-line comment
var y   =   2;
/* multi-line
   comment */
console.log(x + y);
`

const preservedCommentJS = `
var a = 1;
/*! preserved license comment */
var b = 2;
`

func TestMinify_ReducesSize(t *testing.T) {
	result, err := Minify([]byte(simpleJS))
	if err != nil {
		t.Fatalf("Minify failed: %v", err)
	}
	if len(result) == 0 {
		t.Fatal("Minify returned empty result")
	}
	if len(result) >= len(simpleJS) {
		t.Errorf("expected minified size (%d) < original (%d)", len(result), len(simpleJS))
	}
}

func TestMinify_RemovesSingleLineComments(t *testing.T) {
	result, err := Minify([]byte(simpleJS))
	if err != nil {
		t.Fatalf("Minify failed: %v", err)
	}
	if bytes.Contains(result, []byte("//")) {
		t.Error("Minify did not remove single-line comments")
	}
	if bytes.Contains(result, []byte("single-line comment")) {
		t.Error("Minify did not remove comment text")
	}
}

func TestMinify_RemovesBlockComments(t *testing.T) {
	result, err := Minify([]byte(simpleJS))
	if err != nil {
		t.Fatalf("Minify failed: %v", err)
	}
	if bytes.Contains(result, []byte("multi-line")) {
		t.Error("Minify did not remove block comment text")
	}
}

func TestMinify_PreservesLicenseComments(t *testing.T) {
	result, err := Minify([]byte(preservedCommentJS))
	if err != nil {
		t.Fatalf("Minify failed: %v", err)
	}
	if !bytes.Contains(result, []byte("/*!")) {
		t.Error("Minify removed a preserved /*! ... */ comment")
	}
	if !bytes.Contains(result, []byte("preserved license comment")) {
		t.Error("Minify removed preserved license comment text")
	}
}

func TestMinify_EmptyInput(t *testing.T) {
	result, err := Minify([]byte{})
	if err != nil {
		t.Fatalf("Minify with empty input failed: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty result, got %q", result)
	}
}

func TestMinify_PreservesSemantics(t *testing.T) {
	input := `var hello = "world";`
	result, err := Minify([]byte(input))
	if err != nil {
		t.Fatalf("Minify failed: %v", err)
	}
	s := string(result)
	if !strings.Contains(s, "hello") {
		t.Error("Minify removed identifier 'hello'")
	}
	if !strings.Contains(s, `"world"`) {
		t.Error("Minify removed string literal 'world'")
	}
}

func TestMinify_FunctionExpression(t *testing.T) {
	input := `
(function() {
    var   i   = 0;
    while (i < 10) {
        i++;
    }
    return i;
})();
`
	result, err := Minify([]byte(input))
	if err != nil {
		t.Fatalf("Minify failed: %v", err)
	}
	if len(result) == 0 {
		t.Fatal("Minify returned empty result for function expression")
	}
	if len(result) >= len(input) {
		t.Errorf("Minify did not reduce size: %d >= %d", len(result), len(input))
	}
}

func TestMinify_StringLiterals(t *testing.T) {
	input := `var s = "hello   world";`
	result, err := Minify([]byte(input))
	if err != nil {
		t.Fatalf("Minify failed: %v", err)
	}
	// whitespace inside string literals must be preserved
	if !strings.Contains(string(result), "hello   world") {
		t.Error("Minify corrupted whitespace inside string literal")
	}
}


# wasmpack

A powerful CLI tool and Go package for building, optimizing, and packaging WebAssembly (WASM) modules compiled from Go code into a single JavaScript file. Perfect for embedding WASM functionality directly in your web applications without needing separate file dependencies.

## Features

- **Build Go to WASM**: Compile Go source code directly to WebAssembly
- **Compression**: Automatic DEFLATE compression of WASM binaries using base64 encoding
- **Optimization**: Optional WASM optimization using `wasm-opt` tool
- **Code Obfuscation**: Optional code obfuscation using `garble`
- **JavaScript Wrapping**: Automatically wraps WASM modules with Go's `wasm_exec.js` runtime
- **Minification**: Optional JavaScript minification for production builds
- **Custom Prefixing**: Add custom JavaScript code before the generated output
- **Single File Output**: Creates a self-contained JavaScript file that includes everything needed to run your Go WASM code

## Installation

Install the CLI tool:

```bash
go install github.com/malivvan/wasmpack/cmd@latest
```

Or use the package in your Go projects:

```bash
go get github.com/malivvan/wasmpack
```

## Usage

### Command Line Interface

```bash
wasmpack [flags] <source> <output>
```

#### Arguments

- `<source>`: Path to Go source file/directory or a compiled `.wasm` file
- `<output>`: Output JavaScript file path

#### Flags

- `-name string`: Name of the global function (if not specified, the WASM code runs immediately)
- `-minify`: Minify the output JavaScript code (default: false)
- `-silent`: Silent mode - suppress informational output (default: false)
- `-pre string`: Path to a JavaScript file to prefix the output with
- `-post string`: Path to a JavaScript filprefixe to suffix the output with
- `-obfus string`: Arguments to pass to obfuscate.js (optional)
- `-garble string`: Arguments to pass to `garble` for code obfuscation (optional)
- `-opt string`: Arguments to pass to `wasm-opt` for optimization (optional)

#### Examples

**Basic compilation from Go source:**

```bash
wasmpack sample/main.go output.js
```

**With optimization:**

```bash
wasmpack -opt "-O4" sample/main.go output.js
```

**With minification and custom global function:**

```bash
wasmpack -name "init" -m sample/main.go output.js
```

**With code obfuscation:**

```bash
wasmpack -garble "-literals" sample/main.go output.js
```

**With pre JavaScript:**

```bash
wasmpack -pre pre.js sample/main.go output.js
```

**With post JavaScript:**

```bash
wasmpack -post post.js sample/main.go output.js
```

**Optimize a pre-compiled WASM file:**

```bash
wasmpack -opt "-O4" compiled.wasm output.js
```

### Go Package

Use wasmpack as a library in your Go projects:

```go
package main

import (
    "os"
    "github.com/malivvan/wasmpack"
)

func main() {
    // Build Go source to WASM
    wasm, err := wasmpack.Build("main.go", "", "-O4")
    if err != nil {
        panic(err)
    }

    // Pack the WASM binary (compress and encode)
    packed, err := wasmpack.Pack(wasm)
    if err != nil {
        panic(err)
    }

    // Wrap with Go's WASM runtime
    code, err := wasmpack.Wrap("myFunction", packed)
    if err != nil {
        panic(err)
    }

    // Optionally minify
    minified, err := wasmpack.Minify(code)
    if err != nil {
        panic(err)
    }

    // Write to file
    os.WriteFile("output.js", minified, 0644)
}
```

#### Package Functions

- **`Build(path string, garble string, wasmopt string) ([]byte, error)`**: Compiles Go code to WASM
  - `path`: Path to Go source file or directory
  - `garble`: Arguments for garble obfuscation (empty string to skip)
  - `wasmopt`: Arguments for wasm-opt optimization (empty string to skip)
- **`Pack(wasm []byte) ([]byte, error)`**: Compresses WASM binary using DEFLATE and encodes as JavaScript
- **`Wrap(name string, code []byte) ([]byte, error)`**: Wraps WASM code with Go's runtime
  - `name`: Global function name (empty string to run immediately)
- **`Minify(code []byte) ([]byte, error)`**: Minifies JavaScript code
- **`Optimize(path string, opt string) error`**: Optimizes a WASM file using wasm-opt
- **`Obfuscate(path string, obfus string) error`**: Obfuscates Go code using obfuscate.js

## How It Works

1. **Build**: Compiles Go source code to WebAssembly using the Go compiler with `GOOS=js GOARCH=wasm`
2. **Optimize** (optional): Runs `wasm-opt` to reduce WASM binary size
3. **Pack**: Compresses the WASM binary using DEFLATE compression and encodes it as base64 within JavaScript code
4. **Wrap**: Embeds the compressed WASM with Go's `wasm_exec.js` runtime into a single JavaScript file
5. **Minify** (optional): Reduces JavaScript code size for production

The result is a single JavaScript file containing:
- Go's WASM runtime (`wasm_exec.js`)
- Compressed and encoded WASM binary
- Code to decompress and instantiate the WASM module

## Requirements

- **Go 1.25.0 or later**
- **wasm-opt** (optional, for WASM optimization): [Install from Binaryen](https://github.com/WebAssembly/binaryen)
- **garble** (optional, for code obfuscation): `go install github.com/burrowers/garble@latest`

## Example Project

The `sample/` directory contains a simple example:

```html
<!doctype html>
<html lang="en">
<head>
    <title>wasmpack</title>
    <script src="main.js"></script>
    <script>init()</script>
</head>
</html>
```

Build the sample:

```bash
wasmpack -n "init" sample/main.go sample/main.js
```

Then open `sample/index.html` in a browser. The Go code will run and print "Hello, World!" to the browser console.

## Output Size Benefits

The DEFLATE compression in the Pack step typically achieves:
- **60-80% reduction** in WASM binary size before JavaScript wrapping
- Final output is a self-contained JavaScript file with no external dependencies

Example output from the CLI shows compression ratio:
```
output.js: 5.23 MB -> 1.45 MB (27.71%)
```

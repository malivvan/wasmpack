# wasmpack
A CLI tool and Go library for building, optimizing, and packaging WebAssembly modules compiled from Go into a single JavaScript file. Supports self-executing IIFEs, ES modules, CommonJS modules, standalone HTML pages, and a live-reloading development server.

## Features
- **Build Go → WASM** — compiles any Go package with `GOOS=js GOARCH=wasm`
- **TinyGo support** — compile with [tinygo](https://tinygo.org/) instead of the standard Go toolchain
- **Skip build step** — pass a pre-compiled `.wasm` file to skip compilation
- **Compression** — DEFLATE-compresses the binary before embedding (typically 60–80% smaller)
- **Four JS output formats** — IIFE (`.js`), ES module (`.mjs`), CommonJS module (`.cjs`), standalone HTML (`.html`)
- **wasm-opt integration** — optional Binaryen optimisation pass via [wasm-opt](https://github.com/WebAssembly/binaryen)
- **Garble support** — optional Go build obfuscation via [garble](https://github.com/burrowers/garble)
- **JS obfuscation** — powered by [javascript-obfuscator](https://github.com/javascript-obfuscator/javascript-obfuscator) (**no Node.js required**)
- **JS minification** — built-in minifier, no external tools needed
- **Pre / post JS injection** — prepend or append custom JS files
- **Live-reload dev server** — watches source files, rebuilds on change, auto-refreshes the browser

## Installation
```bash
go install github.com/malivvan/wasmpack/cmd@latest
```
Or use the library in your Go project:
```bash
go get github.com/malivvan/wasmpack
```

## CLI usage
### `wasmpack init [path]`
Write a default `wasmpack.yml` configuration file.

```bash
wasmpack init                  # creates ./wasmpack.yml
wasmpack init ./myproject      # creates ./myproject/wasmpack.yml
wasmpack init path/to/cfg.yml  # creates the file at the exact path
```

### `wasmpack pack [flags] <source> <destination>`
Compile `source` and write the result to `destination`.
**Source** — a Go package directory, a `main.go` file, or a pre-compiled `.wasm` file.
When source ends with `.wasm` the build step is skipped entirely.
**Destination extensions** control the output format:

| Extension | Output                                                           |
|-----------|------------------------------------------------------------------|
| `.wasm`   | Raw binary only (no JS wrapper)                                  |
| `.js`     | Self-executing IIFE — runs immediately when the script is loaded |
| `.mjs`    | ES module that exports `run(env, args)`                          |
| `.cjs`    | CommonJS module that exports `run(env, args)`                    |
| `.html`   | Minimal HTML page with the IIFE in a `<script defer>` tag        |

**Pack flags:**

| Flag         | Description                                                          |
|--------------|----------------------------------------------------------------------|
| `-garble`    | Obfuscate the Go build with garble (options from `wasmpack.yml`)     |
| `-tinygo`    | Compile with tinygo instead of go (options from `wasmpack.yml`)      |
| `-wasm-opt`  | Optimise the wasm binary with wasm-opt (options from `wasmpack.yml`) |
| `-minify`    | Minify the JS output                                                 |
| `-obfuscate` | Obfuscate the JS output (options from `wasmpack.yml`)                |

```bash
wasmpack pack ./sample output.html             # standalone HTML page
wasmpack pack ./sample output.js              # self-executing IIFE bundle
wasmpack pack ./sample output.mjs             # ES module
wasmpack pack ./sample output.cjs             # CommonJS module
wasmpack pack ./sample output.wasm            # raw binary only
wasmpack pack compiled.wasm output.js         # skip compilation, use pre-built .wasm
wasmpack pack -tinygo ./sample output.js      # compile with tinygo
wasmpack pack -wasm-opt -minify ./sample out.js  # wasm-opt + minified JS
```

Example output:
```
  build      2.41 MB  (0.89s)
  wrap       html
  write      output.html  2.41 MB → 0.89 MB  (37.0%)
  done       1.12s
```

### `wasmpack dev [flags] <source> <addr>`
Start a live-reloading development server.

**Dev flags:**

| Flag   | Description                                                                                          |
|--------|------------------------------------------------------------------------------------------------------|
| `-coi` | Set `Cross-Origin-Opener-Policy: same-origin` and `Cross-Origin-Embedder-Policy: require-corp` headers, enabling a [cross-origin isolated](https://developer.mozilla.org/en-US/docs/Web/API/crossOriginIsolated) context (required for `SharedArrayBuffer` / `Atomics`) |

```bash
wasmpack dev ./sample :8080            # all interfaces, port 8080
wasmpack dev ./sample localhost:3000   # loopback only
wasmpack dev compiled.wasm :9000       # serve pre-built wasm, watch for changes
wasmpack dev -coi ./sample :8080       # enable cross-origin isolation
```

Example output:
```
  build      2.41 MB  (0.89s)
  watch      /home/user/project/sample
  serve      http://localhost:8080
  · main.go changed
  build      2.41 MB  (0.34s)
  reload     1 client(s)
```

The server:
- Builds and serves a self-contained HTML page at `/`
- Watches all `.go` files in the source directory (or the `.wasm` file) for changes
- Rebuilds automatically on change and reloads connected browsers via SSE
- Skips wasm-opt, garble, tinygo, obfuscation, and minification for fast iteration
## Configuration (`wasmpack.yml`)
`wasmpack.yml` is discovered by walking up from the current working directory — the same mechanism Go uses for `go.mod`. Run `wasmpack init` to generate a documented template.

```yaml
# wasm-opt: options for wasm-opt (used when -wasm-opt flag is passed to pack)
# garble:    options for garble (used when -garble flag is passed to pack)
# tinygo:    options for tinygo (used when -tinygo flag is passed to pack)
# pre:       path to a JS file prepended to the output
# post:      path to a JS file appended to the output
# obfuscate: javascript-obfuscator options (used when -obfuscate flag is passed to pack)
#            see https://github.com/javascript-obfuscator/javascript-obfuscator
pre: ""
post: ""
garble:
  seed: ""       # randomness seed (-seed)
  literals: false # obfuscate string literals (-literals)
  tiny: false    # smaller output (-tiny)
  flags: ""      # extra space-separated garble flags
tinygo:
  target: "wasm" # compile target (e.g. "wasm", "wasi")
  opt: ""        # optimization level (none, 0, 1, 2, s, z)
  flags: ""      # extra space-separated tinygo flags
wasm-opt:
  level: "O2"    # optimization level (O1, O2, O3, O4, Os)
  flags: ""      # extra space-separated wasm-opt flags
obfuscate:
  compact: true
  controlFlowFlattening: false
  # ... all javascript-obfuscator options supported
  # see https://github.com/javascript-obfuscator/javascript-obfuscator
```

## Go library API

```go
import "github.com/malivvan/wasmpack"

// Compile a Go package to raw WASM bytes.
// useGarble and useTinygo are mutually exclusive; useGarble takes precedence.
wasm, err := wasmpack.Build(
    "./mypkg",
    useGarble, wasmpack.GarbleConfig{Literals: true},
    useTinygo, wasmpack.TinygoConfig{Target: "wasm"},
    useWasmOpt, wasmpack.WasmOptConfig{Level: "O2"},
)

// Compress + encode the WASM binary into a JS inflate snippet.
packed, err := wasmpack.Pack(wasm)

// Wrap in wasm_exec.js as a self-executing IIFE.
js, err := wasmpack.WrapIIFE(packed)
// Wrap as an ES module exporting run(env, args).
js, err := wasmpack.WrapESM(packed)
// Wrap as a CommonJS module exporting run(env, args).
js, err := wasmpack.WrapCJS(packed)
// Wrap inside a minimal HTML page (IIFE in <script defer>).
html, err := wasmpack.WrapHTML(packed)

// Run wasm-opt on a WASM file in-place.
err = wasmpack.WasmOpt("/path/to/file.wasm", []string{"-O2"})

// Obfuscate JS (no Node.js required).
out, err := wasmpack.Obfuscate(js, "-controlFlowFlattening")
out, err := wasmpack.ObfuscateWithOptions(js, options)
// Parse obfuscate flag string to options map.
opts, err := wasmpack.ParseObfuscateOptions("-controlFlowFlattening -compact=false")

// Minify JS.
out, err := wasmpack.Minify(js)

// Config helpers.
cfg  := wasmpack.DefaultConfig()
path, err := wasmpack.FindConfig(".")     // walk-up search
cfg,  err  = wasmpack.LoadConfig(path)
err        = wasmpack.WriteDefaultConfig(path)
```

## Output size
DEFLATE compression typically achieves a 60–80% reduction before wrapping:
```
  write    output.js  5.23 MB → 1.45 MB  (27.7%)
```

## Requirements
- **Go 1.25+**
- **wasm-opt** *(optional)*: [Binaryen releases](https://github.com/WebAssembly/binaryen/releases)
- **garble** *(optional)*: `go install mvdan.cc/garble@latest`
- **tinygo** *(optional)*: [tinygo.org/doc/getting-started/overview](https://tinygo.org/getting-started/overview/)

## Sample project
`sample/main.go` prints `Hello, World!` to the browser console.
```bash
# Live-reload dev server
wasmpack dev ./sample :8080
# Build a standalone HTML page
wasmpack pack ./sample sample/output.html
```

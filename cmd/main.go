package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/malivvan/wasmpack"
)

const usageText = `wasmpack — build Go WebAssembly bundles

Commands:
  wasmpack init [path]                   Create new wasmpack.yml.
  wasmpack pack [flags] <source> <dest>  Compile source and write to dest.
  wasmpack dev  [flags] <source> <addr>  Start a live-reloading dev server.

Source argument:
  A path to a Go package directory or main.go file.
  If source ends with .wasm the build step is skipped and the file is used directly.

Destination extensions (pack command):
  .wasm   Write the compiled binary only (no JS wrapper).
  .js     Self-executing IIFE; starts running when the script is loaded.
  .mjs    ES module that exports run(env, args).
  .cjs    CommonJS module that exports run(env, args).
  .html   Minimal HTML page with the IIFE inside a <script defer> tag.

Pack flags:
  -garble     Obfuscate the Go build with garble (options from wasmpack.yml).
  -tinygo     Compile with tinygo instead of go (options from wasmpack.yml).
  -wasm-opt   Optimise the wasm binary with wasm-opt (options from wasmpack.yml).
  -minify     Minify the JS output.
  -obfuscate  Obfuscate the JS output (options from wasmpack.yml).

Dev flags:
  -coi        Set COOP and COEP headers for cross-origin isolation
              (enables SharedArrayBuffer / Atomics in the browser).

Configuration:
  wasmpack.yml is discovered by walking up from the current working directory
  (similar to go.mod). Run "wasmpack init" to create a template.
`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usageText)
		os.Exit(1)
	}
	switch os.Args[1] {
	case "init":
		cmdInit(os.Args[2:])
	case "pack":
		cmdPack(os.Args[2:])
	case "dev":
		cmdDev(os.Args[2:])
	case "help", "-h", "--help":
		fmt.Print(usageText)
	default:
		fmt.Fprintf(os.Stderr, "wasmpack: unknown command %q\n\n", os.Args[1])
		fmt.Fprint(os.Stderr, usageText)
		os.Exit(1)
	}
}

// ─── config helper ───────────────────────────────────────────────────────────

// loadCfg discovers and loads the nearest wasmpack.yml, walking upward from
// cwd. Falls back to DefaultConfig silently when no file is found, logging a
// warning if the file exists but cannot be parsed.
func loadCfg() *wasmpack.Config {
	cwd, _ := os.Getwd()
	if path, err := wasmpack.FindConfig(cwd); err == nil && path != "" {
		if cfg, err := wasmpack.LoadConfig(path); err == nil {
			logStep("config", path)
			return cfg
		} else {
			logWarn(fmt.Sprintf("could not load %s: %v", path, err))
		}
	}
	return wasmpack.DefaultConfig()
}

// ─── init ───────────────────────────────────────────────────────────────────

func cmdInit(args []string) {
	start := time.Now()
	logBlank()

	dest := wasmpack.ConfigFileName
	if len(args) >= 1 {
		p := args[0]
		if fi, err := os.Stat(p); err == nil && fi.IsDir() {
			dest = filepath.Join(p, wasmpack.ConfigFileName)
		} else if filepath.Ext(p) == "" {
			if err := os.MkdirAll(p, 0o755); err != nil {
				fatalf("init", "creating directory %q: %v", p, err)
			}
			dest = filepath.Join(p, wasmpack.ConfigFileName)
		} else {
			dest = p
		}
	}
	dest = filepath.Clean(dest)

	if _, err := os.Stat(dest); err == nil {
		fatalf("init", "%s already exists", dest)
	}
	if err := wasmpack.WriteDefaultConfig(dest); err != nil {
		fatalf("init", "writing config: %v", err)
	}
	logStep("write", dest)
	logDone(time.Since(start))
}

// ─── pack ───────────────────────────────────────────────────────────────────

func cmdPack(args []string) {
	fs := flag.NewFlagSet("pack", flag.ExitOnError)
	fGarble := fs.Bool("garble", false, "obfuscate Go build with garble")
	fTinygo := fs.Bool("tinygo", false, "compile with tinygo")
	fOptimize := fs.Bool("wasm-opt", false, "optimise wasm with wasm-opt")
	fMinify := fs.Bool("minify", false, "minify JS output")
	fObfuscate := fs.Bool("obfuscate", false, "obfuscate JS output")
	_ = fs.Parse(args)
	rest := fs.Args()
	if len(rest) != 2 {
		fmt.Fprint(os.Stderr, usageText)
		os.Exit(1)
	}
	source := rest[0]
	dest := rest[1]
	ext := strings.ToLower(filepath.Ext(dest))

	start := time.Now()
	logBlank()
	cfg := loadCfg()

	// ── obtain raw wasm bytes ────────────────────────────────────────────────
	var wasm []byte
	if strings.HasSuffix(source, ".wasm") {
		var err error
		wasm, err = os.ReadFile(source)
		if err != nil {
			fatalf("read", "%v", err)
		}
		logStep("read", fmt.Sprintf("%s  %s", source, fmtSize(len(wasm))))
	} else {
		t := time.Now()
		var err error
		wasm, err = wasmpack.Build(source, *fGarble, cfg.Garble, *fTinygo, cfg.Tinygo, *fOptimize, cfg.WasmOpt)
		if err != nil {
			fatalf("build", "%v", err)
		}
		logStep("build", fmt.Sprintf("%s  (%s)", fmtSize(len(wasm)), fmtDur(time.Since(t))))
	}
	rawSize := len(wasm)

	// ── .wasm output ────────────────────────────────────────────────────────
	if ext == ".wasm" {
		if err := os.WriteFile(dest, wasm, 0o644); err != nil {
			fatalf("write", "%v", err)
		}
		logStep("write", dest)
		logDone(time.Since(start))
		return
	}

	// ── pack + wrap ─────────────────────────────────────────────────────────
	if ext != ".js" && ext != ".mjs" && ext != ".cjs" && ext != ".html" {
		fatalf("pack", "unsupported destination extension %q — want .wasm, .js, .mjs, .cjs, or .html", ext)
	}

	packed, err := wasmpack.Pack(wasm)
	if err != nil {
		fatalf("pack", "%v", err)
	}

	wrapFns := map[string]func([]byte) ([]byte, error){
		".js":   wasmpack.WrapIIFE,
		".mjs":  wasmpack.WrapESM,
		".cjs":  wasmpack.WrapCJS,
		".html": wasmpack.WrapHTML,
	}
	wrapLabels := map[string]string{".js": "iife", ".mjs": "esm", ".cjs": "cjs", ".html": "html"}

	code, err := wrapFns[ext](packed)
	if err != nil {
		fatalf("wrap", "%v", err)
	}
	logStep("wrap", wrapLabels[ext])

	// ── pre / post injection ─────────────────────────────────────────────────
	if cfg.Pre != "" {
		pre, err := os.ReadFile(cfg.Pre)
		if err != nil {
			fatalf("pre", "reading %q: %v", cfg.Pre, err)
		}
		code = append(pre, code...)
	}
	if cfg.Post != "" {
		post, err := os.ReadFile(cfg.Post)
		if err != nil {
			fatalf("post", "reading %q: %v", cfg.Post, err)
		}
		code = append(code, post...)
	}

	// ── obfuscate (optional) ─────────────────────────────────────────────────
	if *fObfuscate {
		t := time.Now()
		code, err = wasmpack.ObfuscateWithOptions(code, cfg.Obfuscate.ToOptions())
		if err != nil {
			fatalf("obfuscate", "%v", err)
		}
		logStep("obfuscate", fmt.Sprintf("(%s)", fmtDur(time.Since(t))))
	}

	// ── minify (optional) ────────────────────────────────────────────────────
	if *fMinify {
		before := len(code)
		code, err = wasmpack.Minify(code)
		if err != nil {
			fatalf("minify", "%v", err)
		}
		logStep("minify", fmtSizeRatio(before, len(code)))
	}

	// ── write output ─────────────────────────────────────────────────────────
	if err := os.WriteFile(dest, code, 0o644); err != nil {
		fatalf("write", "%v", err)
	}
	logStep("write", fmt.Sprintf("%s  %s", dest, fmtSizeRatio(rawSize, len(code))))
	logDone(time.Since(start))
}

package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/malivvan/wasmpack"
)

var info = func(format string, a ...interface{}) {}

func main() {
	var name string
	var silent bool
	var minify bool
	var pre string
	var post string
	var output string
	var garble string
	var opt string
	var obfus string
	flag.Usage = func() {
		_, _ = fmt.Fprintf(flag.CommandLine.Output(), "usage %s (flags) <source> <output>:\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.StringVar(&garble, "garble", "", "arguments to pass to garble (default: none)")
	flag.StringVar(&opt, "opt", "", "arguments to pass to wasm-opt (default: none)")
	flag.BoolVar(&minify, "minify", false, "minify the output js code (default: false)")
	flag.BoolVar(&silent, "silent", false, "silent mode (default: false)")
	flag.StringVar(&pre, "pre", "", "pre js code (default: none)")
	flag.StringVar(&post, "post", "", "post js code (default: none)")
	flag.StringVar(&obfus, "obfus", "", "arguments to pass to javascript obfuscator (default: none)")
	flag.StringVar(&name, "name", "", "name of the global function (default: run immediately)")
	flag.Parse()
	if flag.NArg() != 2 {
		flag.Usage()
		os.Exit(1)
	}
	path := flag.Arg(0)
	output = flag.Arg(1)
	if !silent {
		info = func(format string, a ...interface{}) {
			_, _ = fmt.Fprintf(os.Stderr, format+"\n", a...)
		}
	}

	var wasm []byte
	var err error
	if strings.HasSuffix(path, ".wasm") {
		if opt != "" {
			err = wasmpack.Opt(path, opt)
			if err != nil {
				println("wasm-opt error: " + err.Error())
				os.Exit(1)
			}
		}
		wasm, err = os.ReadFile(path)
		if err != nil {
			println("read error: " + err.Error())
			os.Exit(1)
		}
	} else {
		wasm, err = wasmpack.Build(path, garble, opt)
		if err != nil {
			println("build error: " + err.Error())
			os.Exit(1)
		}
	}
	code, err := wasmpack.Pack(wasm)
	if err != nil {
		println("pack error: " + err.Error())
		os.Exit(1)
	}
	code, err = wasmpack.Wrap(name, code)
	if err != nil {
		println("wrap error: " + err.Error())
		os.Exit(1)
	}
	if pre != "" {
		js, err := os.ReadFile(pre)
		if err != nil {
			println("pre read error: " + err.Error())
			os.Exit(1)
		}
		code = append(js, code...)
	}
	if post != "" {
		js, err := os.ReadFile(post)
		if err != nil {
			println("post read error: " + err.Error())
			os.Exit(1)
		}
		code = append(code, js...)
	}
	if obfus != "" {
		code, err = wasmpack.Obfus(code, obfus)
		if err != nil {
			println("obfuscate error: " + err.Error())
			os.Exit(1)
		}
	}
	if minify {
		code, err = wasmpack.Minify(code)
		if err != nil {
			println("minify error: " + err.Error())
			os.Exit(1)
		}
	}

	info("%s: %.2f MB -> %.2f MB (%.2f%%)", strings.TrimPrefix(output, "./"), float64(len(wasm))/1024/1024, float64(len(code))/1024/1024, float64(len(code))/float64(len(wasm))*100)
	var w io.Writer
	if output != "" {
		f, err := os.Create(output)
		if err != nil {
			println("create error: " + err.Error())
			os.Exit(1)
		}
		defer func() { _ = f.Close() }()
		w = f
	} else {
		w = os.Stdout
	}
	_, err = w.Write(code)
	if err != nil {
		println("write error: " + err.Error())
		os.Exit(1)
	}
}

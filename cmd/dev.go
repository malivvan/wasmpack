package main

import (
	"bytes"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/malivvan/wasmpack"
)

// devReloadScript is injected at the end of the <head> so the browser
// reconnects via SSE and reloads when the server signals a rebuild.
const devReloadScript = `<script>
(function(){
  var es = new EventSource('/_reload');
  es.addEventListener('reload', function(){ window.location.reload(); });
  es.onerror = function(){ setTimeout(function(){ location.reload(); }, 1000); };
})();
</script>`

// devServer holds the mutable state shared between the HTTP handler, the file
// watcher, and SSE broadcast goroutine.
type devServer struct {
	mu      sync.Mutex
	html    []byte
	coi     bool // serve Cross-Origin Isolation headers (COOP + COEP)
	clients map[chan struct{}]struct{}
}

func cmdDev(args []string) {
	fs := flag.NewFlagSet("dev", flag.ExitOnError)
	fCOI := fs.Bool("coi", false, "set COOP and COEP headers for cross-origin isolation (required by SharedArrayBuffer / Atomics)")
	_ = fs.Parse(args)
	rest := fs.Args()
	if len(rest) != 2 {
		fmt.Fprintf(os.Stderr, "usage: wasmpack dev [flags] <source> <addr>\n")
		fmt.Fprintf(os.Stderr, "  addr examples: :8080  localhost:3000\n")
		fmt.Fprintf(os.Stderr, "flags:\n")
		fs.PrintDefaults()
		os.Exit(1)
	}
	source := rest[0]
	addr := rest[1]
	// Allow bare port numbers like "8080".
	if !strings.Contains(addr, ":") {
		addr = ":" + addr
	}

	logBlank()
	cfg := loadCfg()

	s := &devServer{clients: make(map[chan struct{}]struct{}), coi: *fCOI}

	// Initial build.
	if err := s.rebuild(source, cfg); err != nil {
		fatalf("build", "%v", err)
	}

	// Start file watcher in the background.
	go s.watch(source, cfg)

	mux := http.NewServeMux()
	mux.HandleFunc("/", s.serveHTML)
	mux.HandleFunc("/_reload", s.serveSSE)

	display := addr
	if strings.HasPrefix(display, ":") {
		display = "localhost" + display
	}
	if *fCOI {
		logStep("coi", "cross-origin isolation enabled")
	}
	logStep("serve", "http://"+display)

	if err := http.ListenAndServe(addr, mux); err != nil {
		fatalf("serve", "%v", err)
	}
}

// rebuild compiles source (or reads the .wasm file), packs, and wraps the
// result into the in-memory HTML page. wasm-opt, obfuscate, and minify are
// intentionally skipped in dev mode for fast iteration.
func (s *devServer) rebuild(source string, cfg *wasmpack.Config) error {
	t := time.Now()
	var wasm []byte
	var err error

	if strings.HasSuffix(source, ".wasm") {
		wasm, err = os.ReadFile(source)
	} else {
		// Skip wasm-opt, garble, and tinygo in dev mode to keep rebuilds fast.
		wasm, err = wasmpack.Build(source, false, wasmpack.GarbleConfig{}, false, wasmpack.TinygoConfig{}, false, wasmpack.WasmOptConfig{})
	}
	if err != nil {
		return fmt.Errorf("build: %w", err)
	}

	packed, err := wasmpack.Pack(wasm)
	if err != nil {
		return fmt.Errorf("pack: %w", err)
	}

	js, err := wasmpack.WrapIIFE(packed)
	if err != nil {
		return fmt.Errorf("wrap: %w", err)
	}

	var buf bytes.Buffer
	fmt.Fprintf(&buf,
		"<!DOCTYPE html>\n<html>\n<head>\n%s\n<script defer>\n%s\n</script>\n</head>\n<body></body>\n</html>\n",
		devReloadScript, js)

	s.mu.Lock()
	s.html = buf.Bytes()
	s.mu.Unlock()

	logStep("build", fmt.Sprintf("%s  (%s)", fmtSize(len(wasm)), fmtDur(time.Since(t))))
	return nil
}

// watch monitors source files for changes and triggers a rebuild + SSE
// broadcast when they are modified.
func (s *devServer) watch(source string, cfg *wasmpack.Config) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		logWarn(fmt.Sprintf("could not create file watcher: %v", err))
		return
	}
	defer watcher.Close()

	// Resolve the directory to watch.
	watchPath := source
	if !strings.HasSuffix(source, ".wasm") {
		if fi, err := os.Stat(source); err == nil && !fi.IsDir() {
			watchPath = filepath.Dir(source)
		}
	}
	watchPath, _ = filepath.Abs(watchPath)

	if err := watcher.Add(watchPath); err != nil {
		logWarn(fmt.Sprintf("could not watch %s: %v", watchPath, err))
		return
	}
	logStep("watch", watchPath)

	var debounce *time.Timer
	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			name := event.Name
			isGo := strings.HasSuffix(name, ".go")
			isWasm := strings.HasSuffix(name, ".wasm")
			if (!isGo && !isWasm) || event.Op&(fsnotify.Write|fsnotify.Create) == 0 {
				continue
			}
			if debounce != nil {
				debounce.Stop()
			}
			debounce = time.AfterFunc(200*time.Millisecond, func() {
				logChange(filepath.Base(name) + " changed")
				if err := s.rebuild(source, cfg); err != nil {
					logWarn(fmt.Sprintf("rebuild failed: %v", err))
					return
				}
				n := s.broadcast()
				if n > 0 {
					logStep("reload", fmt.Sprintf("%d client(s)", n))
				}
			})

		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			logWarn(fmt.Sprintf("watcher: %v", err))
		}
	}
}

// broadcast sends a reload signal to all connected SSE clients and returns the
// number of clients notified.
func (s *devServer) broadcast() int {
	s.mu.Lock()
	snapshot := make([]chan struct{}, 0, len(s.clients))
	for ch := range s.clients {
		snapshot = append(snapshot, ch)
	}
	s.mu.Unlock()

	for _, ch := range snapshot {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
	return len(snapshot)
}

// serveHTML serves the current in-memory HTML page.
func (s *devServer) serveHTML(w http.ResponseWriter, r *http.Request) {
	if s.coi {
		w.Header().Set("Cross-Origin-Opener-Policy", "same-origin")
		w.Header().Set("Cross-Origin-Embedder-Policy", "require-corp")
	}
	s.mu.Lock()
	html := s.html
	s.mu.Unlock()
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(html)
}

// serveSSE handles the Server-Sent Events endpoint used for live reload.
func (s *devServer) serveSSE(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, canFlush := w.(http.Flusher)

	ch := make(chan struct{}, 1)
	s.mu.Lock()
	s.clients[ch] = struct{}{}
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		delete(s.clients, ch)
		s.mu.Unlock()
	}()

	// Initial ping so the browser knows the stream is live.
	fmt.Fprintf(w, "event: connected\ndata: {}\n\n")
	if canFlush {
		flusher.Flush()
	}

	for {
		select {
		case _, valid := <-ch:
			if !valid {
				return
			}
			fmt.Fprintf(w, "event: reload\ndata: {}\n\n")
			if canFlush {
				flusher.Flush()
			}
		case <-r.Context().Done():
			return
		}
	}
}

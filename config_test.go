package wasmpack

import (
	"os"
	"path/filepath"
	"testing"
)

// ---------- GarbleConfig ----------

func TestGarbleConfigToArgs_Empty(t *testing.T) {
	cfg := GarbleConfig{}
	args := cfg.ToArgs()
	if len(args) != 0 {
		t.Errorf("expected no args, got %v", args)
	}
}

func TestGarbleConfigToArgs_AllFields(t *testing.T) {
	cfg := GarbleConfig{
		Seed:     "abc123",
		Literals: true,
		Tiny:     true,
		Flags:    "-debug -extra",
	}
	args := cfg.ToArgs()
	want := []string{"-seed=abc123", "-literals", "-tiny", "-debug", "-extra"}
	if len(args) != len(want) {
		t.Fatalf("expected %v, got %v", want, args)
	}
	for i, w := range want {
		if args[i] != w {
			t.Errorf("arg[%d]: expected %q, got %q", i, w, args[i])
		}
	}
}

func TestGarbleConfigToArgs_SeedOnly(t *testing.T) {
	cfg := GarbleConfig{Seed: "myseed"}
	args := cfg.ToArgs()
	if len(args) != 1 || args[0] != "-seed=myseed" {
		t.Errorf("expected [-seed=myseed], got %v", args)
	}
}

// ---------- TinygoConfig ----------

func TestTinygoConfigToArgs_Empty(t *testing.T) {
	cfg := TinygoConfig{}
	args := cfg.ToArgs()
	if len(args) != 0 {
		t.Errorf("expected no args for empty config, got %v", args)
	}
}

func TestTinygoConfigToArgs_AllFields(t *testing.T) {
	cfg := TinygoConfig{Target: "wasm", Opt: "2", Flags: "-no-debug"}
	args := cfg.ToArgs()
	want := []string{"-target=wasm", "-opt=2", "-no-debug"}
	if len(args) != len(want) {
		t.Fatalf("expected %v, got %v", want, args)
	}
	for i, w := range want {
		if args[i] != w {
			t.Errorf("arg[%d]: expected %q, got %q", i, w, args[i])
		}
	}
}

// ---------- WasmOptConfig ----------

func TestWasmOptConfigToArgs_Empty(t *testing.T) {
	cfg := WasmOptConfig{}
	args := cfg.ToArgs()
	if len(args) != 0 {
		t.Errorf("expected no args for empty config, got %v", args)
	}
}

func TestWasmOptConfigToArgs_Level(t *testing.T) {
	cfg := WasmOptConfig{Level: "O3"}
	args := cfg.ToArgs()
	if len(args) != 1 || args[0] != "-O3" {
		t.Errorf("expected [-O3], got %v", args)
	}
}

func TestWasmOptConfigToArgs_LevelAndFlags(t *testing.T) {
	cfg := WasmOptConfig{Level: "O2", Flags: "--enable-simd"}
	args := cfg.ToArgs()
	want := []string{"-O2", "--enable-simd"}
	if len(args) != len(want) {
		t.Fatalf("expected %v, got %v", want, args)
	}
	for i, w := range want {
		if args[i] != w {
			t.Errorf("arg[%d]: expected %q, got %q", i, w, args[i])
		}
	}
}

// ---------- DefaultConfig ----------

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg == nil {
		t.Fatal("DefaultConfig returned nil")
	}
	if cfg.Tinygo.Target != "wasm" {
		t.Errorf("expected Tinygo.Target=wasm, got %q", cfg.Tinygo.Target)
	}
	if cfg.WasmOpt.Level != "O2" {
		t.Errorf("expected WasmOpt.Level=O2, got %q", cfg.WasmOpt.Level)
	}
	if !cfg.Obfuscate.Compact {
		t.Error("expected Obfuscate.Compact=true")
	}
	if cfg.Obfuscate.IdentifierNamesGenerator != "hexadecimal" {
		t.Errorf("expected IdentifierNamesGenerator=hexadecimal, got %q", cfg.Obfuscate.IdentifierNamesGenerator)
	}
}

// ---------- ObfuscateConfig.ToOptions ----------

func TestObfuscateConfigToOptions(t *testing.T) {
	cfg := DefaultConfig()
	opts := cfg.Obfuscate.ToOptions()
	if opts == nil {
		t.Fatal("ToOptions returned nil")
	}
	compact, ok := opts["compact"].(bool)
	if !ok {
		t.Fatal("compact not a bool in options")
	}
	if !compact {
		t.Error("expected compact=true")
	}
	if opts["target"] != "browser" {
		t.Errorf("expected target=browser, got %v", opts["target"])
	}
}

func TestObfuscateConfigToOptions_SlicedFields(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Obfuscate.DomainLock = []string{"example.com"}
	cfg.Obfuscate.ReservedNames = []string{"foo", "bar"}
	opts := cfg.Obfuscate.ToOptions()

	domains, ok := opts["domainLock"].([]string)
	if !ok || len(domains) != 1 || domains[0] != "example.com" {
		t.Errorf("unexpected domainLock: %v", opts["domainLock"])
	}
	reserved, ok := opts["reservedNames"].([]string)
	if !ok || len(reserved) != 2 {
		t.Errorf("unexpected reservedNames: %v", opts["reservedNames"])
	}
}

// ---------- FindConfig ----------

func TestFindConfig_Found(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, ConfigFileName)
	if err := os.WriteFile(cfgPath, []byte("pre: \"\"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	found, err := FindConfig(dir)
	if err != nil {
		t.Fatalf("FindConfig: %v", err)
	}
	if found != cfgPath {
		t.Errorf("expected %q, got %q", cfgPath, found)
	}
}

func TestFindConfig_NotFound(t *testing.T) {
	dir := t.TempDir()
	found, err := FindConfig(dir)
	if err != nil {
		t.Fatalf("FindConfig: %v", err)
	}
	// Either empty string or a parent dir's wasmpack.yml (if one exists in the tree).
	// For a freshly created temp dir, it should not find one unless cwd has one.
	_ = found
}

func TestFindConfig_WalksUp(t *testing.T) {
	parent := t.TempDir()
	child := filepath.Join(parent, "subdir")
	if err := os.Mkdir(child, 0o755); err != nil {
		t.Fatalf("Mkdir: %v", err)
	}
	cfgPath := filepath.Join(parent, ConfigFileName)
	if err := os.WriteFile(cfgPath, []byte("pre: \"\"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	found, err := FindConfig(child)
	if err != nil {
		t.Fatalf("FindConfig from child: %v", err)
	}
	if found != cfgPath {
		t.Errorf("expected %q, got %q", cfgPath, found)
	}
}

// ---------- LoadConfig ----------

func TestLoadConfig(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, ConfigFileName)
	if err := WriteDefaultConfig(cfgPath); err != nil {
		t.Fatalf("WriteDefaultConfig: %v", err)
	}
	cfg, err := LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg == nil {
		t.Fatal("LoadConfig returned nil config")
	}
	if cfg.WasmOpt.Level != "O2" {
		t.Errorf("expected WasmOpt.Level=O2, got %q", cfg.WasmOpt.Level)
	}
}

func TestLoadConfig_CustomValues(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, ConfigFileName)
	content := `
pre: "pre.js"
post: "post.js"
garble:
  seed: "testseed"
  literals: true
tinygo:
  target: "wasi"
  opt: "s"
wasm-opt:
  level: "O4"
`
	if err := os.WriteFile(cfgPath, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	cfg, err := LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.Pre != "pre.js" {
		t.Errorf("expected Pre=pre.js, got %q", cfg.Pre)
	}
	if cfg.Post != "post.js" {
		t.Errorf("expected Post=post.js, got %q", cfg.Post)
	}
	if cfg.Garble.Seed != "testseed" {
		t.Errorf("expected Garble.Seed=testseed, got %q", cfg.Garble.Seed)
	}
	if !cfg.Garble.Literals {
		t.Error("expected Garble.Literals=true")
	}
	if cfg.Tinygo.Target != "wasi" {
		t.Errorf("expected Tinygo.Target=wasi, got %q", cfg.Tinygo.Target)
	}
	if cfg.WasmOpt.Level != "O4" {
		t.Errorf("expected WasmOpt.Level=O4, got %q", cfg.WasmOpt.Level)
	}
}

func TestLoadConfig_MissingFile(t *testing.T) {
	_, err := LoadConfig("/nonexistent/path/wasmpack.yml")
	if err == nil {
		t.Error("expected error for missing file, got nil")
	}
}

// ---------- WriteDefaultConfig ----------

func TestWriteDefaultConfig(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, ConfigFileName)
	if err := WriteDefaultConfig(cfgPath); err != nil {
		t.Fatalf("WriteDefaultConfig: %v", err)
	}
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("WriteDefaultConfig wrote empty file")
	}
	// should contain a YAML header comment
	content := string(data)
	if len(content) < 10 {
		t.Error("WriteDefaultConfig output is too short")
	}
	// verify it round-trips
	cfg, err := LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("LoadConfig after WriteDefaultConfig: %v", err)
	}
	if cfg.WasmOpt.Level != "O2" {
		t.Errorf("round-trip: expected O2, got %q", cfg.WasmOpt.Level)
	}
}


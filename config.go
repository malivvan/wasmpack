package wasmpack

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// ConfigFileName is the name of the wasmpack configuration file.
const ConfigFileName = "wasmpack.yml"

// Config holds all wasmpack project configuration.
type Config struct {
	Pre       string          `yaml:"pre"`
	Post      string          `yaml:"post"`
	Garble    GarbleConfig    `yaml:"garble"`
	Tinygo    TinygoConfig    `yaml:"tinygo"`
	WasmOpt   WasmOptConfig   `yaml:"wasm-opt"`
	Obfuscate ObfuscateConfig `yaml:"obfuscate"`
}

// GarbleConfig holds options passed to the garble build tool.
type GarbleConfig struct {
	Seed     string `yaml:"seed"`     // randomness seed (-seed)
	Literals bool   `yaml:"literals"` // obfuscate string literals (-literals)
	Tiny     bool   `yaml:"tiny"`     // smaller output (-tiny)
	Flags    string `yaml:"flags"`    // extra space-separated garble flags
}

// ToArgs converts the GarbleConfig into a slice of garble flag strings.
func (g GarbleConfig) ToArgs() []string {
	var args []string
	if g.Seed != "" {
		args = append(args, "-seed="+g.Seed)
	}
	if g.Literals {
		args = append(args, "-literals")
	}
	if g.Tiny {
		args = append(args, "-tiny")
	}
	args = append(args, splitArgs(g.Flags)...)
	return args
}

// TinygoConfig holds options passed to the tinygo compiler.
type TinygoConfig struct {
	Target string `yaml:"target"` // compile target (e.g. "wasm", "wasi")
	Opt    string `yaml:"opt"`    // optimization level (none, 0, 1, 2, s, z)
	Flags  string `yaml:"flags"`  // extra space-separated tinygo flags
}

// ToArgs converts the TinygoConfig into a slice of tinygo flag strings.
func (t TinygoConfig) ToArgs() []string {
	var args []string
	if t.Target != "" {
		args = append(args, "-target="+t.Target)
	}
	if t.Opt != "" {
		args = append(args, "-opt="+t.Opt)
	}
	args = append(args, splitArgs(t.Flags)...)
	return args
}

// WasmOptConfig holds options passed to the wasm-opt tool.
type WasmOptConfig struct {
	Level string `yaml:"level"` // optimization level (O1, O2, O3, O4, Os)
	Flags string `yaml:"flags"` // extra space-separated wasm-opt flags
}

// ToArgs converts the WasmOptConfig into a slice of wasm-opt flag strings.
func (o WasmOptConfig) ToArgs() []string {
	var args []string
	if o.Level != "" {
		args = append(args, "-"+o.Level)
	}
	args = append(args, splitArgs(o.Flags)...)
	return args
}

// ObfuscateConfig holds all javascript-obfuscator options.
type ObfuscateConfig struct {
	Compact                               bool              `yaml:"compact"`
	ControlFlowFlattening                 bool              `yaml:"controlFlowFlattening"`
	ControlFlowFlatteningThreshold        float64           `yaml:"controlFlowFlatteningThreshold"`
	DeadCodeInjection                     bool              `yaml:"deadCodeInjection"`
	DeadCodeInjectionThreshold            float64           `yaml:"deadCodeInjectionThreshold"`
	DebugProtection                       bool              `yaml:"debugProtection"`
	DebugProtectionInterval               int               `yaml:"debugProtectionInterval"`
	DisableConsoleOutput                  bool              `yaml:"disableConsoleOutput"`
	DomainLock                            []string          `yaml:"domainLock"`
	DomainLockRedirectURL                 string            `yaml:"domainLockRedirectURL"`
	Exclude                               []string          `yaml:"exclude"`
	ForceTransformStrings                 []string          `yaml:"forceTransformStrings"`
	IdentifierNamesCache                  map[string]string `yaml:"identifierNamesCache"`
	IdentifierNamesGenerator              string            `yaml:"identifierNamesGenerator"`
	IdentifiersDictionary                 []string          `yaml:"identifiersDictionary"`
	IdentifiersPrefix                     string            `yaml:"identifiersPrefix"`
	IgnoreImports                         bool              `yaml:"ignoreImports"`
	InputFileName                         string            `yaml:"inputFileName"`
	Log                                   bool              `yaml:"log"`
	NumbersToExpressions                  bool              `yaml:"numbersToExpressions"`
	OptionsPreset                         string            `yaml:"optionsPreset"`
	RenameGlobals                         bool              `yaml:"renameGlobals"`
	RenameProperties                      bool              `yaml:"renameProperties"`
	RenamePropertiesMode                  string            `yaml:"renamePropertiesMode"`
	ReservedNames                         []string          `yaml:"reservedNames"`
	ReservedStrings                       []string          `yaml:"reservedStrings"`
	Seed                                  string            `yaml:"seed"`
	SelfDefending                         bool              `yaml:"selfDefending"`
	Simplify                              bool              `yaml:"simplify"`
	SourceMap                             bool              `yaml:"sourceMap"`
	SourceMapBaseURL                      string            `yaml:"sourceMapBaseURL"`
	SourceMapFileName                     string            `yaml:"sourceMapFileName"`
	SourceMapMode                         string            `yaml:"sourceMapMode"`
	SplitStrings                          bool              `yaml:"splitStrings"`
	SplitStringsChunkLength               int               `yaml:"splitStringsChunkLength"`
	StringArray                           bool              `yaml:"stringArray"`
	StringArrayCallsTransform             bool              `yaml:"stringArrayCallsTransform"`
	StringArrayCallsTransformThreshold    float64           `yaml:"stringArrayCallsTransformThreshold"`
	StringArrayEncoding                   []string          `yaml:"stringArrayEncoding"`
	StringArrayIndexesType                []string          `yaml:"stringArrayIndexesType"`
	StringArrayIndexShift                 bool              `yaml:"stringArrayIndexShift"`
	StringArrayRotate                     bool              `yaml:"stringArrayRotate"`
	StringArrayShuffle                    bool              `yaml:"stringArrayShuffle"`
	StringArrayWrappersCount              int               `yaml:"stringArrayWrappersCount"`
	StringArrayWrappersChainedCalls       bool              `yaml:"stringArrayWrappersChainedCalls"`
	StringArrayWrappersParametersMaxCount int               `yaml:"stringArrayWrappersParametersMaxCount"`
	StringArrayWrappersType               string            `yaml:"stringArrayWrappersType"`
	StringArrayThreshold                  float64           `yaml:"stringArrayThreshold"`
	Target                                string            `yaml:"target"`
	TransformObjectKeys                   bool              `yaml:"transformObjectKeys"`
	UnicodeEscapeSequence                 bool              `yaml:"unicodeEscapeSequence"`
}

// ToOptions converts the ObfuscateConfig into the options map consumed by the
// javascript-obfuscator engine.
func (c *ObfuscateConfig) ToOptions() map[string]any {
	options := map[string]any{
		"compact":                               c.Compact,
		"controlFlowFlattening":                 c.ControlFlowFlattening,
		"controlFlowFlatteningThreshold":        c.ControlFlowFlatteningThreshold,
		"deadCodeInjection":                     c.DeadCodeInjection,
		"deadCodeInjectionThreshold":            c.DeadCodeInjectionThreshold,
		"debugProtection":                       c.DebugProtection,
		"debugProtectionInterval":               c.DebugProtectionInterval,
		"disableConsoleOutput":                  c.DisableConsoleOutput,
		"domainLockRedirectURL":                 c.DomainLockRedirectURL,
		"identifierNamesGenerator":              c.IdentifierNamesGenerator,
		"identifiersPrefix":                     c.IdentifiersPrefix,
		"ignoreImports":                         c.IgnoreImports,
		"inputFileName":                         c.InputFileName,
		"log":                                   c.Log,
		"numbersToExpressions":                  c.NumbersToExpressions,
		"optionsPreset":                         c.OptionsPreset,
		"renameGlobals":                         c.RenameGlobals,
		"renameProperties":                      c.RenameProperties,
		"renamePropertiesMode":                  c.RenamePropertiesMode,
		"seed":                                  c.Seed,
		"selfDefending":                         c.SelfDefending,
		"simplify":                              c.Simplify,
		"sourceMap":                             c.SourceMap,
		"sourceMapBaseURL":                      c.SourceMapBaseURL,
		"sourceMapFileName":                     c.SourceMapFileName,
		"sourceMapMode":                         c.SourceMapMode,
		"splitStrings":                          c.SplitStrings,
		"splitStringsChunkLength":               c.SplitStringsChunkLength,
		"stringArray":                           c.StringArray,
		"stringArrayCallsTransform":             c.StringArrayCallsTransform,
		"stringArrayCallsTransformThreshold":    c.StringArrayCallsTransformThreshold,
		"stringArrayIndexShift":                 c.StringArrayIndexShift,
		"stringArrayRotate":                     c.StringArrayRotate,
		"stringArrayShuffle":                    c.StringArrayShuffle,
		"stringArrayWrappersCount":              c.StringArrayWrappersCount,
		"stringArrayWrappersChainedCalls":       c.StringArrayWrappersChainedCalls,
		"stringArrayWrappersParametersMaxCount": c.StringArrayWrappersParametersMaxCount,
		"stringArrayWrappersType":               c.StringArrayWrappersType,
		"stringArrayThreshold":                  c.StringArrayThreshold,
		"target":                                c.Target,
		"transformObjectKeys":                   c.TransformObjectKeys,
		"unicodeEscapeSequence":                 c.UnicodeEscapeSequence,
	}
	if len(c.DomainLock) > 0 {
		options["domainLock"] = c.DomainLock
	}
	if len(c.Exclude) > 0 {
		options["exclude"] = c.Exclude
	}
	if len(c.ForceTransformStrings) > 0 {
		options["forceTransformStrings"] = c.ForceTransformStrings
	}
	if len(c.IdentifierNamesCache) > 0 {
		options["identifierNamesCache"] = c.IdentifierNamesCache
	}
	if len(c.IdentifiersDictionary) > 0 {
		options["identifiersDictionary"] = c.IdentifiersDictionary
	}
	if len(c.ReservedNames) > 0 {
		options["reservedNames"] = c.ReservedNames
	}
	if len(c.ReservedStrings) > 0 {
		options["reservedStrings"] = c.ReservedStrings
	}
	if len(c.StringArrayEncoding) > 0 {
		options["stringArrayEncoding"] = c.StringArrayEncoding
	}
	if len(c.StringArrayIndexesType) > 0 {
		options["stringArrayIndexesType"] = c.StringArrayIndexesType
	}
	return options
}

// DefaultConfig returns a Config populated with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		Garble:   GarbleConfig{},
		Tinygo:   TinygoConfig{Target: "wasm"},
		WasmOpt:  WasmOptConfig{Level: "O2"},
		Pre:      "",
		Post:     "",
		Obfuscate: ObfuscateConfig{
			Compact:                               true,
			ControlFlowFlattening:                 false,
			ControlFlowFlatteningThreshold:        0.75,
			DeadCodeInjection:                     false,
			DeadCodeInjectionThreshold:            0.4,
			DebugProtection:                       false,
			DebugProtectionInterval:               0,
			DisableConsoleOutput:                  false,
			DomainLock:                            []string{},
			DomainLockRedirectURL:                 "about:blank",
			Exclude:                               []string{},
			ForceTransformStrings:                 []string{},
			IdentifierNamesCache:                  map[string]string{},
			IdentifierNamesGenerator:              "hexadecimal",
			IdentifiersDictionary:                 []string{},
			IdentifiersPrefix:                     "",
			IgnoreImports:                         false,
			InputFileName:                         "",
			Log:                                   false,
			NumbersToExpressions:                  false,
			OptionsPreset:                         "default",
			RenameGlobals:                         false,
			RenameProperties:                      false,
			RenamePropertiesMode:                  "safe",
			ReservedNames:                         []string{},
			ReservedStrings:                       []string{},
			Seed:                                  "",
			SelfDefending:                         false,
			Simplify:                              true,
			SourceMap:                             false,
			SourceMapBaseURL:                      "",
			SourceMapFileName:                     "",
			SourceMapMode:                         "separate",
			SplitStrings:                          false,
			SplitStringsChunkLength:               0,
			StringArray:                           true,
			StringArrayCallsTransform:             false,
			StringArrayCallsTransformThreshold:    0.5,
			StringArrayEncoding:                   []string{},
			StringArrayIndexesType:                []string{"hexadecimal-number"},
			StringArrayIndexShift:                 true,
			StringArrayRotate:                     true,
			StringArrayShuffle:                    true,
			StringArrayWrappersCount:              1,
			StringArrayWrappersChainedCalls:       true,
			StringArrayWrappersParametersMaxCount: 2,
			StringArrayWrappersType:               "variable",
			StringArrayThreshold:                  0.8,
			Target:                                "browser",
			TransformObjectKeys:                   false,
			UnicodeEscapeSequence:                 false,
		},
	}
}

// FindConfig searches for a wasmpack.yml file starting from startDir and
// walking up to the filesystem root, like go.mod discovery.
// Returns an empty string if no config file is found.
func FindConfig(startDir string) (string, error) {
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return "", err
	}
	for {
		candidate := filepath.Join(dir, ConfigFileName)
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", nil
}

// LoadConfig reads and parses a wasmpack.yml file.
// Fields not present in the file keep their DefaultConfig values.
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	cfg := DefaultConfig()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

// WriteDefaultConfig writes a commented default wasmpack.yml to path.
func WriteDefaultConfig(path string) error {
	cfg := DefaultConfig()
	body, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	header := "# wasmpack.yml\n" +
		"# Run 'wasmpack init' to regenerate this file.\n" +
		"#\n" +
		"# garble:    options for garble (used when -garble flag is passed to pack)\n" +
		"# tinygo:    options for tinygo (used when -tinygo flag is passed to pack)\n" +
		"# wasm-opt: options for wasm-opt (used when -wasm-opt flag is passed to pack)\n" +
		"# pre:       path to a JS file prepended to the output\n" +
		"# post:      path to a JS file appended to the output\n" +
		"# obfuscate: javascript-obfuscator options (used when -obfuscate flag is passed to pack)\n" +
		"#            see https://github.com/javascript-obfuscator/javascript-obfuscator\n" +
		"\n"
	return os.WriteFile(path, append([]byte(header), body...), 0o644)
}

package wasmpack

import "testing"

const testCode = ` (function(){
            var variable1 = '5' - 3;
            var variable2 = '5' + 3;
            var variable3 = '5' + - '2';
            var variable4 = ['10','10','10','10','10'].map(parseInt);
            var variable5 = 'foo ' + 1 + 1;
            console.log(variable1);
            console.log(variable2);
            console.log(variable3);
            console.log(variable4);
            console.log(variable5);
        })();`

func TestObfuscate(t *testing.T) {
	code, err := Obfuscate([]byte(testCode), "-compact -controlFlowFlattening -controlFlowFlatteningThreshold 1")
	if err != nil {
		t.Fatalf("obfuscation failed: %s", err)
	}
	t.Log(string(code))
}

// ---------- ParseObfuscateOptions ----------

func TestParseObfuscateOptions_Empty(t *testing.T) {
	opts, err := ParseObfuscateOptions("")
	if err != nil {
		t.Fatalf("ParseObfuscateOptions with empty string failed: %v", err)
	}
	// No non-default flags → map should be empty (defaults are not emitted)
	if len(opts) != 0 {
		t.Errorf("expected empty options map, got %v", opts)
	}
}

func TestParseObfuscateOptions_Compact_False(t *testing.T) {
	opts, err := ParseObfuscateOptions("-compact=false")
	if err != nil {
		t.Fatalf("ParseObfuscateOptions failed: %v", err)
	}
	v, ok := opts["compact"].(bool)
	if !ok {
		t.Fatalf("compact missing or wrong type: %v", opts["compact"])
	}
	if v {
		t.Error("expected compact=false")
	}
}

func TestParseObfuscateOptions_ControlFlowFlattening(t *testing.T) {
	opts, err := ParseObfuscateOptions("-controlFlowFlattening")
	if err != nil {
		t.Fatalf("ParseObfuscateOptions failed: %v", err)
	}
	v, ok := opts["controlFlowFlattening"].(bool)
	if !ok {
		t.Fatalf("controlFlowFlattening missing or wrong type: %v", opts["controlFlowFlattening"])
	}
	if !v {
		t.Error("expected controlFlowFlattening=true")
	}
}

func TestParseObfuscateOptions_Threshold(t *testing.T) {
	opts, err := ParseObfuscateOptions("-controlFlowFlatteningThreshold 0.5")
	if err != nil {
		t.Fatalf("ParseObfuscateOptions failed: %v", err)
	}
	v, ok := opts["controlFlowFlatteningThreshold"].(float64)
	if !ok {
		t.Fatalf("threshold missing or wrong type: %v", opts["controlFlowFlatteningThreshold"])
	}
	if v != 0.5 {
		t.Errorf("expected threshold=0.5, got %v", v)
	}
}

func TestParseObfuscateOptions_DomainLock(t *testing.T) {
	opts, err := ParseObfuscateOptions("-domainLock example.com,test.org")
	if err != nil {
		t.Fatalf("ParseObfuscateOptions failed: %v", err)
	}
	domains, ok := opts["domainLock"].([]string)
	if !ok {
		t.Fatalf("domainLock missing or wrong type: %v", opts["domainLock"])
	}
	if len(domains) != 2 {
		t.Errorf("expected 2 domains, got %v", domains)
	}
}

func TestParseObfuscateOptions_Seed(t *testing.T) {
	opts, err := ParseObfuscateOptions("-seed myseed123")
	if err != nil {
		t.Fatalf("ParseObfuscateOptions failed: %v", err)
	}
	seed, ok := opts["seed"].(string)
	if !ok {
		t.Fatalf("seed missing or wrong type: %v", opts["seed"])
	}
	if seed != "myseed123" {
		t.Errorf("expected seed=myseed123, got %q", seed)
	}
}

func TestParseObfuscateOptions_Target(t *testing.T) {
	opts, err := ParseObfuscateOptions("-target node")
	if err != nil {
		t.Fatalf("ParseObfuscateOptions failed: %v", err)
	}
	target, ok := opts["target"].(string)
	if !ok {
		t.Fatalf("target missing or wrong type: %v", opts["target"])
	}
	if target != "node" {
		t.Errorf("expected target=node, got %q", target)
	}
}

func TestParseObfuscateOptions_InvalidFlag(t *testing.T) {
	_, err := ParseObfuscateOptions("-nonExistentFlag")
	if err == nil {
		t.Error("expected error for unknown flag, got nil")
	}
}

func TestParseObfuscateOptions_MultipleFlags(t *testing.T) {
	opts, err := ParseObfuscateOptions("-controlFlowFlattening -deadCodeInjection -compact=false")
	if err != nil {
		t.Fatalf("ParseObfuscateOptions failed: %v", err)
	}
	if opts["controlFlowFlattening"] != true {
		t.Error("expected controlFlowFlattening=true")
	}
	if opts["deadCodeInjection"] != true {
		t.Error("expected deadCodeInjection=true")
	}
	if opts["compact"] != false {
		t.Error("expected compact=false")
	}
}

// ---------- ObfuscateWithOptions ----------

func TestObfuscateWithOptions_Default(t *testing.T) {
	js := []byte(`var x = 1; console.log(x);`)
	opts := map[string]any{
		"compact": true,
	}
	result, err := ObfuscateWithOptions(js, opts)
	if err != nil {
		t.Fatalf("ObfuscateWithOptions failed: %v", err)
	}
	if len(result) == 0 {
		t.Fatal("ObfuscateWithOptions returned empty result")
	}
}

func TestObfuscateWithOptions_EmptyOptions(t *testing.T) {
	js := []byte(`var y = 42;`)
	result, err := ObfuscateWithOptions(js, map[string]any{})
	if err != nil {
		t.Fatalf("ObfuscateWithOptions with empty options failed: %v", err)
	}
	if len(result) == 0 {
		t.Fatal("ObfuscateWithOptions returned empty result")
	}
}

func TestObfuscate_FlagString(t *testing.T) {
	js := []byte(`(function(){ var a = 1; return a; })();`)
	result, err := Obfuscate(js, "-compact=false")
	if err != nil {
		t.Fatalf("Obfuscate failed: %v", err)
	}
	if len(result) == 0 {
		t.Fatal("Obfuscate returned empty result")
	}
}

func TestObfuscate_InvalidJS(t *testing.T) {
	// javascript-obfuscator may or may not fail on invalid JS; just ensure no panic.
	_, _ = Obfuscate([]byte(`this is not valid javascript !!@@##`), "")
}

// ---------- optionToArray / optionToMap (internal helpers) ----------

func TestOptionToArray_Empty(t *testing.T) {
	result := optionToArray("")
	if len(result) != 0 {
		t.Errorf("expected empty, got %v", result)
	}
}

func TestOptionToArray_Single(t *testing.T) {
	result := optionToArray("foo")
	if len(result) != 1 || result[0] != "foo" {
		t.Errorf("expected [foo], got %v", result)
	}
}

func TestOptionToArray_Multiple(t *testing.T) {
	result := optionToArray("a,b, c ")
	if len(result) != 3 {
		t.Fatalf("expected 3 items, got %v", result)
	}
	if result[2] != "c" {
		t.Errorf("expected trimmed 'c', got %q", result[2])
	}
}

func TestOptionToMap_Empty(t *testing.T) {
	result := optionToMap("")
	if len(result) != 0 {
		t.Errorf("expected empty map, got %v", result)
	}
}

func TestOptionToMap_Pairs(t *testing.T) {
	result := optionToMap("key1=val1,key2=val2")
	if result["key1"] != "val1" {
		t.Errorf("expected key1=val1, got %q", result["key1"])
	}
	if result["key2"] != "val2" {
		t.Errorf("expected key2=val2, got %q", result["key2"])
	}
}

func TestOptionToMap_InvalidPair(t *testing.T) {
	// pair without '=' is silently skipped
	result := optionToMap("noequals")
	if len(result) != 0 {
		t.Errorf("expected empty map for invalid pair, got %v", result)
	}
}

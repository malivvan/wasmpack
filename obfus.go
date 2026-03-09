package wasmpack

import (
	_ "embed"
	"encoding/json"
	"flag"
	"fmt"
	"strings"

	"github.com/grafana/sobek"
)

//go:embed obfus.js
var obfusJS string

func optionToArray(option string) (result []string) {
	if option == "" {
		return []string{}
	}
	result = strings.Split(option, ",")
	for i := range result {
		result[i] = strings.TrimSpace(result[i])
	}
	return result
}

func optionToMap(option string) (result map[string]string) {
	result = make(map[string]string)
	if option == "" {
		return result
	}
	pairs := strings.Split(option, ",")
	for _, pair := range pairs {
		kv := strings.SplitN(pair, "=", 2)
		if len(kv) != 2 {
			continue
		}
		key := strings.TrimSpace(kv[0])
		value := strings.TrimSpace(kv[1])
		result[key] = value
	}
	return result
}
func ParseObfusOptions(args string) (map[string]any, error) {
	options := make(map[string]any)
	flags := flag.NewFlagSet("obfuscate", flag.ExitOnError)
	compact := flags.Bool("compact", true, "compact code (default: true)")
	controlFlowFlattening := flags.Bool("controlFlowFlattening", false, "enable control flow flattening (default: false)")
	controlFlowFlatteningThreshold := flags.Float64("controlFlowFlatteningThreshold", 0.75, "control flow flattening threshold (default: 0.75)")
	deadCodeInjection := flags.Bool("deadCodeInjection", false, "enable dead code injection (default: false)")
	deadCodeInjectionThreshold := flags.Float64("deadCodeInjectionThreshold", 0.4, "dead code injection threshold (default: 0.4)")
	debugProtection := flags.Bool("debugProtection", false, "enable debug protection (default: false)")
	debugProtectionInterval := flags.Int("debugProtectionInterval", 0, "debug protection interval (default: 0)")
	disableConsoleOutput := flags.Bool("disableConsoleOutput", false, "disable console output (default: false)")
	domainLock := flags.String("domainLock", "", "comma-separated list of domains to lock the code to (default: [])")
	domainLockRedirectURL := flags.String("domainLockRedirectURL", "about:blank", "URL to redirect if domain lock fails (default: about:blank)")
	exclude := flags.String("exclude", "", "comma-separated list of file names or globs to exclude from obfuscation (default: [])")
	forceTransformStrings := flags.String("forceTransformStrings", "", "comma-separated list of RegExp patterns to force transform string literals (default: [])")
	identifierNamesCache := flags.String("identifierNamesCache", "", "comma-separated list of key=value pairs for identifier names cache (default: null)")
	identifierNamesGenerator := flags.String("identifierNamesGenerator", "hexadecimal", "identifier names generator (default: hexadecimal)")
	identifiersDictionary := flags.String("identifiersDictionary", "", "comma-separated list of identifiers for dictionary generator (default: [])")
	identifiersPrefix := flags.String("identifiersPrefix", "", "prefix for all global identifiers (default: '')")
	ignoreImports := flags.Bool("ignoreImports", false, "prevent obfuscation of require imports (default: false)")
	inputFileName := flags.String("inputFileName", "", "name of the input file for source map generation (default: '')")
	log := flags.Bool("log", false, "enable logging (default: false)")
	numbersToExpressions := flags.Bool("numbersToExpressions", false, "enable numbers to expressions transformation (default: false)")
	optionsPreset := flags.String("optionsPreset", "default", "options preset (default: default)")
	renameGlobals := flags.Bool("renameGlobals", false, "enable renaming of global identifiers (default: false)")
	renameProperties := flags.Bool("renameProperties", false, "enable renaming of property identifiers (default: false)")
	renamePropertiesMode := flags.String("renamePropertiesMode", "safe", "mode for renaming property identifiers (default: safe)")
	reservedNames := flags.String("reservedNames", "", "comma-separated list of RegExp patterns to disable obfuscation of identifiers (default: none)")
	reservedStrings := flags.String("reservedStrings", "", "comma-separated list of RegExp patterns to disable transformation of string literals (default: [])")
	seed := flags.String("seed", "", "seed for random generator (default: '')")
	selfDefending := flags.Bool("selfDefending", false, "enable self defending (default: false)")
	simplify := flags.Bool("simplify", true, "enable code simplification (default: true)")
	sourceMap := flags.Bool("sourceMap", false, "enable source map generation (default: '')")
	sourceMapBaseURL := flags.String("sourceMapBaseURL", "", "base URL for sources in source map (default: '')")
	sourceMapFileName := flags.String("sourceMapFileName", "", "file name for output source map (default: '')")
	sourceMapMode := flags.String("sourceMapMode", "separate", "mode for source map generation (default: separate)")
	splitStrings := flags.Bool("splitStrings", false, "enable splitting of string literals (default: false)")
	splitStringsChunkLength := flags.Int("splitStringsChunkLength", 0, "chunk length for splitting string literals (default: 0)")
	stringArray := flags.Bool("stringArray", true, "enable string array transformation (default: true)")
	stringArrayCallsTransform := flags.Bool("stringArrayCallsTransform", false, "enable transformation of calls to the string array (default: false)")
	stringArrayCallsTransformThreshold := flags.Float64("stringArrayCallsTransformThreshold", 0.5, "threshold for transformation of calls to the string array (default: 0.5)")
	stringArrayEncoding := flags.String("stringArrayEncoding", "", "comma-separated list of encodings for string array values (default: none)")
	stringArrayIndexesType := flags.String("stringArrayIndexesType", "", "type of string array call indexes (default: ['hexadecimal-number'])")
	stringArrayIndexShift := flags.Bool("stringArrayIndexShift", true, "index shift for string array calls (default: true)")
	stringArrayRotate := flags.Bool("stringArrayRotate", true, "enable rotation of the string array (default: true)")
	stringArrayShuffle := flags.Bool("stringArrayShuffle", true, "enable shuffling of the string array (default: true)")
	stringArrayWrappersCount := flags.Int("stringArrayWrappersCount", 1, "count of wrappers for the string array (default: 1)")
	stringArrayWrappersChainedCalls := flags.Bool("stringArrayWrappersChainedCalls", true, "enable chained calls between string array wrappers (default: true)")
	stringArrayWrappersParametersMaxCount := flags.Int("stringArrayWrappersParametersMaxCount", 2, "maximum number of parameters for string array wrappers (default: 2)")
	stringArrayWrappersType := flags.String("stringArrayWrappersType", "variable", "type of wrappers for the string array (default: variable)")
	stringArrayThreshold := flags.Float64("stringArrayThreshold", 0.8, "threshold for inserting string literals into the string array (default: 0.8)")
	target := flags.String("target", "browser", "target environment for obfuscated code (default: browser)")
	transformObjectKeys := flags.Bool("transformObjectKeys", false, "enable transformation of object keys (default: false)")
	unicodeEscapeSequence := flags.Bool("unicodeEscapeSequence", false, "enable unicode escape sequence for string literals (default: false)")
	err := flags.Parse(splitArgs(args))
	if err != nil {
		return nil, err
	}
	if compact != nil && !*compact {
		options["compact"] = *compact
	}
	if controlFlowFlattening != nil && *controlFlowFlattening {
		options["controlFlowFlattening"] = *controlFlowFlattening
	}
	if controlFlowFlatteningThreshold != nil && *controlFlowFlatteningThreshold != 0.75 {
		options["controlFlowFlatteningThreshold"] = *controlFlowFlatteningThreshold
	}
	if deadCodeInjection != nil && *deadCodeInjection {
		options["deadCodeInjection"] = *deadCodeInjection
	}
	if deadCodeInjectionThreshold != nil && *deadCodeInjectionThreshold != 0.4 {
		options["deadCodeInjectionThreshold"] = *deadCodeInjectionThreshold
	}
	if debugProtection != nil && *debugProtection {
		options["debugProtection"] = *debugProtection
	}
	if debugProtectionInterval != nil && *debugProtectionInterval > 0 {
		options["debugProtectionInterval"] = *debugProtectionInterval
	}
	if disableConsoleOutput != nil && *disableConsoleOutput {
		options["disableConsoleOutput"] = *disableConsoleOutput
	}
	if domainLock != nil && *domainLock != "" {
		options["domainLock"] = optionToArray(*domainLock)
	}
	if domainLockRedirectURL != nil && *domainLockRedirectURL != "about:blank" {
		options["domainLockRedirectURL"] = *domainLockRedirectURL
	}
	if exclude != nil && *exclude != "" {
		options["exclude"] = optionToArray(*exclude)
	}
	if forceTransformStrings != nil && *forceTransformStrings != "" {
		options["forceTransformStrings"] = optionToArray(*forceTransformStrings)
	}
	if identifierNamesCache != nil && *identifierNamesCache != "" {
		options["identifierNamesCache"] = optionToMap(*identifierNamesCache)
	}
	if identifierNamesGenerator != nil && *identifierNamesGenerator != "hexadecimal" {
		options["identifierNamesGenerator"] = *identifierNamesGenerator
	}
	if identifiersDictionary != nil && *identifiersDictionary != "" {
		options["identifiersDictionary"] = optionToArray(*identifiersDictionary)
	}
	if identifiersPrefix != nil && *identifiersPrefix != "" {
		options["identifiersPrefix"] = *identifiersPrefix
	}
	if ignoreImports != nil && *ignoreImports {
		options["ignoreImports"] = *ignoreImports
	}
	if inputFileName != nil && *inputFileName != "" {
		options["inputFileName"] = *inputFileName
	}
	if log != nil && *log {
		options["log"] = *log
	}
	if numbersToExpressions != nil && *numbersToExpressions {
		options["numbersToExpressions"] = *numbersToExpressions
	}
	if optionsPreset != nil && *optionsPreset != "default" {
		options["optionsPreset"] = *optionsPreset
	}
	if renameGlobals != nil && *renameGlobals {
		options["renameGlobals"] = *renameGlobals
	}
	if renameProperties != nil && *renameProperties {
		options["renameProperties"] = *renameProperties
	}
	if renamePropertiesMode != nil && *renamePropertiesMode != "safe" {
		options["renamePropertiesMode"] = *renamePropertiesMode
	}
	if reservedNames != nil && *reservedNames != "" {
		options["reservedNames"] = optionToArray(*reservedNames)
	}
	if reservedStrings != nil && *reservedStrings != "" {
		options["reservedStrings"] = optionToArray(*reservedStrings)
	}
	if seed != nil && *seed != "" {
		options["seed"] = *seed
	}
	if selfDefending != nil && *selfDefending {
		options["selfDefending"] = *selfDefending
	}
	if simplify != nil && !*simplify {
		options["simplify"] = *simplify
	}
	if sourceMap != nil && *sourceMap {
		options["sourceMap"] = *sourceMap
	}
	if sourceMapBaseURL != nil && *sourceMapBaseURL != "" {
		options["sourceMapBaseURL"] = *sourceMapBaseURL
	}
	if sourceMapFileName != nil && *sourceMapFileName != "" {
		options["sourceMapFileName"] = *sourceMapFileName
	}
	if sourceMapMode != nil && *sourceMapMode != "separate" {
		options["sourceMapMode"] = *sourceMapMode
	}
	if splitStrings != nil && *splitStrings {
		options["splitStrings"] = *splitStrings
	}
	if splitStringsChunkLength != nil && *splitStringsChunkLength > 0 {
		options["splitStringsChunkLength"] = *splitStringsChunkLength
	}
	if stringArray != nil && !*stringArray {
		options["stringArray"] = *stringArray
	}
	if stringArrayCallsTransform != nil && *stringArrayCallsTransform {
		options["stringArrayCallsTransform"] = *stringArrayCallsTransform
	}
	if stringArrayCallsTransformThreshold != nil && *stringArrayCallsTransformThreshold != 0.5 {
		options["stringArrayCallsTransformThreshold"] = *stringArrayCallsTransformThreshold
	}

	if stringArrayEncoding != nil && *stringArrayEncoding != "" {
		options["stringArrayEncoding"] = optionToArray(*stringArrayEncoding)
	}
	if stringArrayIndexesType != nil && *stringArrayIndexesType != "" {
		if stringArrayIndexesTypeArray := optionToArray(*stringArrayIndexesType); len(stringArrayIndexesTypeArray) > 0 && !(len(stringArrayIndexesTypeArray) == 1 && stringArrayIndexesTypeArray[0] == "hexadecimal-number") {
			options["stringArrayIndexesType"] = stringArrayIndexesTypeArray
		}
	}
	if stringArrayIndexShift != nil && !*stringArrayIndexShift {
		options["stringArrayIndexShift"] = *stringArrayIndexShift
	}
	if stringArrayRotate != nil && !*stringArrayRotate {
		options["stringArrayRotate"] = *stringArrayRotate
	}
	if stringArrayShuffle != nil && !*stringArrayShuffle {
		options["stringArrayShuffle"] = *stringArrayShuffle
	}
	if stringArrayWrappersCount != nil && *stringArrayWrappersCount != 1 {
		options["stringArrayWrappersCount"] = *stringArrayWrappersCount
	}
	if stringArrayWrappersChainedCalls != nil && !*stringArrayWrappersChainedCalls {
		options["stringArrayWrappersChainedCalls"] = *stringArrayWrappersChainedCalls
	}
	if stringArrayWrappersParametersMaxCount != nil && *stringArrayWrappersParametersMaxCount != 2 {
		options["stringArrayWrappersParametersMaxCount"] = *stringArrayWrappersParametersMaxCount
	}
	if stringArrayWrappersType != nil && *stringArrayWrappersType != "variable" {
		options["stringArrayWrappersType"] = *stringArrayWrappersType
	}
	if stringArrayThreshold != nil && *stringArrayThreshold != 0.8 {
		options["stringArrayThreshold"] = *stringArrayThreshold
	}
	if target != nil && *target != "browser" {
		options["target"] = *target
	}
	if transformObjectKeys != nil && *transformObjectKeys {
		options["transformObjectKeys"] = *transformObjectKeys
	}
	if unicodeEscapeSequence != nil && *unicodeEscapeSequence {
		options["unicodeEscapeSequence"] = *unicodeEscapeSequence
	}

	return options, nil
}

func Obfus(js []byte, args string) ([]byte, error) {
	options, err := ParseObfusOptions(args)
	if err != nil {
		return nil, err
	}
	fmt.Printf("Obfuscating JavaScript with options: %+v\n", options)
	optionsJSON, err := json.MarshalIndent(options, "", "  ")
	fmt.Printf("Obfuscation options:\n%s\n", string(optionsJSON))
	vm := sobek.New()
	vm.SetFieldNameMapper(sobek.TagFieldNameMapper("json", true))
	vm.GlobalObject().Set("self", vm.GlobalObject())
	_, err = vm.RunString(obfusJS)
	if err != nil {
		return nil, err
	}
	obfuscator := vm.GlobalObject().Get("JavaScriptObfuscator")
	if obfuscator == sobek.Undefined() {
		return nil, fmt.Errorf("JavaScriptObfuscator is undefined")
	}
	obfuscate, ok := sobek.AssertFunction(obfuscator.ToObject(vm).Get("obfuscate"))
	if !ok {
		return nil, fmt.Errorf("obfuscate is not a function")
	}
	result, err := obfuscate(sobek.Undefined(), vm.ToValue(string(js)), vm.ToValue(options))
	if err != nil {
		return nil, err
	}
	return []byte(result.String()), nil
}

// Code generated by "stringer -type=Engine --trimprefix=Engine"; DO NOT EDIT.

package model

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[engineUnknown-0]
	_ = x[EngineCLI-1]
	_ = x[EngineNoop-2]
	_ = x[EngineDocker-3]
	_ = x[EngineWasm-4]
	_ = x[EngineLanguage-5]
	_ = x[EnginePythonWasm-6]
	_ = x[engineDone-7]
}

const _Engine_name = "engineUnknownCLINoopDockerWasmLanguagePythonWasmengineDone"

var _Engine_index = [...]uint8{0, 13, 16, 20, 26, 30, 38, 48, 58}

func (i Engine) String() string {
	if i < 0 || i >= Engine(len(_Engine_index)-1) {
		return "Engine(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _Engine_name[_Engine_index[i]:_Engine_index[i+1]]
}

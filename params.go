package gopiston

import "time"

// Function to pass the Params struct.
type Param func(*Params)

// Struct that contains all piston parameters.
type Params struct {
	requestBody *RequestBody
}

// Stdin (optional) The text to pass as stdin to the program. Must be a string or left out. Defaults to blank string.
func Stdin(input string) Param {
	return func(param *Params) {
		param.requestBody.Stdin = input
	}
}

// Args (optional) The arguments to pass to the program. Must be an array or left out. Defaults to [].
func Args(args []string) Param {
	return func(param *Params) {
		param.requestBody.Args = args
	}
}

// CompileTimeout (optional) The maximum time allowed for the compile stage to finish before bailing out in milliseconds. Must be a "time.Duration" object. Defaults to 10 seconds.
func CompileTimeout(timeout time.Duration) Param {

	return func(param *Params) {
		param.requestBody.CompileTimeout = int(timeout.Milliseconds())
	}
}

// RunTimeout (optional) The maximum time allowed for the run stage to finish before bailing out in milliseconds. Must be a "time.Duration" object. Defaults to 3 seconds.
func RunTimeout(timeout time.Duration) Param {

	return func(param *Params) {
		param.requestBody.RunTimeout = int(timeout.Milliseconds())
	}
}

// CompileMemoryLimit (optional) The maximum amount of memory the compile stage is allowed to use in bytes. Must be a number or left out. Defaults to -1 (no limit)
func CompileMemoryLimit(limit int) Param {
	return func(param *Params) {
		param.requestBody.CompileMemoryLimit = limit
	}

}

// RunMemoryLimit (optional) The maximum amount of memory the run stage is allowed to use in bytes. Must be a number or left out. Defaults to -1 (no limit)
func RunMemoryLimit(limit int) Param {

	return func(param *Params) {
		param.requestBody.RunMemoryLimit = limit
	}
}

// CompileCpuTime (optional) The maximum CPU-time allowed for the compile stage to finish before bailing out in milliseconds. Must be a "time.Duration" object. Defaults to 10 seconds.
func CompileCpuTime(timeout time.Duration) Param {
	return func(param *Params) {
		param.requestBody.CompileCpuTime = int(timeout.Milliseconds())
	}
}

// RunCpuTime (optional) The maximum CPU-time allowed for the run stage to finish before bailing out in milliseconds. Must be a "time.Duration" object. Defaults to 3 seconds.
func RunCpuTime(timeout time.Duration) Param {
	return func(param *Params) {
		param.requestBody.RunCpuTime = int(timeout.Milliseconds())
	}
}

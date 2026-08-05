package gopiston

import "time"

// Param is an optional setting for an execution job, passed to Execute.
type Param func(*Params)

// Params carries the optional settings applied to an execution job.
type Params struct {
	requestBody *RequestBody
}

// Stdin sets the text passed to the program's standard input. It defaults to
// the empty string. Piston appends a trailing newline if one is not present.
func Stdin(input string) Param {
	return func(param *Params) {
		param.requestBody.Stdin = input
	}
}

// Args sets the command line arguments passed to the program. It defaults to
// no arguments.
func Args(args []string) Param {
	return func(param *Params) {
		param.requestBody.Args = args
	}
}

// CompileTimeout sets how long the compile stage may run before it is killed.
// It defaults to the instance's configured maximum, typically 10 seconds, and
// may not exceed it.
func CompileTimeout(timeout time.Duration) Param {
	return func(param *Params) {
		param.requestBody.CompileTimeout = int(timeout.Milliseconds())
	}
}

// RunTimeout sets how long the run stage may run before it is killed. It
// defaults to the instance's configured maximum, typically 3 seconds, and may
// not exceed it.
func RunTimeout(timeout time.Duration) Param {
	return func(param *Params) {
		param.requestBody.RunTimeout = int(timeout.Milliseconds())
	}
}

// CompileMemoryLimit sets the maximum memory, in bytes, the compile stage may
// use. It defaults to the instance's configured maximum, or to no limit when
// none is configured.
func CompileMemoryLimit(limit int) Param {
	return func(param *Params) {
		param.requestBody.CompileMemoryLimit = limit
	}
}

// RunMemoryLimit sets the maximum memory, in bytes, the run stage may use. It
// defaults to the instance's configured maximum, or to no limit when none is
// configured.
func RunMemoryLimit(limit int) Param {
	return func(param *Params) {
		param.requestBody.RunMemoryLimit = limit
	}
}

// CompileCpuTime sets the maximum CPU time the compile stage may consume
// before it is killed. It defaults to the instance's configured maximum,
// typically 10 seconds, and may not exceed it.
func CompileCpuTime(timeout time.Duration) Param {
	return func(param *Params) {
		param.requestBody.CompileCpuTime = int(timeout.Milliseconds())
	}
}

// RunCpuTime sets the maximum CPU time the run stage may consume before it is
// killed. It defaults to the instance's configured maximum, typically 3
// seconds, and may not exceed it.
func RunCpuTime(timeout time.Duration) Param {
	return func(param *Params) {
		param.requestBody.RunCpuTime = int(timeout.Milliseconds())
	}
}

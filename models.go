package gopiston

// Runtime is a language runtime installed on a Piston instance.
type Runtime struct {
	// Language is the runtime's language name, as accepted by Execute.
	Language string `json:"language"`

	// Version is the installed version of the language.
	Version string `json:"version"`

	// Aliases are alternative names accepted by Execute in place of Language.
	Aliases []string `json:"aliases"`

	// Runtime names the engine running the language, and is set only when the
	// language has more than one possible engine (for example "node" or "deno"
	// for JavaScript).
	Runtime string `json:"runtime,omitempty"`
}

// Runtimes is a list of runtimes installed on a Piston instance.
type Runtimes []Runtime

// Code is a single file making up an execution job. The first file passed to
// Execute is the entry point.
type Code struct {
	// Name is the file name to write. It must not contain a path; Files sets
	// it from the base name. When empty, Piston picks a name.
	Name string `json:"name,omitempty"`

	// Content is the file's contents, interpreted according to Encoding.
	Content string `json:"content"`

	// Encoding is the encoding of Content: "utf8" (the default), "base64" or
	// "hex".
	Encoding string `json:"encoding,omitempty"`
}

// RequestBody is the payload sent to the execute endpoint. Execute builds it;
// the Param options mutate it.
type RequestBody struct {
	Language           string   `json:"language"`
	Version            string   `json:"version"`
	Files              []Code   `json:"files"`
	Stdin              string   `json:"stdin,omitempty"`
	Args               []string `json:"args,omitempty"`
	CompileTimeout     int      `json:"compile_timeout,omitempty"`
	RunTimeout         int      `json:"run_timeout,omitempty"`
	CompileCpuTime     int      `json:"compile_cpu_time,omitempty"`
	RunCpuTime         int      `json:"run_cpu_time,omitempty"`
	CompileMemoryLimit int      `json:"compile_memory_limit,omitempty"`
	RunMemoryLimit     int      `json:"run_memory_limit,omitempty"`
}

// PistonExecution is the result of an execution job.
type PistonExecution struct {
	// Language is the name — never an alias — of the runtime that ran the job.
	Language string `json:"language"`

	// Version is the version of the runtime that ran the job.
	Version string `json:"version"`

	// Run holds the results of the run stage.
	Run Stage `json:"run"`

	// Compile holds the results of the compile stage, and is nil for languages
	// that have no compile stage.
	Compile *Stage `json:"compile,omitempty"`
}

// Stage is the result of one stage of an execution job.
//
// Exactly one of Code and Signal is meaningful: a process killed by a signal
// reports no exit code, in which case Piston sends a null code that decodes to
// zero here. Check Signal before trusting Code.
type Stage struct {
	// Stdout is everything the stage wrote to standard output.
	Stdout string `json:"stdout"`

	// Stderr is everything the stage wrote to standard error.
	Stderr string `json:"stderr"`

	// Output is stdout and stderr interleaved in the order produced.
	Output string `json:"output"`

	// Code is the process's exit code. See the note on Stage about Signal.
	Code int `json:"code"`

	// Signal is the signal that killed the process, if any. A stage that hit
	// its timeout or output limit reports "SIGKILL".
	Signal string `json:"signal"`

	// Message describes why the stage was terminated early, if it was.
	Message string `json:"message"`

	// Status is Piston's termination status code, such as "TO" for a timeout
	// or "OL"/"EL" for exceeding the stdout/stderr limits.
	Status string `json:"status"`

	// CpuTime is the CPU time consumed, in milliseconds.
	CpuTime float64 `json:"cpu_time"`

	// WallTime is the elapsed wall-clock time, in milliseconds.
	WallTime float64 `json:"wall_time"`

	// Memory is the peak memory used, in bytes.
	Memory float64 `json:"memory"`
}

// Package is an installable runtime known to a Piston instance.
type Package struct {
	// Language is the name of the contained runtime.
	Language string `json:"language"`

	// LanguageVersion is the version of the contained runtime.
	LanguageVersion string `json:"language_version"`

	// Installed reports whether the package is currently installed.
	Installed bool `json:"installed"`
}

// packageRequestBody is the payload sent to install or uninstall a package.
type packageRequestBody struct {
	Language string `json:"language"`
	Version  string `json:"version"`
}

// PackageInstallation identifies the package affected by an install or
// uninstall.
type PackageInstallation struct {
	Language string `json:"language"`
	Version  string `json:"version"`
}

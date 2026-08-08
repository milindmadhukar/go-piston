package gopiston

import "time"

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

// File is one file making up an execution job. The first file passed to
// Execute is the entry point.
type File struct {
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
	Files              []File   `json:"files"`
	Stdin              string   `json:"stdin,omitempty"`
	Args               []string `json:"args,omitempty"`
	CompileTimeout     int      `json:"compile_timeout,omitempty"`
	RunTimeout         int      `json:"run_timeout,omitempty"`
	CompileCpuTime     int      `json:"compile_cpu_time,omitempty"`
	RunCpuTime         int      `json:"run_cpu_time,omitempty"`
	CompileMemoryLimit int      `json:"compile_memory_limit,omitempty"`
	RunMemoryLimit     int      `json:"run_memory_limit,omitempty"`
}

// Execution is the result of an execution job.
type Execution struct {
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

// EventType identifies which kind of message a Session received.
type EventType string

const (
	// EventRuntime reports the runtime chosen for the job, and is the first
	// event of a session. Language and Version are set.
	EventRuntime EventType = "runtime"

	// EventStage reports that a stage started. Stage is set to "compile" or
	// "run".
	EventStage EventType = "stage"

	// EventStdout carries a chunk the process wrote to standard output. Data
	// is set.
	EventStdout EventType = "stdout"

	// EventStderr carries a chunk the process wrote to standard error. Data
	// is set.
	EventStderr EventType = "stderr"

	// EventExit reports that a stage ended. Stage is set, along with exactly
	// one of Code and Signal.
	EventExit EventType = "exit"

	// EventError reports a fatal error. The instance closes the connection
	// immediately afterwards, so the following call to Next returns an error
	// carrying this Message.
	EventError EventType = "error"
)

// Event is a single message from an interactive Session. Which fields are set
// depends on Type; see the EventType constants.
//
// EventStdout and EventStderr both correspond to the protocol's "data"
// message, whose stream is only ever stdout or stderr, and are split here so
// callers can switch on Type alone.
type Event struct {
	// Type identifies the event and determines which other fields are set.
	Type EventType

	// Language is the name of the runtime that ran the job. EventRuntime only.
	Language string

	// Version is the version of the runtime that ran the job. EventRuntime
	// only.
	Version string

	// Stage is "compile" or "run". EventStage and EventExit only.
	Stage string

	// Data is a chunk of process output. EventStdout and EventStderr only. It
	// is a chunk, not a line: a single write may arrive split across events,
	// and a chunk may hold several lines.
	Data string

	// Code is the stage's exit code, or nil when a signal ended it. EventExit
	// only. It is a pointer because exactly one of Code and Signal is
	// meaningful, and the instance sends a null code when Signal applies.
	Code *int

	// Signal is the signal that ended the stage, or empty when it exited
	// normally. EventExit only.
	Signal string

	// Message describes the failure. EventError only.
	Message string
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

// OperationKind distinguishes the two things a package operation can do.
type OperationKind string

// The kinds of package operation.
const (
	OperationInstall   OperationKind = "install"
	OperationUninstall OperationKind = "uninstall"
)

// OperationState is where a package operation has got to.
type OperationState string

// The states a package operation passes through. Running is the only
// non-terminal one.
const (
	OperationRunning   OperationState = "running"
	OperationSucceeded OperationState = "succeeded"
	OperationFailed    OperationState = "failed"
)

// operationRequestBody is the payload sent to start a package operation.
type operationRequestBody struct {
	Kind     OperationKind `json:"kind"`
	Language string        `json:"language"`
	Version  string        `json:"version"`
}

// Operation is an install or uninstall running in the background on the
// instance. See Client.StartOperation.
//
// An Operation is a record of work in flight rather than durable state: the
// instance keeps only the most recent completed ones and forgets all of them
// when it restarts. What survives is the effect — whether the package is
// installed — which GetPackages reports.
type Operation struct {
	// ID identifies the operation for GetOperation, GetOperationLog and
	// ConnectOperation.
	ID string `json:"id"`

	// Kind is whether this operation installs or uninstalls.
	Kind OperationKind `json:"kind"`

	// Language and Version are the resolved package, so Version is a concrete
	// version even when a selector such as "5.x" was requested.
	Language string `json:"language"`
	Version  string `json:"version"`

	// State is the operation's progress. Compare against OperationRunning,
	// OperationSucceeded and OperationFailed, or use Done.
	State OperationState `json:"state"`

	// Started is when the operation began, in Unix milliseconds. StartedAt
	// returns it as a time.Time.
	Started int64 `json:"started"`

	// Finished is when the operation settled, in Unix milliseconds. It is nil
	// while the operation is still running.
	Finished *int64 `json:"finished,omitempty"`

	// Error describes the failure when State is OperationFailed, and is empty
	// otherwise.
	Error string `json:"error,omitempty"`
}

// Done reports whether the operation has settled, successfully or not.
func (operation Operation) Done() bool { return operation.State != OperationRunning }

// StartedAt returns when the operation began.
func (operation Operation) StartedAt() time.Time {
	return time.UnixMilli(operation.Started)
}

// FinishedAt returns when the operation settled, or the zero time if it is
// still running.
func (operation Operation) FinishedAt() time.Time {
	if operation.Finished == nil {
		return time.Time{}
	}
	return time.UnixMilli(*operation.Finished)
}

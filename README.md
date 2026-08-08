# go-piston

A Go client library for the [Piston](https://github.com/engineer-man/piston) code execution engine, covering the `runtimes`, `execute`, `packages` and `operations` endpoints, plus the interactive `connect` WebSocket.

Piston instances vary in what they serve — `operations` is a later addition that older ones do not have. Nothing here assumes it: `SupportsOperations` reports what an instance can do, and everything else works against any v2 instance.

## Piston API access

> **Important:** As of February 15, 2026, the public Piston API is no longer freely available. To obtain an API key, contact `EngineerMan` on Discord after reading the eligibility criteria below.
>
> Keys are not granted for:
> - Projects that cost money
> - Temporary projects
> - Individual projects
> - Portfolio projects
> - High school, college, or university assignments
> - Conceptual projects
> - Vibe-coded AI slop projects
> - Projects that generally don't benefit anyone
>
> Key issuance is entirely at the maintainer's discretion. If a key is not issued, you are welcome to run your own instance of Piston, since it is open source.

Because of this, most users of this library should point it at a self-hosted Piston instance rather than the official API. `NewClient` is designed around that: pass your instance's base URL directly, and opt into the official API explicitly if you have a key.

## Installation

```
go get github.com/milindmadhukar/go-piston/v2
```

Coming from v1? See [Migrating from v1](#migrating-from-v1).

## Usage

### Self-hosted instance (recommended)

```go
client := piston.NewClient("http://localhost:2000/api/v2/")
```

The base URL works with or without a trailing slash. `client.BaseURL()` returns the normalized form.

### Official Piston API (requires an API key)

```go
client := piston.NewClient(piston.OfficialAPIBaseURL, piston.WithAPIKey("your-key"))
```

`NewClient` also accepts `piston.WithHTTPClient` to supply a custom `*http.Client`.

A client's target is fixed at construction and cannot be changed afterwards, so `client.IsOfficialAPI()` is always an accurate answer to "am I talking to the public API?".

### Context

Every request method takes a `context.Context` as its first argument, which is passed through to the underlying HTTP request. Use it to apply timeouts or cancellation to calls made against your Piston instance:

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

execution, err := client.Execute(ctx, "python", "", files)
```

This is separate from the limits Piston applies to the executed code itself, which are set with `piston.RunTimeout` and the other options.

### Example

```go
package main

import (
	"context"
	"fmt"
	"log"

	piston "github.com/milindmadhukar/go-piston/v2"
)

func main() {
	client := piston.NewClient("http://localhost:2000/api/v2/")
	ctx := context.Background()

	execution, err := client.Execute(ctx, "python", "", // An empty version uses the highest installed version.
		[]piston.File{
			{Content: "inp = input()\nprint(inp[::-1])"},
		}, // Files to execute.
		piston.Stdin("hello world"), // Input passed to the program via stdin.
	)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(execution.GetOutput())
}
```

Output:

```
dlrow olleh
```

See the [examples directory](examples) for more, including timeouts, memory limits, multi-file execution, error handling, and configuring clients for self-hosted vs. the official API. Each is a runnable program; CI executes every one of them against a live instance on each push.

Shorter, copy-paste snippets for each API also appear as [examples on pkg.go.dev](https://pkg.go.dev/github.com/milindmadhukar/go-piston/v2#pkg-examples).

## Error handling

A non-2xx response is returned as an `*piston.APIError` carrying the status code and the instance's own message. Sentinel errors let you classify a failure without parsing text:

```go
switch {
case errors.Is(err, piston.ErrAPIKeyRequired):
	// this instance needs a key; pass piston.WithAPIKey(...)
case errors.Is(err, piston.ErrUnauthorized):
	// the configured key was rejected
case errors.Is(err, piston.ErrRateLimited):
	// back off and retry
}

var apiErr *piston.APIError
if errors.As(err, &apiErr) {
	log.Printf("status=%d message=%q", apiErr.StatusCode, apiErr.Message)
}
```

The full set is `ErrBadRequest`, `ErrUnauthorized`, `ErrAPIKeyRequired`, `ErrNotFound`, `ErrConflict`, `ErrRateLimited`, `ErrServer`, `ErrLanguageNotFound`, `ErrUnsupportedByOfficialAPI` and `ErrOperationsUnsupported`. `ErrAPIKeyRequired` implies `ErrUnauthorized`, so check it first; it is derived from the client's own configuration rather than the server's wording. `ErrOperationsUnsupported` likewise implies `ErrNotFound`.

Not every failure carries a JSON body, so do not rely on `APIError.Message` being set:

- `Execute` answers an internal failure with a **500 and an empty body**. `Message` is empty and `Error()` falls back to the status text; the raw body is always on `Body`.
- A request body that is not valid JSON is rejected with `{"stack": ...}` rather than Piston's usual `{"message": ...}`. The first line is used as `Message`.

Self-hosted Piston does **not** rate limit, so `ErrRateLimited` is only seen from the official API or a proxy. Instead, work beyond `PISTON_MAX_CONCURRENT_JOBS` queues on the instance until a slot frees — so give requests a generous context deadline rather than a short client timeout.

## Interactive execution

`Execute` sends all input up front and returns everything at once when the job ends. `Connect` instead opens a WebSocket session against the instance's `/connect` endpoint, streaming output as it is produced and accepting input while the process is still running:

```go
session, err := client.Connect(ctx, "python", "",
	[]piston.File{{Name: "main.py", Content: source}})
if err != nil {
	log.Fatal(err)
}
defer session.Close()

for {
	event, err := session.Next(ctx)
	if errors.Is(err, io.EOF) {
		break // the job finished
	}
	if err != nil {
		log.Fatal(err)
	}

	switch event.Type {
	case piston.EventStage:
		if event.Stage == "run" {
			session.SendStdin(ctx, "hello\n")
		}
	case piston.EventStdout:
		fmt.Print(event.Data)
	case piston.EventExit:
		// exactly one of event.Code and event.Signal is set
	}
}
```

The event types are `EventRuntime`, `EventStage`, `EventStdout`, `EventStderr`, `EventExit` and `EventError`. A session that completes normally ends with `io.EOF`; anything else is a real failure, including the close-code sentinels `ErrInitializationTimeout`, `ErrAlreadyInitialized`, `ErrNotInitialized`, `ErrStdinOnly` and `ErrInvalidSignal`.

A session may be read from one goroutine while another writes to it, but `Next` must not be called concurrently with itself.

> **Self-hosted only.** Interactive execution is not available on the official Piston API; a client targeting it fails with `ErrUnsupportedByOfficialAPI` without connecting.

Two practical notes:

- `Event.Data` is a **chunk, not a line** — one write may arrive split across events, and one event may carry several lines.
- Most runtimes **block-buffer stdout** when it is not a terminal, so a program that never flushes delivers nothing until it exits. Flush on the other side (`print(..., flush=True)` in Python).

### Stopping a job

`Close` is the portable way to stop a job early. On instances that support it, closing the session kills the running stage and releases the sandbox and concurrency slot immediately; older ones let the job run on until its own wall-time limit.

`SendSignal` delivers a signal by name to the sandbox supervisor. Two accepted cases deliberately do nothing on the instance's side, and neither is an error — a job still running afterwards has not misbehaved:

- `SIGSTOP`, `SIGTSTP`, `SIGTTIN` and `SIGTTOU` are never delivered, because a stopped job would hold its concurrency slot forever.
- The real-time signal names (`SIGRTMIN+n`, `SIGRTMAX-n`) are accepted and have no effect.

An unrecognised name is a real error and ends the session with `ErrInvalidSignal`.

> **Older instances drop signals entirely.** Before this was fixed, Piston's router emitted an event named `signal` while the job handler listened for one named `kill`, so a signal was validated, accepted and then discarded — a process sent `SIGKILL` ran to completion and exited 0. If you must stop a job on an arbitrary instance, close the session.

## Package management

`GetPackages`, `InstallPackage` and `UninstallPackage` operate on a Piston instance's own runtime store, so they are only available when self-hosting. On a client targeting the official API they fail with `ErrUnsupportedByOfficialAPI` without making a request.

```go
packages, err := client.GetPackages(ctx)
installation, err := client.InstallPackage(ctx, "python", "3.10.0")
```

Listing packages makes the instance consult the upstream package index and can be slow; installing one downloads and unpacks a runtime and can take minutes. Give both a generous context timeout.

### Installing in the background

`InstallPackage` holds the request open until the runtime is installed. That is workable for a package that is a tarball to fetch, and not for one compiled from source, which can take an hour. The `operations` endpoints start the same work and return immediately with an id to follow:

```go
operation, err := client.InstallPackageAsync(ctx, "gcc", "15.3.0")

session, err := client.ConnectOperation(ctx, operation.ID)
defer session.Close()

for {
	event, err := session.Next(ctx)
	if errors.Is(err, io.EOF) {
		break // the instance closes the socket once the operation settles
	}
	if err != nil {
		return err
	}
	if event.Type == piston.OperationEventLog {
		fmt.Print(event.Data)
	}
}
```

Everything logged before the connection opened is replayed first, so attaching late still shows the whole run. `GetOperation` and `GetOperationLog` poll for the same information instead, and both work while the operation is still running.

> **Not on every instance.** These endpoints are a later addition. Probe once with `SupportsOperations` and keep the answer; an instance without them answers `ErrOperationsUnsupported`, and `InstallPackage` still works there.

Three things worth knowing:

- Only one operation per package may be in flight; a second fails with `ErrConflict`.
- An operation is a record of work in flight, not durable state. The instance keeps only the most recent completed ones and forgets all of them on restart, after which `GetOperation` reports `ErrNotFound` — which is indistinguishable from an id that never existed. The durable answer to "is it installed" is `GetPackages`.
- `Operation.Finished` is a `*int64` because the field is **absent** while running, not null. `FinishedAt` returns the zero time in that case.

`ServerVersion` reports what an instance calls itself (`"3.1.1"`). It is a diagnostic only: the endpoint it reads lives at the instance root rather than under the API base URL, so it fails behind a proxy that exposes only `/api/v2`. Never gate a feature on it — use `SupportsOperations`, which asks the endpoint in question directly.

## Testing

The unit tests need no network and no configuration:

```
go test ./...
```

The live tests run against a real Piston instance. Point them at a self-hosted one:

```
export PISTON_BASE_URL=http://localhost:2000/api/v2/
go test ./...
```

Without `PISTON_BASE_URL` they target the official API, which needs `PISTON_API_KEY`; execution tests skip themselves when no key is configured. Tests that need a specific language check the instance's installed runtimes first and skip if it is unavailable, so a self-hosted instance does not need every language installed.

Package install/uninstall tests mutate the target instance, so they are skipped unless `PISTON_TEST_PACKAGE_MANAGEMENT=true`. Choose the package with `PISTON_TEST_PACKAGE` (for example `bash=5.2.0`); avoid large runtimes, which can take many minutes to install.

Tests for the `operations` endpoints probe the instance first and skip when it does not serve them, so the suite is meaningful against any instance.

CI runs the full suite against a Piston instance started in Docker, so it passes without any secret. It does so **twice**, as a matrix over upstream Piston and the fork that adds the `operations` endpoints: upstream is the compatibility floor, where everything using those endpoints must skip rather than fail, and the fork is where that code is actually exercised. If a `PISTON_API_KEY` repository secret is set, a further job runs the suite against the official API; without the secret that job is skipped rather than failed.

## Migrating from v1

v2 changes the import path and the client API. v1 remains available at the unsuffixed path, so existing code keeps working until you choose to move.

Update the import to carry the `/v2` suffix:

```go
import piston "github.com/milindmadhukar/go-piston/v2"
```

Then apply the renames:

| v1 | v2 |
| --- | --- |
| `CreateDefaultClient()`, `New(key, http, url)` | `NewClient(baseURL, ...ClientOption)` |
| `Code` | `File` — matching the API's own vocabulary, so `Files()` returns `[]File` |
| `PistonExecution` | `Execution` |
| `GetRuntimes() (*Runtimes, error)` | `([]Runtime, error)` |
| `GetLanguages() *[]string` | `([]string, error)` — no longer swallows the error |
| `GetPackages() (*[]Package, error)` | `([]Package, error)` |
| `PistonExecution.Compile Stage` | `Execution.Compile *Stage` — nil means no compile stage |
| `client.BaseURL`, `client.ApiKey`, `client.HttpClient` | unexported; use `BaseURL()` and `IsOfficialAPI()` |
| errors from `errors.New` | typed `*APIError` plus sentinels — see [Error handling](#error-handling) |

Two behavioral changes have no compile error to point at them, so they are worth checking by hand:

- **`Execute` with an empty version** now sends the `*` selector and lets the instance resolve it. v1 picked the first matching runtime from `/runtimes`, which was not necessarily the newest — an instance with several versions of one language could silently run the older one.
- **`Files()` sends the base name** of each path. v1 sent the path as given, which the API rejects, since a file name must contain no path.

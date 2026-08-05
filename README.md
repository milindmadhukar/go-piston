# go-piston

A Go client library for the [Piston](https://github.com/engineer-man/piston) code execution engine, covering the `runtimes`, `execute`, and `packages` endpoints.

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
go get github.com/milindmadhukar/go-piston
```

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

	piston "github.com/milindmadhukar/go-piston"
)

func main() {
	client := piston.NewClient("http://localhost:2000/api/v2/")
	ctx := context.Background()

	execution, err := client.Execute(ctx, "python", "", // An empty version uses the highest installed version.
		[]piston.Code{
			{Content: "inp = input()\nprint(inp[::-1])"},
		}, // Code to execute.
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

See the [examples directory](_examples) for more, including timeouts, memory limits, multi-file execution, error handling, and configuring clients for self-hosted vs. the official API.

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

The full set is `ErrBadRequest`, `ErrUnauthorized`, `ErrAPIKeyRequired`, `ErrNotFound`, `ErrRateLimited`, `ErrServer`, `ErrLanguageNotFound` and `ErrUnsupportedByOfficialAPI`. `ErrAPIKeyRequired` implies `ErrUnauthorized`, so check it first; it is derived from the client's own configuration rather than the server's wording.

## Package management

`GetPackages`, `InstallPackage` and `UninstallPackage` operate on a Piston instance's own runtime store, so they are only available when self-hosting. On a client targeting the official API they fail with `ErrUnsupportedByOfficialAPI` without making a request.

```go
packages, err := client.GetPackages(ctx)
installation, err := client.InstallPackage(ctx, "python", "3.10.0")
```

Listing packages makes the instance consult the upstream package index and can be slow; installing one downloads and unpacks a runtime and can take minutes. Give both a generous context timeout.

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

CI runs the full suite against a Piston instance started in Docker, so it passes without any secret. If a `PISTON_API_KEY` repository secret is set, a second job additionally runs the suite against the official API; without the secret that job is skipped rather than failed.

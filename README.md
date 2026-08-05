# go-piston

A Go client library for the [Piston](https://github.com/engineer-man/piston) code execution engine, covering the `runtimes` and `execute` endpoints.

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

### Official Piston API (requires an API key)

```go
client := piston.NewClient(piston.OfficialAPIBaseURL, piston.WithAPIKey("your-key"))
```

`NewClient` also accepts `piston.WithHTTPClient` to supply a custom `*http.Client`.

### Context

Every request method takes a `context.Context` as its first argument, which is passed through to the underlying HTTP request. Use it to apply timeouts or cancellation to calls made against your Piston instance:

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

execution, err := client.Execute(ctx, "python", "", files)
```

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

	execution, err := client.Execute(ctx, "python", "", // Language and version; an empty version resolves to the latest supported version.
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

See the [examples directory](_examples) for more, including timeouts, memory limits, multi-file execution, and configuring clients for self-hosted vs. the official API.

## Testing

The test suite runs against a live Piston instance. By default it targets the official API and requires an API key:

```
export PISTON_API_KEY=your-key
go test ./...
```

To run it against a self-hosted instance instead, set `PISTON_BASE_URL`:

```
export PISTON_BASE_URL=http://localhost:2000/api/v2/
go test ./...
```

Tests that need a specific language (e.g. Python or C++) check the target instance's installed runtimes first and skip themselves if that language isn't available, so a self-hosted instance doesn't need every language installed to run the suite.

CI runs the suite against the official API using a `PISTON_API_KEY` repository secret. Pull requests from forks won't have access to that secret, so those CI runs are expected to fail until a maintainer adds it or reruns the job with access.

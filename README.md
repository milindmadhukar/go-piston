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

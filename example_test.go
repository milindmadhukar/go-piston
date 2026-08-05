package gopiston_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"time"

	gopiston "github.com/milindmadhukar/go-piston/v2"
)

// These examples have no "Output:" comment on purpose: an example that
// declares one is executed by go test, which would require a live Piston
// instance and make the unit suite depend on the network. Without one they are
// still compiled on every go test and still rendered on pkg.go.dev.

func ExampleNewClient() {
	// Most callers should self-host Piston and pass their instance's URL.
	// The trailing slash is optional.
	client := gopiston.NewClient("http://localhost:2000/api/v2/")

	fmt.Println(client.BaseURL(), client.IsOfficialAPI())
}

func ExampleNewClient_officialAPI() {
	// The official API is whitelist-only and needs a key.
	client := gopiston.NewClient(
		gopiston.OfficialAPIBaseURL,
		gopiston.WithAPIKey("your-key"),
	)

	fmt.Println(client.IsOfficialAPI())
}

func ExampleClient_Execute() {
	client := gopiston.NewClient("http://localhost:2000/api/v2/")

	// An empty version runs the highest installed version of the language.
	execution, err := client.Execute(context.Background(), "python", "",
		[]gopiston.File{
			{Content: "print('hello')"},
		})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Print(execution.GetOutput())
}

// Optional behavior is supplied through Param options.
func ExampleClient_Execute_withOptions() {
	client := gopiston.NewClient("http://localhost:2000/api/v2/")

	execution, err := client.Execute(context.Background(), "python", "",
		[]gopiston.File{
			{Content: "import sys\nprint(sys.stdin.read(), sys.argv[1:])"},
		},
		gopiston.Stdin("piped input"),
		gopiston.Args([]string{"first", "second"}),
		gopiston.RunTimeout(2*time.Second),
		gopiston.RunMemoryLimit(64*1024*1024),
	)
	if err != nil {
		log.Fatal(err)
	}

	// A stage killed by a timeout or memory limit reports a signal rather
	// than a meaningful exit code.
	if execution.Run.Signal != "" {
		fmt.Println("killed by", execution.Run.Signal)
		return
	}
	fmt.Print(execution.GetOutput())
}

// A compiled language reports a compile stage; an interpreted one leaves it
// nil.
func ExampleClient_Execute_compiled() {
	client := gopiston.NewClient("http://localhost:2000/api/v2/")

	execution, err := client.Execute(context.Background(), "c++", "",
		[]gopiston.File{
			{Name: "main.cpp", Content: "#include <iostream>\nint main(){std::cout<<\"hi\";}"},
		})
	if err != nil {
		log.Fatal(err)
	}

	if execution.Compile != nil && execution.Compile.Code != 0 {
		fmt.Println("compilation failed:", execution.Compile.Stderr)
		return
	}
	fmt.Print(execution.GetOutput())
}

// Files reads a job's source from disk instead of embedding it in a string.
// The first file is the entry point.
func ExampleFiles() {
	client := gopiston.NewClient("http://localhost:2000/api/v2/")

	files, err := gopiston.Files("main.py", "helper.py")
	if err != nil {
		log.Fatal(err)
	}

	execution, err := client.Execute(context.Background(), "python", "", files)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Print(execution.GetOutput())
}

// Connect streams a job's output while it runs, and accepts input while the
// process is still alive — neither of which Execute can do.
func ExampleClient_Connect() {
	client := gopiston.NewClient("http://localhost:2000/api/v2/")

	session, err := client.Connect(context.Background(), "python", "",
		[]gopiston.File{{Content: "import sys\nprint(sys.stdin.readline().strip(), flush=True)"}})
	if err != nil {
		log.Fatal(err)
	}
	defer session.Close()

	ctx := context.Background()

	for {
		event, err := session.Next(ctx)
		if errors.Is(err, io.EOF) {
			// The job finished and the instance closed the connection.
			break
		}
		if err != nil {
			log.Fatal(err)
		}

		switch event.Type {
		case gopiston.EventStage:
			// The process is running, so it can be written to now.
			if event.Stage == "run" {
				if err := session.SendStdin(ctx, "hello\n"); err != nil {
					log.Fatal(err)
				}
			}
		case gopiston.EventStdout:
			// Data is a chunk, not a line.
			fmt.Print(event.Data)
		case gopiston.EventExit:
			// Exactly one of Code and Signal is set.
			if event.Signal != "" {
				fmt.Println("killed by", event.Signal)
			}
		}
	}
}

func ExampleClient_GetRuntimes() {
	client := gopiston.NewClient("http://localhost:2000/api/v2/")

	runtimes, err := client.GetRuntimes(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	for _, runtime := range runtimes {
		fmt.Printf("%s %s %v\n", runtime.Language, runtime.Version, runtime.Aliases)
	}
}

// Package management operates on an instance's own runtime store, so it is
// only available when self-hosting.
func ExampleClient_InstallPackage() {
	client := gopiston.NewClient("http://localhost:2000/api/v2/")

	// Installing downloads and unpacks a runtime, which can take minutes.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	installation, err := client.InstallPackage(ctx, "python", "3.10.0")
	if err != nil {
		// A client targeting the official API fails here without making a
		// request.
		if errors.Is(err, gopiston.ErrUnsupportedByOfficialAPI) {
			log.Fatal("package management requires a self-hosted instance")
		}
		log.Fatal(err)
	}

	fmt.Println("installed", installation.Language, installation.Version)
}

// A failed request is reported as an *APIError, and can be classified with
// errors.Is against the package's sentinel errors.
func ExampleAPIError() {
	client := gopiston.NewClient(gopiston.OfficialAPIBaseURL)

	_, err := client.Execute(context.Background(), "python", "",
		[]gopiston.File{{Content: "print('hello')"}})
	if err == nil {
		return
	}

	switch {
	case errors.Is(err, gopiston.ErrAPIKeyRequired):
		// Check this before ErrUnauthorized, which it implies.
		fmt.Println("this instance needs a key; pass gopiston.WithAPIKey")
	case errors.Is(err, gopiston.ErrUnauthorized):
		fmt.Println("the configured key was rejected")
	case errors.Is(err, gopiston.ErrRateLimited):
		fmt.Println("rate limited; back off and retry")
	}

	// APIError carries the status code, the instance's own message, and the
	// raw body.
	var apiErr *gopiston.APIError
	if errors.As(err, &apiErr) {
		fmt.Printf("status=%d message=%q\n", apiErr.StatusCode, apiErr.Message)
	}
}

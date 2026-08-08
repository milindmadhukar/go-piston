package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	piston "github.com/milindmadhukar/go-piston/v2"
)

func main() {
	// Package operations are only available on a self-hosted instance.
	client := piston.NewClient("http://localhost:2000/api/v2/")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	// The operations endpoints are an addition to the Piston API, so ask
	// before using them. An instance without them still installs packages,
	// just synchronously, via InstallPackage.
	supported, err := client.SupportsOperations(ctx)
	if err != nil {
		log.Fatal(err)
	}
	if !supported {
		fmt.Println("This instance does not serve the operations API; use InstallPackage instead.")
		return
	}

	// Listing is read-only, and is how a client rebuilds its view of what is
	// in flight after it restarts.
	operations, err := client.GetOperations(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%d operation(s) the instance still remembers, newest first:\n", len(operations))
	for _, operation := range operations {
		fmt.Printf("  %s %s %s — %s\n", operation.Kind, operation.Language, operation.Version, operation.State)
	}

	// Installing mutates the instance and can take a long time, so it happens
	// only when asked for by name. Everything above runs unconditionally.
	language, version, ok := target()
	if !ok {
		fmt.Println("\nSet PISTON_EXAMPLE_PACKAGE=<language>=<version> to install one in the background and stream its log.")
		return
	}

	// Starting returns as soon as the instance has accepted the work. This is
	// the point of the operations API: a package compiled from source can take
	// an hour, which no HTTP request should be held open for.
	operation, err := client.InstallPackageAsync(ctx, language, version)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("\nstarted %s of %s %s (id %s)\n",
		operation.Kind, operation.Language, operation.Version, operation.ID)

	// Following the log is also how to wait: the instance closes the socket
	// once the operation settles. Everything logged before this connected is
	// replayed first, so nothing is missed by attaching late.
	if err := followLog(ctx, client, operation.ID); err != nil {
		log.Fatal(err)
	}

	// The polled view is the same record the stream reported on.
	final, err := client.GetOperation(ctx, operation.ID)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s in %s\n", final.State, final.FinishedAt().Sub(final.StartedAt()).Round(time.Millisecond))

	// Leave the instance as it was found.
	cleanup, err := client.UninstallPackageAsync(ctx, language, version)
	if err != nil {
		log.Fatal(err)
	}
	if err := followLog(ctx, client, cleanup.ID); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("uninstalled %s %s\n", language, version)
}

// followLog prints an operation's output as it is produced and returns once
// the operation has settled.
func followLog(ctx context.Context, client *piston.Client, id string) error {
	session, err := client.ConnectOperation(ctx, id)
	if err != nil {
		return err
	}
	defer session.Close()

	for {
		event, err := session.Next(ctx)
		if errors.Is(err, io.EOF) {
			// The instance closed the socket, which it does only once the
			// operation is over.
			return nil
		}
		if err != nil {
			return err
		}

		switch event.Type {
		case piston.OperationEventLog:
			// Data is not necessarily one line: the replayed backlog arrives
			// as a single event.
			fmt.Printf("  | %s\n", event.Data)
		case piston.OperationEventState:
			if event.State == piston.OperationFailed {
				return fmt.Errorf("operation failed: %s", event.Error)
			}
		}
	}
}

// target reads the package to install from PISTON_EXAMPLE_PACKAGE, as
// "language=version".
func target() (language, version string, ok bool) {
	language, version, found := strings.Cut(os.Getenv("PISTON_EXAMPLE_PACKAGE"), "=")
	if !found || language == "" || version == "" {
		return "", "", false
	}
	return language, version, true
}

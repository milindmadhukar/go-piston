package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"strings"
	"time"

	piston "github.com/milindmadhukar/go-piston/v2"
)

// The program reads three lines and echoes each one as it arrives.
//
// flush=True matters: Python block buffers stdout when it is not attached to a
// terminal, so without it nothing would arrive until the process exited, which
// would defeat the point of an interactive session.
const source = `import sys
for _ in range(3):
    line = sys.stdin.readline()
    print("echo:", line.strip(), flush=True)
`

func main() {
	// Interactive execution is only available on a self-hosted instance.
	client := piston.NewClient("http://localhost:2000/api/v2/")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	session, err := client.Connect(ctx, "python", "",
		[]piston.File{{Name: "main.py", Content: source}})
	if err != nil {
		log.Fatal(err)
	}
	defer session.Close()

	start := time.Now()
	sent := 0

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
		case piston.EventRuntime:
			fmt.Printf("runtime: %s %s\n", event.Language, event.Version)

		case piston.EventStage:
			fmt.Printf("stage:   %s\n", event.Stage)

			// The process is running now, so send it the first line.
			if event.Stage == "run" {
				sent++
				if err := session.SendStdin(ctx, fmt.Sprintf("line %d\n", sent)); err != nil {
					log.Fatal(err)
				}
			}

		case piston.EventStdout:
			// Data is a chunk rather than a line, so it carries whatever
			// newlines the process wrote. Trim them only for display.
			// The elapsed time shows output arriving while the process is
			// still running, rather than all at once when it exits.
			fmt.Printf("stdout:  %-14s (after %s)\n",
				strings.TrimRight(event.Data, "\n"), time.Since(start).Round(time.Millisecond))

			// Feed the next line in response to the previous echo.
			if sent < 3 {
				sent++
				if err := session.SendStdin(ctx, fmt.Sprintf("line %d\n", sent)); err != nil {
					log.Fatal(err)
				}
			}

		case piston.EventStderr:
			fmt.Printf("stderr:  %s", event.Data)

		case piston.EventExit:
			if event.Signal != "" {
				fmt.Printf("exit:    %s killed by %s\n", event.Stage, event.Signal)
			} else {
				fmt.Printf("exit:    %s exited %d\n", event.Stage, *event.Code)
			}

		case piston.EventError:
			log.Fatal("piston reported: ", event.Message)
		}
	}
}

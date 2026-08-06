package gopiston

import (
	"errors"
	"io"
	"strings"
	"testing"
)

// A live interactive job must stream output as it is produced and accept input
// while the process is still running.
func TestInteractiveSession(t *testing.T) {
	requireInteractiveAccess(t)
	requireLanguage(t, "python")

	// Echoes each line back as it arrives. flush=True matters: Python block
	// buffers stdout when it is not attached to a terminal, so without it
	// nothing would arrive until the process exited — which would defeat the
	// point of the endpoint and make this test pass for the wrong reason.
	source := `import sys
for line in sys.stdin:
    print("echo:", line.strip(), flush=True)
`

	session, err := client.Connect(testContext(t), "python", "",
		[]File{{Name: "main.py", Content: source}})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()

	ctx := testContext(t)

	// The runtime event always comes first.
	event, err := session.Next(ctx)
	if err != nil {
		t.Fatal(err)
	}
	assert(event.Type, EventRuntime, t)
	assert(event.Language, "python", t)
	if event.Version == "" {
		t.Error("Expected a version on the runtime event")
	}

	// Then the run stage starts.
	event, err = session.Next(ctx)
	if err != nil {
		t.Fatal(err)
	}
	assert(event.Type, EventStage, t)
	assert(event.Stage, "run", t)

	// Write once the process is running, and read the echo back. Receiving it
	// before the job has ended is what proves this is streaming rather than
	// buffered.
	if err := session.SendStdin(ctx, "first\n"); err != nil {
		t.Fatal(err)
	}

	var output strings.Builder
	for {
		event, err = session.Next(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if event.Type == EventStdout {
			output.WriteString(event.Data)
		}
		if strings.Contains(output.String(), "echo: first") {
			break
		}
	}

	// A second write on the same session must also be delivered.
	if err := session.SendStdin(ctx, "second\n"); err != nil {
		t.Fatal(err)
	}
	for !strings.Contains(output.String(), "echo: second") {
		event, err = session.Next(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if event.Type == EventStdout {
			output.WriteString(event.Data)
		}
	}
}

// A job that ends on its own must report the run stage's exit and then close
// with io.EOF rather than an error.
func TestInteractiveSessionCompletes(t *testing.T) {
	requireInteractiveAccess(t)
	requireLanguage(t, "python")

	session, err := client.Connect(testContext(t), "python", "",
		[]File{{Content: "print('done')"}})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()

	ctx := testContext(t)

	var sawExit bool
	var stdout strings.Builder

	for {
		event, err := session.Next(ctx)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatal(err)
		}

		switch event.Type {
		case EventStdout:
			stdout.WriteString(event.Data)
		case EventExit:
			if event.Stage == "run" {
				sawExit = true
				if event.Code == nil || *event.Code != 0 {
					t.Errorf("Expected the run stage to exit 0, got code=%v signal=%q", event.Code, event.Signal)
				}
			}
		}
	}

	if !sawExit {
		t.Error("Expected an exit event for the run stage")
	}
	if !strings.Contains(stdout.String(), "done") {
		t.Errorf("Expected the program's output, got %q", stdout.String())
	}
}

// An unknown runtime is rejected after the connection is established, so it
// surfaces from Next rather than from Connect.
func TestInteractiveUnknownRuntime(t *testing.T) {
	requireInteractiveAccess(t)

	session, err := client.Connect(testContext(t), "not-a-real-language", "1.0.0",
		[]File{{Content: "x"}})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()

	ctx := testContext(t)

	event, err := session.Next(ctx)
	if err != nil {
		// Some instances close without sending an error event first.
		return
	}
	assert(event.Type, EventError, t)
	if event.Message == "" {
		t.Error("Expected the error event to explain the failure")
	}

	if _, err := session.Next(ctx); err == nil {
		t.Error("Expected the session to end after an error event")
	}
}

// The instance validates and accepts a signal packet.
//
// This deliberately does not assert that the process dies: Piston's router
// emits an event named "signal" while its job handler listens for one named
// "kill", so signals are currently dropped. Asserting termination would encode
// that bug as the expected behavior. What matters here is that a valid signal
// is not rejected with ErrInvalidSignal.
func TestInteractiveSignalAccepted(t *testing.T) {
	requireInteractiveAccess(t)
	requireLanguage(t, "python")

	session, err := client.Connect(testContext(t), "python", "",
		[]File{{Content: "import time\nprint('up', flush=True)\ntime.sleep(30)"}})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()

	ctx := testContext(t)

	// Wait until the process is actually running before signalling it.
	for {
		event, err := session.Next(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if event.Type == EventStdout && strings.Contains(event.Data, "up") {
			break
		}
	}

	if err := session.SendSignal(ctx, "SIGKILL"); err != nil {
		t.Fatalf("Expected a valid signal to be accepted, got %v", err)
	}
}

// An unrecognized signal name ends the session with ErrInvalidSignal.
func TestInteractiveInvalidSignal(t *testing.T) {
	requireInteractiveAccess(t)
	requireLanguage(t, "python")

	session, err := client.Connect(testContext(t), "python", "",
		[]File{{Content: "import time\nprint('up', flush=True)\ntime.sleep(2)"}})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()

	ctx := testContext(t)

	// Wait for the process to be running first. The instance creates the job
	// asynchronously while handling init, and rejects anything arriving
	// before it exists with ErrNotInitialized rather than validating it.
	for {
		event, err := session.Next(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if event.Type == EventStdout && strings.Contains(event.Data, "up") {
			break
		}
	}

	if err := session.SendSignal(ctx, "SIGNOTAREALSIGNAL"); err != nil {
		t.Fatal(err)
	}

	for {
		_, err := session.Next(ctx)
		if err == nil {
			continue
		}
		if !errors.Is(err, ErrInvalidSignal) {
			t.Errorf("Expected ErrInvalidSignal, got %v", err)
		}
		return
	}
}

package gopiston

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/coder/websocket"
)

// fakeInstance stands up a WebSocket endpoint that behaves like Piston's
// /connect, driven by the given handler. It returns a client pointed at it.
func fakeInstance(t *testing.T, handle func(ctx context.Context, conn *websocket.Conn)) *Client {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer conn.CloseNow()
		handle(r.Context(), conn)
	}))
	t.Cleanup(server.Close)

	return NewClient(server.URL, WithHTTPClient(server.Client()))
}

// writeEvent sends one server-to-client message.
func writeEvent(ctx context.Context, conn *websocket.Conn, msg map[string]any) {
	data, _ := json.Marshal(msg)
	_ = conn.Write(ctx, websocket.MessageText, data)
}

// readMessage reads one client-to-server message.
func readMessage(ctx context.Context, conn *websocket.Conn) map[string]any {
	_, data, err := conn.Read(ctx)
	if err != nil {
		return nil
	}
	var msg map[string]any
	_ = json.Unmarshal(data, &msg)
	return msg
}

// The init frame must carry the same body an /execute request would, with the
// params applied.
func TestConnectSendsInit(t *testing.T) {
	got := make(chan map[string]any, 1)

	client := fakeInstance(t, func(ctx context.Context, conn *websocket.Conn) {
		got <- readMessage(ctx, conn)
		conn.Close(CloseJobCompleted, "Job Completed")
	})

	session, err := client.Connect(context.Background(), "python", "3.10.0",
		[]File{{Name: "main.py", Content: "print(1)"}},
		Args([]string{"one"}),
		RunMemoryLimit(1024),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()

	msg := <-got
	assert(msg["type"], "init", t)
	assert(msg["language"], "python", t)
	assert(msg["version"], "3.10.0", t)
	assert(msg["run_memory_limit"], float64(1024), t)

	files, _ := msg["files"].([]any)
	if len(files) != 1 {
		t.Fatalf("Expected 1 file, got %d", len(files))
	}
	file, _ := files[0].(map[string]any)
	assert(file["name"], "main.py", t)
	assert(file["content"], "print(1)", t)

	args, _ := msg["args"].([]any)
	if len(args) != 1 || args[0] != "one" {
		t.Errorf("Expected args [one], got %v", args)
	}
}

// An empty version must become the "*" selector, as with Execute.
func TestConnectSendsLatestSelector(t *testing.T) {
	got := make(chan map[string]any, 1)

	client := fakeInstance(t, func(ctx context.Context, conn *websocket.Conn) {
		got <- readMessage(ctx, conn)
		conn.Close(CloseJobCompleted, "Job Completed")
	})

	session, err := client.Connect(context.Background(), "python", "", []File{{Content: "print(1)"}})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()

	assert((<-got)["version"], latestVersionSelector, t)
}

// Every server message must decode to the right event, and a normally
// completed job must end with io.EOF rather than an error.
func TestSessionEventSequence(t *testing.T) {
	client := fakeInstance(t, func(ctx context.Context, conn *websocket.Conn) {
		readMessage(ctx, conn) // init
		writeEvent(ctx, conn, map[string]any{"type": "runtime", "language": "python", "version": "3.10.0"})
		writeEvent(ctx, conn, map[string]any{"type": "stage", "stage": "run"})
		writeEvent(ctx, conn, map[string]any{"type": "data", "stream": "stdout", "data": "out"})
		writeEvent(ctx, conn, map[string]any{"type": "data", "stream": "stderr", "data": "err"})
		writeEvent(ctx, conn, map[string]any{"type": "exit", "stage": "run", "code": 0, "signal": nil})
		conn.Close(CloseJobCompleted, "Job Completed")
	})

	session, err := client.Connect(context.Background(), "python", "", []File{{Content: "x"}})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()

	ctx := context.Background()

	event := mustNext(t, session, ctx)
	assert(event.Type, EventRuntime, t)
	assert(event.Language, "python", t)
	assert(event.Version, "3.10.0", t)

	event = mustNext(t, session, ctx)
	assert(event.Type, EventStage, t)
	assert(event.Stage, "run", t)

	event = mustNext(t, session, ctx)
	assert(event.Type, EventStdout, t)
	assert(event.Data, "out", t)

	event = mustNext(t, session, ctx)
	assert(event.Type, EventStderr, t)
	assert(event.Data, "err", t)

	event = mustNext(t, session, ctx)
	assert(event.Type, EventExit, t)
	assert(event.Stage, "run", t)
	if event.Code == nil || *event.Code != 0 {
		t.Errorf("Expected exit code 0, got %v", event.Code)
	}

	if _, err := session.Next(ctx); !errors.Is(err, io.EOF) {
		t.Errorf("Expected io.EOF on normal completion, got %v", err)
	}
}

// The runtime frame is the only place an interactive session learns which
// engine it got, and the field is absent whenever the instance has nothing to
// disambiguate — or predates the field entirely.
func TestSessionRuntimeEventCarriesEngine(t *testing.T) {
	for _, tc := range []struct {
		name  string
		frame map[string]any
		want  string
	}{
		{
			name:  "engine named",
			frame: map[string]any{"type": "runtime", "language": "javascript", "version": "1.3.14", "runtime": "bun"},
			want:  "bun",
		},
		{
			name:  "engine omitted",
			frame: map[string]any{"type": "runtime", "language": "bash", "version": "5.2.0"},
			want:  "",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := fakeInstance(t, func(ctx context.Context, conn *websocket.Conn) {
				readMessage(ctx, conn)
				writeEvent(ctx, conn, tc.frame)
				conn.Close(CloseJobCompleted, "Job Completed")
			})

			session, err := client.Connect(context.Background(), "x", "", []File{{Content: "x"}})
			if err != nil {
				t.Fatal(err)
			}
			defer session.Close()

			event := mustNext(t, session, context.Background())
			assert(event.Type, EventRuntime, t)
			assert(event.Runtime, tc.want, t)
		})
	}
}

// A stage ended by a signal reports a null code, which must stay
// distinguishable from a genuine exit code of zero.
func TestSessionExitBySignal(t *testing.T) {
	client := fakeInstance(t, func(ctx context.Context, conn *websocket.Conn) {
		readMessage(ctx, conn)
		writeEvent(ctx, conn, map[string]any{"type": "exit", "stage": "run", "code": nil, "signal": "SIGKILL"})
		conn.Close(CloseJobCompleted, "Job Completed")
	})

	session, err := client.Connect(context.Background(), "bash", "", []File{{Content: "sleep 10"}})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()

	event := mustNext(t, session, context.Background())
	assert(event.Type, EventExit, t)
	assert(event.Signal, "SIGKILL", t)
	if event.Code != nil {
		t.Errorf("Expected a nil code when a signal ended the stage, got %d", *event.Code)
	}
}

func TestSessionCloseCodes(t *testing.T) {
	for _, tc := range []struct {
		name string
		code websocket.StatusCode
		want error
	}{
		{"already initialized", CloseAlreadyInitialized, ErrAlreadyInitialized},
		{"initialization timeout", CloseInitializationTimeout, ErrInitializationTimeout},
		{"not initialized", CloseNotInitialized, ErrNotInitialized},
		{"stdin only", CloseStdinOnly, ErrStdinOnly},
		{"invalid signal", CloseInvalidSignal, ErrInvalidSignal},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := fakeInstance(t, func(ctx context.Context, conn *websocket.Conn) {
				readMessage(ctx, conn)
				conn.Close(tc.code, "")
			})

			session, err := client.Connect(context.Background(), "python", "", []File{{Content: "x"}})
			if err != nil {
				t.Fatal(err)
			}
			defer session.Close()

			if _, err := session.Next(context.Background()); !errors.Is(err, tc.want) {
				t.Errorf("Expected %v, got %v", tc.want, err)
			}
		})
	}
}

// The instance sends an error event and then closes 4002, so the close should
// report what the error event said rather than a bare "job failed".
func TestSessionNotifiedErrorCarriesMessage(t *testing.T) {
	client := fakeInstance(t, func(ctx context.Context, conn *websocket.Conn) {
		readMessage(ctx, conn)
		writeEvent(ctx, conn, map[string]any{"type": "error", "message": "python-9.9.9 runtime is unknown"})
		conn.Close(CloseNotifiedError, "Notified Error")
	})

	session, err := client.Connect(context.Background(), "python", "9.9.9", []File{{Content: "x"}})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()

	ctx := context.Background()

	event := mustNext(t, session, ctx)
	assert(event.Type, EventError, t)
	assert(event.Message, "python-9.9.9 runtime is unknown", t)

	_, err = session.Next(ctx)
	if err == nil {
		t.Fatal("Expected an error after the close")
	}
	if !strings.Contains(err.Error(), "python-9.9.9 runtime is unknown") {
		t.Errorf("Expected the close to carry the error message, got %v", err)
	}
}

func TestSessionSendStdinAndSignal(t *testing.T) {
	got := make(chan map[string]any, 2)

	client := fakeInstance(t, func(ctx context.Context, conn *websocket.Conn) {
		readMessage(ctx, conn) // init
		got <- readMessage(ctx, conn)
		got <- readMessage(ctx, conn)
		conn.Close(CloseJobCompleted, "Job Completed")
	})

	session, err := client.Connect(context.Background(), "python", "", []File{{Content: "x"}})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()

	ctx := context.Background()
	if err := session.SendStdin(ctx, "hello\n"); err != nil {
		t.Fatal(err)
	}
	if err := session.SendSignal(ctx, "SIGKILL"); err != nil {
		t.Fatal(err)
	}

	stdin := <-got
	assert(stdin["type"], "data", t)
	assert(stdin["stream"], "stdin", t)
	assert(stdin["data"], "hello\n", t)

	signal := <-got
	assert(signal["type"], "signal", t)
	assert(signal["signal"], "SIGKILL", t)
}

// Interactive execution is unavailable on the official API, and must fail
// before any connection is attempted.
func TestConnectGuardedOnOfficialAPI(t *testing.T) {
	var connections int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		connections++
	}))
	defer server.Close()

	client := NewClient(OfficialAPIBaseURL, WithHTTPClient(server.Client()))
	client.baseURL = server.URL + "/"

	session, err := client.Connect(context.Background(), "python", "", []File{{Content: "x"}})
	if !errors.Is(err, ErrUnsupportedByOfficialAPI) {
		t.Errorf("Expected ErrUnsupportedByOfficialAPI, got %v", err)
	}
	if session != nil {
		t.Error("Expected no session")
	}
	if connections != 0 {
		t.Errorf("Expected the guard to make no connections, got %d", connections)
	}
}

// The library's default read limit is 32 KiB, which a single chunk of process
// output can exceed.
func TestSessionAcceptsLargeMessages(t *testing.T) {
	large := make([]byte, 100*1024)
	for i := range large {
		large[i] = 'a'
	}

	client := fakeInstance(t, func(ctx context.Context, conn *websocket.Conn) {
		readMessage(ctx, conn)
		writeEvent(ctx, conn, map[string]any{"type": "data", "stream": "stdout", "data": string(large)})
		conn.Close(CloseJobCompleted, "Job Completed")
	})

	session, err := client.Connect(context.Background(), "python", "", []File{{Content: "x"}})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()

	event := mustNext(t, session, context.Background())
	assert(event.Type, EventStdout, t)
	assert(len(event.Data), len(large), t)
}

func mustNext(t *testing.T, session *Session, ctx context.Context) Event {
	t.Helper()

	event, err := session.Next(ctx)
	if err != nil {
		t.Fatalf("Next: %v", err)
	}
	return event
}

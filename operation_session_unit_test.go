package gopiston

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/coder/websocket"
)

// fakeOperationInstance stands up a WebSocket endpoint that behaves like an
// operation's log socket, driven by the given handler.
//
// It is deliberately not fakeInstance: that one speaks the interactive job
// protocol, and the two endpoints share neither their messages nor their close
// codes.
func fakeOperationInstance(t *testing.T, handle func(ctx context.Context, conn *websocket.Conn)) *Client {
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

// A session replays the backlog, streams the rest, then reports the settled
// state and ends with io.EOF on the standard close code.
func TestOperationSessionStreamsLogThenState(t *testing.T) {
	client := fakeOperationInstance(t, func(ctx context.Context, conn *websocket.Conn) {
		// Everything logged before the client attached arrives as one message.
		writeEvent(ctx, conn, map[string]any{"type": "log", "data": "Installing go-1.26.5\nFetching"})
		writeEvent(ctx, conn, map[string]any{"type": "log", "data": "Extracting into ."})
		writeEvent(ctx, conn, map[string]any{"type": "state", "state": "succeeded"})
		conn.Close(websocket.StatusNormalClosure, "Operation finished")
	})

	session, err := client.ConnectOperation(context.Background(), "abc")
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()

	ctx := context.Background()

	first, err := session.Next(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if first.Type != OperationEventLog || first.Data != "Installing go-1.26.5\nFetching" {
		t.Errorf("first event = %+v, want the replayed backlog as one log event", first)
	}

	second, err := session.Next(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if second.Type != OperationEventLog || second.Data != "Extracting into ." {
		t.Errorf("second event = %+v, want a log event", second)
	}

	settled, err := session.Next(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if settled.Type != OperationEventState || settled.State != OperationSucceeded {
		t.Errorf("third event = %+v, want state succeeded", settled)
	}

	// The operation endpoint ends with the standard close code, not the
	// interactive endpoint's 4999.
	if _, err := session.Next(ctx); !errors.Is(err, io.EOF) {
		t.Errorf("err after close = %v, want io.EOF", err)
	}
}

// A failure carries its reason on the state event.
func TestOperationSessionReportsFailure(t *testing.T) {
	client := fakeOperationInstance(t, func(ctx context.Context, conn *websocket.Conn) {
		writeEvent(ctx, conn, map[string]any{
			"type":  "state",
			"state": "failed",
			"error": "sha256 mismatch",
		})
		conn.Close(websocket.StatusNormalClosure, "Operation finished")
	})

	session, err := client.ConnectOperation(context.Background(), "abc")
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()

	event, err := session.Next(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if event.Type != OperationEventState || event.State != OperationFailed || event.Error != "sha256 mismatch" {
		t.Errorf("event = %+v, want a failed state carrying the reason", event)
	}
}

// 4999 is the interactive endpoint's normal completion, and means nothing
// here. Treating it as a clean end would hide a genuinely broken connection.
func TestOperationSessionDoesNotAcceptTheInteractiveCloseCode(t *testing.T) {
	client := fakeOperationInstance(t, func(ctx context.Context, conn *websocket.Conn) {
		conn.Close(CloseJobCompleted, "Job Completed")
	})

	session, err := client.ConnectOperation(context.Background(), "abc")
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()

	if _, err := session.Next(context.Background()); errors.Is(err, io.EOF) {
		t.Error("close code 4999 was accepted as a normal end on an operation socket")
	}
}

func TestOperationSessionRejectsAnUnknownEventType(t *testing.T) {
	client := fakeOperationInstance(t, func(ctx context.Context, conn *websocket.Conn) {
		writeEvent(ctx, conn, map[string]any{"type": "exit", "code": 0})
		conn.Close(websocket.StatusNormalClosure, "")
	})

	session, err := client.ConnectOperation(context.Background(), "abc")
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()

	_, err = session.Next(context.Background())
	if err == nil || errors.Is(err, io.EOF) {
		t.Fatalf("err = %v, want a decode failure for an interactive-only event type", err)
	}
}

// An unknown id is refused before the upgrade, as an ordinary HTTP response.
// That has to surface as the same *APIError every other method produces.
func TestConnectOperationReportsAnUnknownID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message":"Unknown operation nope"}`))
	}))
	t.Cleanup(server.Close)

	client := NewClient(server.URL, WithHTTPClient(server.Client()))

	_, err := client.ConnectOperation(context.Background(), "nope")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.Message != "Unknown operation nope" {
		t.Errorf("APIError = %+v, want the instance's message", apiErr)
	}
}

// The socket is read-only, so a client that finishes early must not disturb
// the operation. Closing is safe at any point.
func TestOperationSessionCloseIsSafeMidStream(t *testing.T) {
	client := fakeOperationInstance(t, func(ctx context.Context, conn *websocket.Conn) {
		writeEvent(ctx, conn, map[string]any{"type": "log", "data": "line"})
		// The instance ignores what the client sends but still reads, which is
		// what answers the closing handshake. A double that stopped reading
		// would make Close block until its own deadline.
		for {
			if _, _, err := conn.Read(ctx); err != nil {
				return
			}
		}
	})

	session, err := client.ConnectOperation(context.Background(), "abc")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := session.Next(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := session.Close(); err != nil {
		t.Errorf("Close() = %v, want nil", err)
	}
}

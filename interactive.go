package gopiston

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/coder/websocket"
)

// WebSocket close codes used by Piston's interactive endpoint. CloseJobCompleted
// is absent from the published API reference but is the normal completion path,
// so it must not be treated as a failure — Next reports it as io.EOF.
//
// Session translates all of these into errors, so using them directly is only
// necessary when speaking the protocol by hand, as a test double does.
const (
	CloseJobCompleted          = 4999
	CloseAlreadyInitialized    = 4000
	CloseInitializationTimeout = 4001
	CloseNotifiedError         = 4002
	CloseNotInitialized        = 4003
	CloseStdinOnly             = 4004
	CloseInvalidSignal         = 4005
)

// Session is a live interactive execution job. It streams the process's output
// as it is produced and accepts input while the process is still running,
// which the one-shot Execute cannot do.
//
// Create one with Client.Connect and always Close it.
//
// A Session may be read from one goroutine while another writes to it — that
// is the point of an interactive job — but Next must not be called
// concurrently with itself, nor SendStdin with SendSignal.
type Session struct {
	conn *websocket.Conn

	// lastError holds the message from the most recent EventError, so the
	// close that immediately follows it can report why.
	lastError string
}

// wsMessage is the wire form of every message in either direction.
type wsMessage struct {
	Type   string `json:"type"`
	Stream string `json:"stream,omitempty"`
	Data   string `json:"data,omitempty"`
	Signal string `json:"signal,omitempty"`

	// Server to client only.
	Language string `json:"language,omitempty"`
	Version  string `json:"version,omitempty"`
	Runtime  string `json:"runtime,omitempty"`
	Stage    string `json:"stage,omitempty"`
	Code     *int   `json:"code,omitempty"`
	Message  string `json:"message,omitempty"`
}

// initMessage is an execute request plus the "init" discriminator.
type initMessage struct {
	Type string `json:"type"`
	RequestBody
}

// Connect opens an interactive execution job on the instance and returns a
// Session streaming its progress.
//
// The arguments match Execute: language and version name a runtime, an empty
// version selects the highest installed one, and files is the job's source
// with the first file as the entry point. The Stdin param has no effect here —
// the instance discards it for interactive jobs; use Session.SendStdin
// instead.
//
// Connect returns once the job has been requested; it does not wait for the
// instance to accept it. A rejected job — an unknown language, say — therefore
// surfaces from the first call to Next rather than from Connect.
//
// Interactive execution is not available on the official Piston API, so a
// client targeting it fails with an error matching
// ErrUnsupportedByOfficialAPI without connecting.
func (client *Client) Connect(ctx context.Context, language string, version string, files []File, params ...Param) (*Session, error) {
	if client.officialAPI {
		return nil, fmt.Errorf("piston: connect: %w", ErrUnsupportedByOfficialAPI)
	}

	if version == "" {
		version = latestVersionSelector
	}

	reqBody := RequestBody{
		Language: language,
		Version:  version,
		Files:    files,
	}
	processParams(&reqBody, params...)

	header := http.Header{}
	if client.apiKey != "" {
		header.Set("Authorization", client.apiKey)
	}

	conn, _, err := websocket.Dial(ctx, client.endpoint("connect"), &websocket.DialOptions{
		HTTPClient: client.httpClient,
		HTTPHeader: header,
	})
	if err != nil {
		return nil, fmt.Errorf("piston: connect: %w", err)
	}

	// The library defaults to a 32 KiB limit per message, which a single
	// chunk of process output can exceed.
	conn.SetReadLimit(maxResponseBytes)

	session := &Session{conn: conn}

	// The instance closes the connection if init does not arrive within one
	// second of the handshake, so this cannot be deferred.
	if err := session.send(ctx, initMessage{Type: "init", RequestBody: reqBody}); err != nil {
		conn.Close(websocket.StatusInternalError, "init failed")
		return nil, err
	}

	return session, nil
}

// Next returns the next event from the instance, blocking until one arrives.
//
// It returns io.EOF once the job has finished and the instance has closed the
// connection. Any other error ends the session too: the close-code sentinels
// (ErrInitializationTimeout and friends), a transport failure, or ctx being
// cancelled.
func (session *Session) Next(ctx context.Context) (Event, error) {
	_, data, err := session.conn.Read(ctx)
	if err != nil {
		return Event{}, session.readError(err)
	}

	var msg wsMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return Event{}, fmt.Errorf("piston: decode event: %w", err)
	}

	event := Event{
		Language: msg.Language,
		Version:  msg.Version,
		Runtime:  msg.Runtime,
		Stage:    msg.Stage,
		Data:     msg.Data,
		Code:     msg.Code,
		Signal:   msg.Signal,
		Message:  msg.Message,
	}

	switch msg.Type {
	case "runtime":
		event.Type = EventRuntime
	case "stage":
		event.Type = EventStage
	case "exit":
		event.Type = EventExit
	case "error":
		event.Type = EventError
		// Retained so the close that follows can say what went wrong.
		session.lastError = msg.Message
	case "data":
		switch msg.Stream {
		case "stdout":
			event.Type = EventStdout
		case "stderr":
			event.Type = EventStderr
		default:
			return Event{}, fmt.Errorf("piston: unknown data stream %q", msg.Stream)
		}
	default:
		return Event{}, fmt.Errorf("piston: unknown event type %q", msg.Type)
	}

	return event, nil
}

// readError converts a read failure into either io.EOF for a normally
// completed job, a close-code sentinel, or the underlying error.
func (session *Session) readError(err error) error {
	switch websocket.CloseStatus(err) {
	case CloseJobCompleted:
		return io.EOF
	case CloseAlreadyInitialized:
		return fmt.Errorf("piston: %w", ErrAlreadyInitialized)
	case CloseInitializationTimeout:
		return fmt.Errorf("piston: %w", ErrInitializationTimeout)
	case CloseNotInitialized:
		return fmt.Errorf("piston: %w", ErrNotInitialized)
	case CloseStdinOnly:
		return fmt.Errorf("piston: %w", ErrStdinOnly)
	case CloseInvalidSignal:
		return fmt.Errorf("piston: %w", ErrInvalidSignal)
	case CloseNotifiedError:
		if session.lastError != "" {
			return fmt.Errorf("piston: %s", session.lastError)
		}
		return fmt.Errorf("piston: job failed")
	}
	return fmt.Errorf("piston: read event: %w", err)
}

// SendStdin writes data to the running process's standard input.
//
// The data is passed through as-is, so include a trailing newline when the
// process reads by line.
//
// Wait for the run stage before calling this. The instance creates the job
// asynchronously while handling the connection, and rejects anything that
// arrives before the job exists, ending the session with ErrNotInitialized.
// Waiting for an EventStage whose Stage is "run" is enough.
func (session *Session) SendStdin(ctx context.Context, data string) error {
	return session.send(ctx, wsMessage{Type: "data", Stream: "stdin", Data: data})
}

// SendSignal sends a signal, by name, to the running process — "SIGKILL",
// "SIGTERM" and so on. An unrecognized name ends the session with
// ErrInvalidSignal on the next read.
//
// As with SendStdin, wait for the run stage first: a signal arriving before
// the job exists ends the session with ErrNotInitialized instead of being
// validated.
//
// Two accepted cases do nothing, by design on the instance's side, and neither
// is reported as an error — a job that keeps running after them has not
// misbehaved:
//
//   - SIGSTOP, SIGTSTP, SIGTTIN and SIGTTOU are never delivered. A stopped job
//     would hold its concurrency slot forever.
//   - The real-time signal names, SIGRTMIN+n and SIGRTMAX-n, are accepted but
//     have no effect.
//
// Delivery is to the sandbox supervisor rather than to the program itself.
// Older Piston instances, including everything before the fix in
// engineer-man/piston's successor, validated the signal and then dropped it —
// their router emitted an event named "signal" while the job handler listened
// for one named "kill" — so against those a job cannot be terminated this way.
// Closing the session is the portable way to stop a job.
func (session *Session) SendSignal(ctx context.Context, signal string) error {
	return session.send(ctx, wsMessage{Type: "signal", Signal: signal})
}

func (session *Session) send(ctx context.Context, msg any) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("piston: encode message: %w", err)
	}
	if err := session.conn.Write(ctx, websocket.MessageText, data); err != nil {
		return fmt.Errorf("piston: send message: %w", err)
	}
	return nil
}

// Close ends the session. It is safe to call after the job has already
// finished, so it can always be deferred.
//
// Closing a session whose job is still running kills the running stage and
// releases the instance's sandbox and concurrency slot immediately, which
// makes this the portable way to stop a job early. Older instances instead
// left the job running until it hit its own wall-time limit.
func (session *Session) Close() error {
	return session.conn.Close(websocket.StatusNormalClosure, "")
}

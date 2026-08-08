package gopiston

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/coder/websocket"
)

// OperationEventType identifies what an OperationEvent carries.
type OperationEventType string

// The kinds of event an OperationSession produces.
const (
	// OperationEventLog carries one or more log lines in Data.
	OperationEventLog OperationEventType = "log"

	// OperationEventState reports that the operation has settled. It is the
	// last event of a session; the instance closes the connection immediately
	// afterwards, which the following Next reports as io.EOF.
	OperationEventState OperationEventType = "state"
)

// OperationEvent is one message from an OperationSession.
type OperationEvent struct {
	// Type is which kind of event this is. Switch on it before reading the
	// other fields.
	Type OperationEventType

	// Data holds log output, for OperationEventLog. It is not necessarily one
	// line: the backlog replayed when the session opens arrives as a single
	// event, and lines carry no trailing newline of their own.
	Data string

	// State is the settled state, for OperationEventState. It is never
	// OperationRunning.
	State OperationState

	// Error describes the failure, for an OperationEventState whose State is
	// OperationFailed.
	Error string
}

// operationWSMessage is the wire form of a message on an operation's log
// socket. It is deliberately separate from wsMessage: the operation log and
// the interactive job endpoints share neither their message types nor their
// close codes, and conflating them would let a change to one silently accept
// the other's traffic.
type operationWSMessage struct {
	Type  string `json:"type"`
	Data  string `json:"data,omitempty"`
	State string `json:"state,omitempty"`
	Error string `json:"error,omitempty"`
}

// OperationSession streams a package operation's log as it is produced.
//
// Create one with Client.ConnectOperation and always Close it. The session is
// read-only — the instance ignores anything sent to it — so unlike Session
// there is nothing to synchronize, but Next must not be called concurrently
// with itself.
type OperationSession struct {
	conn *websocket.Conn
}

// ConnectOperation opens a live stream of an operation's log.
//
// Everything the operation has already logged is replayed as the first event,
// so attaching late still shows the whole run. If the operation has already
// settled by the time the connection opens, the only event is the final
// OperationEventState.
//
// The id must name an operation the instance still remembers; an unknown one
// fails with an error matching ErrNotFound before the connection is upgraded.
// Package operations are unavailable on the official API, so a client
// targeting it fails with ErrUnsupportedByOfficialAPI without connecting.
func (client *Client) ConnectOperation(ctx context.Context, id string) (*OperationSession, error) {
	if client.officialAPI {
		return nil, fmt.Errorf("piston: connect operation: %w", ErrUnsupportedByOfficialAPI)
	}

	header := http.Header{}
	if client.apiKey != "" {
		header.Set("Authorization", client.apiKey)
	}

	conn, resp, err := websocket.Dial(ctx, client.operationEndpoint(id, "connect"), &websocket.DialOptions{
		HTTPClient: client.httpClient,
		HTTPHeader: header,
	})
	if err != nil {
		// A refused upgrade is an ordinary HTTP response — 404 for an unknown
		// id — and the dial error alone does not say which. Rebuilding the
		// *APIError from the response keeps errors.Is working here as it does
		// on every other method.
		if resp != nil {
			defer resp.Body.Close()
			body, _ := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
			return nil, newAPIError(resp.StatusCode, body, client.apiKey == "")
		}
		return nil, fmt.Errorf("piston: connect operation: %w", err)
	}

	// A replayed backlog arrives as one message and can be large.
	conn.SetReadLimit(maxResponseBytes)

	return &OperationSession{conn: conn}, nil
}

// Next returns the next event, blocking until one arrives.
//
// It returns io.EOF once the operation has settled and the instance has closed
// the connection, which always follows an OperationEventState. Any other error
// ends the session.
func (session *OperationSession) Next(ctx context.Context) (OperationEvent, error) {
	_, data, err := session.conn.Read(ctx)
	if err != nil {
		// The operation endpoint signals a normal end with the standard close
		// code, not with the interactive endpoint's 4999.
		if websocket.CloseStatus(err) == websocket.StatusNormalClosure {
			return OperationEvent{}, io.EOF
		}
		return OperationEvent{}, fmt.Errorf("piston: read operation event: %w", err)
	}

	var msg operationWSMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return OperationEvent{}, fmt.Errorf("piston: decode operation event: %w", err)
	}

	switch msg.Type {
	case "log":
		return OperationEvent{Type: OperationEventLog, Data: msg.Data}, nil
	case "state":
		return OperationEvent{
			Type:  OperationEventState,
			State: OperationState(msg.State),
			Error: msg.Error,
		}, nil
	}

	return OperationEvent{}, fmt.Errorf("piston: unknown operation event type %q", msg.Type)
}

// Close ends the session. It is safe to call after the operation has already
// finished, so it can always be deferred.
//
// Closing does not cancel the operation — it runs on the instance, not on this
// connection. Reconnect with ConnectOperation to pick the log back up.
func (session *OperationSession) Close() error {
	return session.conn.Close(websocket.StatusNormalClosure, "")
}

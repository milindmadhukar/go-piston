package gopiston

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

// fakeOperationsInstance stands up an HTTP endpoint driven by the given
// handler and returns a client pointed at it.
func fakeOperationsInstance(t *testing.T, handle http.HandlerFunc) *Client {
	t.Helper()

	server := httptest.NewServer(handle)
	t.Cleanup(server.Close)

	return NewClient(server.URL, WithHTTPClient(server.Client()))
}

// notFoundJSON answers the way an instance without the operations API does:
// its catch-all 404, in Piston's error shape.
func notFoundJSON(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotFound)
	_, _ = w.Write([]byte(`{"message":"Not Found"}`))
}

// Starting an operation is a 202, not a 200, and the body is the operation.
func TestStartOperationAcceptsAccepted(t *testing.T) {
	var gotBody map[string]any

	client := fakeOperationsInstance(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/operations" {
			t.Errorf("got %s %s, want POST /operations", r.Method, r.URL.Path)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", ct)
		}
		_ = json.NewDecoder(r.Body).Decode(&gotBody)

		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"id":"abc","kind":"install","language":"gcc",
			"version":"15.3.0","state":"running","started":1786116772596}`))
	})

	operation, err := client.InstallPackageAsync(context.Background(), "gcc", "15.x")
	if err != nil {
		t.Fatal(err)
	}

	if gotBody["kind"] != "install" || gotBody["language"] != "gcc" || gotBody["version"] != "15.x" {
		t.Errorf("request body = %v, want the requested kind, language and selector", gotBody)
	}
	if operation.ID != "abc" || operation.Kind != OperationInstall {
		t.Errorf("operation = %+v, want id abc and kind install", operation)
	}
	// The resolved version, not the selector that was asked for.
	if operation.Version != "15.3.0" {
		t.Errorf("Version = %q, want the resolved 15.3.0", operation.Version)
	}
	if operation.Done() {
		t.Error("Done() = true for a running operation")
	}
	if got := operation.StartedAt().UnixMilli(); got != 1786116772596 {
		t.Errorf("StartedAt() = %d, want 1786116772596", got)
	}
	if !operation.FinishedAt().IsZero() {
		t.Errorf("FinishedAt() = %v, want the zero time while running", operation.FinishedAt())
	}
}

// A running operation omits "finished" and "error" entirely rather than
// sending them as null, so both must survive as absent.
func TestOperationDecodesAbsentFinishedAndError(t *testing.T) {
	client := fakeOperationsInstance(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"id":"abc","kind":"uninstall","language":"bash",
			"version":"5.2.0","state":"failed","started":1000,"finished":2500,
			"error":"bash-5.2.0 is not installed"}`))
	})

	operation, err := client.GetOperation(context.Background(), "abc")
	if err != nil {
		t.Fatal(err)
	}

	if operation.State != OperationFailed || !operation.Done() {
		t.Errorf("State = %q, Done = %v; want failed and done", operation.State, operation.Done())
	}
	if operation.Error != "bash-5.2.0 is not installed" {
		t.Errorf("Error = %q, want the instance's message", operation.Error)
	}
	if operation.Finished == nil || *operation.Finished != 2500 {
		t.Errorf("Finished = %v, want 2500", operation.Finished)
	}
	if got := operation.FinishedAt().UnixMilli(); got != 2500 {
		t.Errorf("FinishedAt() = %d, want 2500", got)
	}
}

// One operation per package at a time; a second is a 409.
func TestStartOperationReportsConflict(t *testing.T) {
	client := fakeOperationsInstance(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"message":"install of gcc-15.3.0 is already running"}`))
	})

	_, err := client.InstallPackageAsync(context.Background(), "gcc", "15.3.0")
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("err = %v, want ErrConflict", err)
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.Message != "install of gcc-15.3.0 is already running" {
		t.Errorf("APIError message = %+v, want the instance's explanation", apiErr)
	}
}

// An instance without the endpoints 404s the whole namespace. That must be
// distinguishable from an unknown package, which is also a 404.
func TestOperationsUnsupportedInstance(t *testing.T) {
	client := fakeOperationsInstance(t, func(w http.ResponseWriter, r *http.Request) {
		notFoundJSON(w)
	})

	_, err := client.GetOperations(context.Background())
	if !errors.Is(err, ErrOperationsUnsupported) {
		t.Fatalf("GetOperations err = %v, want ErrOperationsUnsupported", err)
	}
	// Callers that only classify by status must keep working.
	if !errors.Is(err, ErrNotFound) {
		t.Error("ErrOperationsUnsupported no longer implies ErrNotFound")
	}

	_, err = client.InstallPackageAsync(context.Background(), "gcc", "15.3.0")
	if !errors.Is(err, ErrOperationsUnsupported) {
		t.Fatalf("StartOperation err = %v, want ErrOperationsUnsupported", err)
	}

	supported, err := client.SupportsOperations(context.Background())
	if err != nil || supported {
		t.Errorf("SupportsOperations = %v, %v; want false, nil", supported, err)
	}
}

// A 404 from /operations/<id> means that id, not a missing API — the two are
// the same status and must not collapse into one meaning.
func TestUnknownOperationIsNotUnsupported(t *testing.T) {
	client := fakeOperationsInstance(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message":"Unknown operation nope"}`))
	})

	_, err := client.GetOperation(context.Background(), "nope")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
	if errors.Is(err, ErrOperationsUnsupported) {
		t.Error("an unknown operation id was reported as the API being unsupported")
	}
}

func TestSupportsOperationsOnAnInstanceThatHasThem(t *testing.T) {
	client := fakeOperationsInstance(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[]`))
	})

	supported, err := client.SupportsOperations(context.Background())
	if err != nil || !supported {
		t.Errorf("SupportsOperations = %v, %v; want true, nil", supported, err)
	}
}

// The log is text/plain, so it is returned verbatim rather than decoded.
func TestGetOperationLogIsPlainText(t *testing.T) {
	const log = "Installing go-1.26.5\nFetched 66879095 bytes, sha256 ok"

	client := fakeOperationsInstance(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/operations/abc/log" {
			t.Errorf("path = %q, want /operations/abc/log", r.URL.Path)
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte(log))
	})

	got, err := client.GetOperationLog(context.Background(), "abc")
	if err != nil {
		t.Fatal(err)
	}
	if got != log {
		t.Errorf("log = %q, want %q", got, log)
	}
}

// A kind that is neither install nor uninstall must not reach the instance,
// which reads anything other than "uninstall" as an install.
func TestStartOperationRejectsAnUnknownKind(t *testing.T) {
	client := fakeOperationsInstance(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("a request was sent for an unknown kind")
	})

	if _, err := client.StartOperation(context.Background(), "reinstall", "gcc", "15.3.0"); err == nil {
		t.Fatal("StartOperation accepted an unknown kind")
	}
}

// Package management needs a self-hosted instance, and that is decided without
// making a request.
func TestOperationsGuardedOnOfficialAPI(t *testing.T) {
	client := NewClient(OfficialAPIBaseURL, WithAPIKey("key"))
	ctx := context.Background()

	if _, err := client.InstallPackageAsync(ctx, "gcc", "15.3.0"); !errors.Is(err, ErrUnsupportedByOfficialAPI) {
		t.Errorf("InstallPackageAsync err = %v, want ErrUnsupportedByOfficialAPI", err)
	}
	if _, err := client.UninstallPackageAsync(ctx, "gcc", "15.3.0"); !errors.Is(err, ErrUnsupportedByOfficialAPI) {
		t.Errorf("UninstallPackageAsync err = %v, want ErrUnsupportedByOfficialAPI", err)
	}
	if _, err := client.GetOperations(ctx); !errors.Is(err, ErrUnsupportedByOfficialAPI) {
		t.Errorf("GetOperations err = %v, want ErrUnsupportedByOfficialAPI", err)
	}
	if _, err := client.GetOperation(ctx, "abc"); !errors.Is(err, ErrUnsupportedByOfficialAPI) {
		t.Errorf("GetOperation err = %v, want ErrUnsupportedByOfficialAPI", err)
	}
	if _, err := client.GetOperationLog(ctx, "abc"); !errors.Is(err, ErrUnsupportedByOfficialAPI) {
		t.Errorf("GetOperationLog err = %v, want ErrUnsupportedByOfficialAPI", err)
	}
	if _, err := client.ConnectOperation(ctx, "abc"); !errors.Is(err, ErrUnsupportedByOfficialAPI) {
		t.Errorf("ConnectOperation err = %v, want ErrUnsupportedByOfficialAPI", err)
	}

	// The official API has no operations to speak of, and saying so costs no
	// request.
	supported, err := client.SupportsOperations(ctx)
	if err != nil || supported {
		t.Errorf("SupportsOperations = %v, %v; want false, nil", supported, err)
	}
}

// A body that is not valid JSON is rejected with a stack trace rather than
// Piston's usual message field, and the error must still say something.
func TestMalformedJSONStackIsReported(t *testing.T) {
	client := fakeOperationsInstance(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"stack":"SyntaxError: Unexpected end of JSON input\n    at parse (<anonymous>)"}`))
	})

	_, err := client.GetOperations(context.Background())

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("err = %v, want an *APIError", err)
	}
	// Only the first line: the frames below it are noise in a one-line error.
	if apiErr.Message != "SyntaxError: Unexpected end of JSON input" {
		t.Errorf("Message = %q, want the first line of the stack", apiErr.Message)
	}
	if !errors.Is(err, ErrBadRequest) {
		t.Error("a 400 carrying a stack no longer matches ErrBadRequest")
	}
}

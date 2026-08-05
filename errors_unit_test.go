package gopiston

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestParseAPIErrorMessage(t *testing.T) {
	for _, tc := range []struct {
		name, body, want string
	}{
		{"json message", `{"message":"whitelist only"}`, "whitelist only"},
		{"trims space", `{"message":"  padded  "}`, "padded"},
		{"explicit null", `{"message":null}`, ""},
		{"absent", `{}`, ""},
		{"html body", "<!doctype html><html>404</html>", ""},
		{"empty body", "", ""},
		{"json array", `[1,2,3]`, ""},
	} {
		if got := parseAPIErrorMessage([]byte(tc.body)); got != tc.want {
			t.Errorf("%s: parseAPIErrorMessage(%q) = %q, want %q", tc.name, tc.body, got, tc.want)
		}
	}
}

// A non-JSON body must never be dumped into the error string; the status text
// stands in and the raw body stays on the struct.
func TestAPIErrorMessageFallsBackToStatusText(t *testing.T) {
	html := "<!doctype html><html><body>a very long error page</body></html>"
	err := newAPIError(http.StatusNotFound, []byte(html), false)

	assert(err.Error(), "piston: not found (HTTP 404)", t)
	assert(string(err.Body), html, t)

	unknown := newAPIError(599, nil, false)
	assert(unknown.Error(), "piston: unexpected response (HTTP 599)", t)
}

func TestAPIErrorMessageUsesServerMessage(t *testing.T) {
	err := newAPIError(http.StatusForbidden, []byte(`{"message":"Public Piston API is now whitelist only"}`), false)
	assert(err.Error(), "piston: Public Piston API is now whitelist only (HTTP 403)", t)
}

// When the client has no key, an unauthorized response should say so.
func TestAPIErrorHintsAtMissingAPIKey(t *testing.T) {
	err := newAPIError(http.StatusForbidden, []byte(`{"message":"whitelist only"}`), true)
	if !strings.Contains(err.Error(), "WithAPIKey") {
		t.Errorf("Expected a hint about WithAPIKey, got %q", err.Error())
	}

	withKey := newAPIError(http.StatusForbidden, []byte(`{"message":"whitelist only"}`), false)
	if strings.Contains(withKey.Error(), "WithAPIKey") {
		t.Errorf("Expected no key hint when a key was configured, got %q", withKey.Error())
	}
}

func TestAPIErrorIs(t *testing.T) {
	for _, tc := range []struct {
		status   int
		matches  []error
		excludes []error
	}{
		{http.StatusBadRequest, []error{ErrBadRequest}, []error{ErrUnauthorized, ErrServer}},
		{http.StatusUnauthorized, []error{ErrUnauthorized}, []error{ErrBadRequest}},
		{http.StatusForbidden, []error{ErrUnauthorized}, []error{ErrNotFound}},
		{http.StatusNotFound, []error{ErrNotFound}, []error{ErrUnauthorized}},
		{http.StatusTooManyRequests, []error{ErrRateLimited}, []error{ErrServer}},
		{http.StatusInternalServerError, []error{ErrServer}, []error{ErrBadRequest}},
		{http.StatusBadGateway, []error{ErrServer}, []error{ErrNotFound}},
	} {
		err := error(newAPIError(tc.status, nil, false))
		for _, want := range tc.matches {
			if !errors.Is(err, want) {
				t.Errorf("status %d: expected errors.Is(err, %v)", tc.status, want)
			}
		}
		for _, notWant := range tc.excludes {
			if errors.Is(err, notWant) {
				t.Errorf("status %d: expected NOT errors.Is(err, %v)", tc.status, notWant)
			}
		}
	}
}

// ErrAPIKeyRequired is derived from client state, not from the server's
// wording, so it stays correct if emkc rewords the whitelist notice.
func TestAPIErrorAPIKeyRequired(t *testing.T) {
	missing := error(newAPIError(http.StatusForbidden, nil, true))
	if !errors.Is(missing, ErrAPIKeyRequired) {
		t.Error("Expected ErrAPIKeyRequired when no key was configured")
	}
	if !errors.Is(missing, ErrUnauthorized) {
		t.Error("Expected ErrAPIKeyRequired to imply ErrUnauthorized")
	}

	rejected := error(newAPIError(http.StatusForbidden, nil, false))
	if errors.Is(rejected, ErrAPIKeyRequired) {
		t.Error("Expected no ErrAPIKeyRequired when a key was configured")
	}
	if !errors.Is(rejected, ErrUnauthorized) {
		t.Error("Expected a rejected key to still be ErrUnauthorized")
	}

	// A missing key on a non-auth status is not an API-key problem.
	rateLimited := error(newAPIError(http.StatusTooManyRequests, nil, true))
	if errors.Is(rateLimited, ErrAPIKeyRequired) {
		t.Error("Expected ErrAPIKeyRequired not to match a 429")
	}
}

// The full path: a real HTTP round trip against an error response must surface
// as an *APIError carrying status, message and body.
func TestErrorRoundTrip(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"message":"Public Piston API is now whitelist only as of 2/15/2026."}`))
	}))
	defer server.Close()

	c := NewClient(server.URL, WithHTTPClient(server.Client()))
	_, err := c.GetRuntimes(context.Background())
	if err == nil {
		t.Fatal("Expected an error")
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("Expected an *APIError, got %T: %v", err, err)
	}
	assert(apiErr.StatusCode, http.StatusForbidden, t)
	assert(apiErr.Message, "Public Piston API is now whitelist only as of 2/15/2026.", t)
	if !errors.Is(err, ErrUnauthorized) || !errors.Is(err, ErrAPIKeyRequired) {
		t.Errorf("Expected the error to classify as unauthorized/key-required, got %v", err)
	}
}

// Requests must carry the API key and, for bodied requests only, a JSON
// content type — and must hit the expected path.
func TestRequestShape(t *testing.T) {
	type captured struct {
		method, path, auth, contentType, body string
	}
	var got []captured

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, r.ContentLength)
		if r.ContentLength > 0 {
			_, _ = r.Body.Read(buf)
		}
		got = append(got, captured{
			method:      r.Method,
			path:        r.URL.Path,
			auth:        r.Header.Get("Authorization"),
			contentType: r.Header.Get("Content-Type"),
			body:        string(buf),
		})
		_, _ = w.Write([]byte(`[]`))
	}))
	defer server.Close()

	// No trailing slash, to prove normalization feeds the right path.
	c := NewClient(server.URL+"/api/v2", WithAPIKey("secret"), WithHTTPClient(server.Client()))
	if _, err := c.GetRuntimes(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Execute(context.Background(), "python", "3.10.0", []File{{Content: "print(1)"}}); err != nil {
		// The stub returns `[]`, which fails to decode into a struct; the
		// request itself is what matters here.
		_ = err
	}

	if len(got) != 2 {
		t.Fatalf("Expected 2 requests, got %d", len(got))
	}

	assert(got[0].method, http.MethodGet, t)
	assert(got[0].path, "/api/v2/runtimes", t)
	assert(got[0].auth, "secret", t)
	// A bodyless GET should not claim a JSON entity.
	assert(got[0].contentType, "", t)

	assert(got[1].method, http.MethodPost, t)
	assert(got[1].path, "/api/v2/execute", t)
	assert(got[1].auth, "secret", t)
	assert(got[1].contentType, "application/json", t)
	if !strings.Contains(got[1].body, `"language":"python"`) {
		t.Errorf("Expected the execute body to carry the language, got %s", got[1].body)
	}
}

// An empty version must be sent as the "*" selector so the server resolves the
// latest itself, rather than the client guessing from an unordered list.
func TestExecuteSendsLatestSelector(t *testing.T) {
	var body string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(buf)
		body = string(buf)
		_, _ = w.Write([]byte(`{"language":"python","version":"3.10.0","run":{}}`))
	}))
	defer server.Close()

	c := NewClient(server.URL, WithHTTPClient(server.Client()))
	if _, err := c.Execute(context.Background(), "python", "", []File{{Content: "print(1)"}}); err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(body, `"version":"*"`) {
		t.Errorf("Expected an empty version to be sent as %q, got %s", latestVersionSelector, body)
	}
}

// An interpreted language has no compile stage, and nil must be distinguishable
// from a stage that exited 0.
func TestCompileStageAbsentDecodesToNil(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"language":"python","version":"3.10.0","run":{"stdout":"hi\n","output":"hi\n","code":0}}`))
	}))
	defer server.Close()

	c := NewClient(server.URL, WithHTTPClient(server.Client()))
	execution, err := c.Execute(context.Background(), "python", "3.10.0", []File{{Content: "print('hi')"}})
	if err != nil {
		t.Fatal(err)
	}

	if execution.Compile != nil {
		t.Errorf("Expected no compile stage for an interpreted language, got %+v", execution.Compile)
	}
	assert(execution.GetOutput(), "hi\n", t)
}

// A JSON null body used to decode into a nil *Runtimes that every caller then
// dereferenced. Returning a slice makes this a no-op instead of a panic.
func TestNullRuntimesBodyDoesNotPanic(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`null`))
	}))
	defer server.Close()

	c := NewClient(server.URL, WithHTTPClient(server.Client()))

	runtimes, err := c.GetRuntimes(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	assert(len(runtimes), 0, t)

	languages, err := c.GetLanguages(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	assert(len(languages), 0, t)

	if _, err := c.GetLatestVersion(context.Background(), "python"); !errors.Is(err, ErrLanguageNotFound) {
		t.Errorf("Expected ErrLanguageNotFound, got %v", err)
	}
}

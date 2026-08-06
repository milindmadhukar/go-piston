package gopiston

import (
	"context"
	"net/http"
	"os"
	"slices"
	"sync"
	"testing"
	"time"
)

// The live tests run against a real Piston instance.
//
//	PISTON_BASE_URL   base URL of the instance to test against. Defaults to
//	                  the official API, which is whitelist-only, so point this
//	                  at a self-hosted instance to exercise the full suite.
//	PISTON_API_KEY    API key, required by the official API.
//
// The unit tests below and in the *_unit_test.go files need neither.
var (
	baseURL = os.Getenv("PISTON_BASE_URL")
	apiKey  = os.Getenv("PISTON_API_KEY")
)

var client = newTestClient()

func newTestClient() *Client {
	target := baseURL
	if target == "" {
		target = OfficialAPIBaseURL
	}

	opts := []ClientOption{WithAPIKey(apiKey)}

	// The official API rejects anything faster than one request per 200ms with
	// a 429. Round-trip latency alone spaces these tests out from most places,
	// but not from a CI runner sitting close to emkc, so throttle explicitly
	// rather than let the suite pass or fail on where it runs from.
	if isOfficialAPI(normalizeBaseURL(target)) {
		opts = append(opts, WithHTTPClient(&http.Client{
			Transport: &throttledTransport{minInterval: 300 * time.Millisecond},
		}))
	}

	return NewClient(target, opts...)
}

// throttledTransport spaces the requests it sends at least minInterval apart.
// The lock is held across the wait, so requests serialize — which is what the
// rate limit wants, and costs nothing here because the live tests are
// sequential anyway.
type throttledTransport struct {
	minInterval time.Duration

	mu   sync.Mutex
	last time.Time
}

func (transport *throttledTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	transport.mu.Lock()
	if wait := transport.minInterval - time.Since(transport.last); wait > 0 {
		time.Sleep(wait)
	}
	transport.last = time.Now()
	transport.mu.Unlock()

	return http.DefaultTransport.RoundTrip(req)
}

// testContext bounds every live request. A Piston instance consulting the
// upstream package index can hang for minutes, and an unbounded context turns
// that into an opaque ten-minute test timeout instead of a clear failure.
func testContext(t *testing.T) context.Context {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	t.Cleanup(cancel)
	return ctx
}

func assert(expected, got interface{}, t *testing.T) {
	if expected != got {
		t.Errorf("Expected - %v, but got %v!", expected, got)
	}
}

// requireExecuteAccess skips the test when the target instance is known to
// reject execution. The official API still serves /runtimes without a key but
// rejects /execute, so without a key there is nothing to learn from running
// these — CI covers them against a self-hosted instance instead.
func requireExecuteAccess(t *testing.T) {
	t.Helper()

	if client.IsOfficialAPI() && apiKey == "" {
		t.Skip("skipping: the official Piston API is whitelist-only; set PISTON_API_KEY, or PISTON_BASE_URL for a self-hosted instance")
	}
}

// requireInteractiveAccess skips the test when the target instance does not
// serve the interactive endpoint. The official API does not, and Connect
// refuses to dial it at all — TestConnectGuardedOnOfficialAPI covers that
// refusal, so there is nothing a live session here could add.
func requireInteractiveAccess(t *testing.T) {
	t.Helper()

	if client.IsOfficialAPI() {
		t.Skip("skipping: the official Piston API does not serve the interactive endpoint; set PISTON_BASE_URL to a self-hosted instance")
	}

	requireExecuteAccess(t)
}

// requireLanguage skips the test when the target instance has no runtime for
// the given language, since a self-hosted instance may install only a subset.
func requireLanguage(t *testing.T, language string) {
	t.Helper()

	requireExecuteAccess(t)

	runtimes, err := client.GetRuntimes(testContext(t))
	if err != nil {
		t.Fatal(err)
	}

	for _, runtime := range runtimes {
		if runtime.Language == language || slices.Contains(runtime.Aliases, language) {
			return
		}
	}

	t.Skipf("skipping: %q is not installed on this Piston instance", language)
}

package gopiston

import (
	"context"
	"net/http"
	"os"
	"testing"
)

// PISTON_BASE_URL lets the live-network tests target a self-hosted Piston
// instance instead of the official API, e.g. PISTON_BASE_URL=http://localhost:2000/api/v2/.
// It defaults to the official API, which requires PISTON_API_KEY.
var baseURL = os.Getenv("PISTON_BASE_URL")

var apiKey = os.Getenv("PISTON_API_KEY")

var client = newTestClient()

func newTestClient() *Client {
	if baseURL == "" {
		return NewClient(OfficialAPIBaseURL, WithAPIKey(apiKey))
	}
	return NewClient(baseURL, WithAPIKey(apiKey))
}

func assert(expected, got interface{}, t *testing.T) {
	if expected != got {
		t.Errorf("Expected - %v, but got %v!", expected, got)
	}
}

// requireLanguage skips the test if the target Piston instance doesn't have
// the given language (or one of its aliases) installed, since a self-hosted
// instance may only have a subset of runtimes available.
func requireLanguage(t *testing.T, language string) {
	t.Helper()

	runtimes, err := client.GetRuntimes(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	for _, runtime := range *runtimes {
		if runtime.Language == language || isPresent(runtime.Aliases, language) {
			return
		}
	}

	t.Skipf("skipping: %q is not installed on this Piston instance", language)
}

func TestNewClientDefaults(t *testing.T) {
	c := NewClient("http://localhost:2000/api/v2/")

	assert(c.BaseURL, "http://localhost:2000/api/v2/", t)
	assert(c.ApiKey, "", t)
	if c.HttpClient != http.DefaultClient {
		t.Errorf("Expected HttpClient to default to http.DefaultClient")
	}
}

func TestWithAPIKey(t *testing.T) {
	c := NewClient(OfficialAPIBaseURL, WithAPIKey("test-key"))
	assert(c.ApiKey, "test-key", t)
}

func TestWithHTTPClient(t *testing.T) {
	custom := &http.Client{}
	c := NewClient("http://localhost:2000/api/v2/", WithHTTPClient(custom))
	if c.HttpClient != custom {
		t.Errorf("Expected HttpClient to be the custom client passed to WithHTTPClient")
	}
}

func TestOfficialAPIBaseURL(t *testing.T) {
	assert(OfficialAPIBaseURL, "https://emkc.org/api/v2/piston/", t)
}

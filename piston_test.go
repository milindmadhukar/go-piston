package gopiston

import (
	"net/http"
	"os"
	"testing"
)

var apiKey = os.Getenv("PISTON_API_KEY")

var client = NewClient(OfficialAPIBaseURL, WithAPIKey(apiKey))

func assert(expected, got interface{}, t *testing.T) {
	if expected != got {
		t.Errorf("Expected - %v, but got %v!", expected, got)
	}
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

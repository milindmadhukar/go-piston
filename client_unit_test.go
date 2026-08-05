package gopiston

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewClientDefaults(t *testing.T) {
	c := NewClient("http://localhost:2000/api/v2/")

	assert(c.BaseURL(), "http://localhost:2000/api/v2/", t)
	assert(c.apiKey, "", t)
	assert(c.IsOfficialAPI(), false, t)
	if c.httpClient != http.DefaultClient {
		t.Errorf("Expected httpClient to default to http.DefaultClient")
	}
}

func TestWithAPIKey(t *testing.T) {
	c := NewClient(OfficialAPIBaseURL, WithAPIKey("test-key"))
	assert(c.apiKey, "test-key", t)
}

func TestWithHTTPClient(t *testing.T) {
	custom := &http.Client{}
	c := NewClient("http://localhost:2000/api/v2/", WithHTTPClient(custom))
	if c.httpClient != custom {
		t.Errorf("Expected httpClient to be the custom client passed to WithHTTPClient")
	}
}

// A nil client must be ignored rather than stored, since storing it would
// panic on the first request.
func TestWithHTTPClientNil(t *testing.T) {
	c := NewClient("http://localhost:2000/api/v2/", WithHTTPClient(nil))
	if c.httpClient == nil {
		t.Fatal("Expected WithHTTPClient(nil) to leave the default client in place")
	}
}

func TestOfficialAPIBaseURL(t *testing.T) {
	assert(OfficialAPIBaseURL, "https://emkc.org/api/v2/piston/", t)
}

func TestNormalizeBaseURL(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{"http://localhost:2000/api/v2/", "http://localhost:2000/api/v2/"},
		{"http://localhost:2000/api/v2", "http://localhost:2000/api/v2/"},
		{"http://localhost:2000/api/v2///", "http://localhost:2000/api/v2/"},
		{"  https://emkc.org/api/v2/piston  ", "https://emkc.org/api/v2/piston/"},
	} {
		if got := normalizeBaseURL(tc.in); got != tc.want {
			t.Errorf("normalizeBaseURL(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// A base URL without a trailing slash used to concatenate into
// ".../api/v2runtimes"; normalization must make both spellings hit the same
// path.
func TestEndpointJoinsWithoutTrailingSlash(t *testing.T) {
	withSlash := NewClient("http://localhost:2000/api/v2/").endpoint("runtimes")
	withoutSlash := NewClient("http://localhost:2000/api/v2").endpoint("runtimes")

	assert(withSlash, "http://localhost:2000/api/v2/runtimes", t)
	assert(withoutSlash, withSlash, t)
}

func TestIsOfficialAPI(t *testing.T) {
	for _, tc := range []struct {
		baseURL string
		want    bool
	}{
		{OfficialAPIBaseURL, true},
		{"https://emkc.org/api/v2/piston", true},
		{"https://EMKC.org/api/v2/piston/", true},
		{"http://localhost:2000/api/v2/", false},
		{"https://piston.example.com/api/v2/", false},
	} {
		if got := NewClient(tc.baseURL).IsOfficialAPI(); got != tc.want {
			t.Errorf("NewClient(%q).IsOfficialAPI() = %v, want %v", tc.baseURL, got, tc.want)
		}
	}
}

// Package management must fail before any request is made, so a client
// pointed at the official API can never mutate anything.
func TestPackageManagementGuardedOnOfficialAPI(t *testing.T) {
	var requests int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
	}))
	defer server.Close()

	// Point the transport at the recording server while the client still
	// believes it targets the official API, so any leaked request is visible.
	c := NewClient(OfficialAPIBaseURL, WithHTTPClient(server.Client()))
	c.baseURL = server.URL + "/"

	if _, err := c.GetPackages(context.Background()); !errors.Is(err, ErrUnsupportedByOfficialAPI) {
		t.Errorf("GetPackages: expected ErrUnsupportedByOfficialAPI, got %v", err)
	}
	if _, err := c.InstallPackage(context.Background(), "python", "3.10.0"); !errors.Is(err, ErrUnsupportedByOfficialAPI) {
		t.Errorf("InstallPackage: expected ErrUnsupportedByOfficialAPI, got %v", err)
	}
	if _, err := c.UninstallPackage(context.Background(), "python", "3.10.0"); !errors.Is(err, ErrUnsupportedByOfficialAPI) {
		t.Errorf("UninstallPackage: expected ErrUnsupportedByOfficialAPI, got %v", err)
	}

	if requests != 0 {
		t.Errorf("Expected the guard to make no requests, got %d", requests)
	}
}

func TestCompareVersions(t *testing.T) {
	for _, tc := range []struct {
		a, b string
		want string // "<", ">" or "="
	}{
		{"3.10.0", "3.9.4", ">"}, // the case that breaks string ordering
		{"3.9.4", "3.10.0", "<"},
		{"2.7.18", "3.10.0", "<"},
		{"3.10.0", "3.10.0", "="},
		{"1.0", "1.0.0", "="},
		{"0.36.1", "0.4.0", ">"},
	} {
		got := compareVersions(tc.a, tc.b)
		var sign string
		switch {
		case got < 0:
			sign = "<"
		case got > 0:
			sign = ">"
		default:
			sign = "="
		}
		if sign != tc.want {
			t.Errorf("compareVersions(%q, %q) = %d (%s), want %s", tc.a, tc.b, got, sign, tc.want)
		}
	}
}

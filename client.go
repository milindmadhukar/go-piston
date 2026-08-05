package gopiston

import (
	"net/http"
	"net/url"
	"strings"
)

// OfficialAPIBaseURL is the base URL of the officially hosted Piston API.
// Access to it requires an API key granted at the maintainer's discretion; see
// the README for details. Most users should self-host Piston instead and pass
// their own instance's base URL to NewClient.
const OfficialAPIBaseURL = "https://emkc.org/api/v2/piston/"

// officialAPIHost is the host serving the officially hosted Piston API.
const officialAPIHost = "emkc.org"

// Client is a Piston API client. Create one with NewClient; the zero value is
// not usable.
//
// All fields are unexported and set at construction. In particular this means
// the target instance cannot be swapped after the fact, so the official-API
// restrictions enforced by InstallPackage and UninstallPackage cannot be
// bypassed by reassigning the base URL.
type Client struct {
	httpClient  *http.Client
	baseURL     string // always normalized to end in exactly one slash
	apiKey      string
	officialAPI bool
}

// ClientOption configures a Client created via NewClient.
type ClientOption func(*Client)

// WithAPIKey sets the API key sent in the Authorization header of every
// request. It is required by the official Piston API.
func WithAPIKey(apiKey string) ClientOption {
	return func(client *Client) {
		client.apiKey = apiKey
	}
}

// WithHTTPClient overrides the *http.Client used to make requests. A nil
// client is ignored, leaving the default in place.
func WithHTTPClient(httpClient *http.Client) ClientOption {
	return func(client *Client) {
		if httpClient == nil {
			return
		}
		client.httpClient = httpClient
	}
}

// NewClient creates a Client for the Piston instance at baseURL, typically a
// self-hosted instance. To use the official API instead, pass
// OfficialAPIBaseURL together with WithAPIKey.
//
// baseURL is normalized to end in a single slash, so both
// "http://host:2000/api/v2" and "http://host:2000/api/v2/" work.
func NewClient(baseURL string, opts ...ClientOption) *Client {
	client := &Client{
		httpClient: http.DefaultClient,
		baseURL:    normalizeBaseURL(baseURL),
	}

	for _, opt := range opts {
		opt(client)
	}

	// Computed after the options run so that a future base-URL option could
	// not leave the flag stale.
	client.officialAPI = isOfficialAPI(client.baseURL)

	return client
}

// BaseURL returns the normalized base URL the client targets.
func (client *Client) BaseURL() string { return client.baseURL }

// IsOfficialAPI reports whether the client targets the officially hosted
// Piston API rather than a self-hosted instance. Package management is
// unavailable on the official API; see InstallPackage.
func (client *Client) IsOfficialAPI() bool { return client.officialAPI }

// normalizeBaseURL trims surrounding whitespace and collapses any run of
// trailing slashes into exactly one, so endpoint paths can be appended
// directly. Without this, a base URL of "http://host:2000/api/v2" would
// silently produce "http://host:2000/api/v2runtimes".
func normalizeBaseURL(baseURL string) string {
	return strings.TrimRight(strings.TrimSpace(baseURL), "/") + "/"
}

// endpoint returns the absolute URL of a Piston endpoint path such as
// "runtimes".
func (client *Client) endpoint(path string) string { return client.baseURL + path }

// isOfficialAPI reports whether a normalized base URL points at the officially
// hosted Piston API. Detection is host-based so that any emkc.org endpoint is
// recognized, not only the exact OfficialAPIBaseURL string.
func isOfficialAPI(normalizedBaseURL string) bool {
	if strings.EqualFold(normalizedBaseURL, OfficialAPIBaseURL) {
		return true
	}
	parsed, err := url.Parse(normalizedBaseURL)
	if err != nil {
		return false
	}
	return strings.EqualFold(parsed.Hostname(), officialAPIHost)
}

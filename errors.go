package gopiston

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

// Sentinel errors returned by, or matchable against, errors from a Piston
// instance. Match them with errors.Is. To inspect the status code or the
// server's own message, use errors.As with *APIError.
var (
	// ErrBadRequest indicates the request was rejected as malformed (HTTP 400),
	// most often because of an unknown language or version.
	ErrBadRequest = errors.New("bad request")

	// ErrUnauthorized indicates the instance rejected the request's
	// credentials (HTTP 401 or 403). The official Piston API is whitelist-only
	// and returns this to clients without an accepted API key.
	ErrUnauthorized = errors.New("unauthorized")

	// ErrAPIKeyRequired indicates the request was rejected as unauthorized and
	// the client was configured without an API key. It implies
	// ErrUnauthorized: any error matching ErrAPIKeyRequired also matches
	// ErrUnauthorized.
	ErrAPIKeyRequired = errors.New("api key required")

	// ErrNotFound indicates the endpoint or resource does not exist (HTTP 404).
	ErrNotFound = errors.New("not found")

	// ErrRateLimited indicates the instance is rate limiting the client
	// (HTTP 429).
	ErrRateLimited = errors.New("rate limited")

	// ErrServer indicates the instance failed to handle the request (HTTP 5xx).
	ErrServer = errors.New("server error")

	// ErrUnsupportedByOfficialAPI is returned by operations only a self-hosted
	// Piston instance can perform, such as installing or uninstalling packages.
	ErrUnsupportedByOfficialAPI = errors.New("operation is not supported by the official Piston API")

	// ErrLanguageNotFound is returned when the instance has no runtime
	// installed for the requested language.
	ErrLanguageNotFound = errors.New("language not found")
)

// APIError is returned when a Piston instance responds with a non-2xx status.
// Piston reports failures as a JSON body of the form {"message": "..."}, which
// is parsed into Message. Body holds the raw response so that a reply from
// something other than Piston itself — a reverse proxy returning HTML, say —
// is still available for debugging.
type APIError struct {
	// StatusCode is the HTTP status code of the response.
	StatusCode int

	// Message is the "message" field of Piston's JSON error body. It is empty
	// if the body was not JSON or carried no message.
	Message string

	// Body is the raw, unparsed response body.
	Body []byte

	// apiKeyMissing records whether the client that produced this error was
	// configured without an API key. It is what makes ErrAPIKeyRequired
	// detectable without pattern-matching on the server's prose.
	apiKeyMissing bool
}

func newAPIError(statusCode int, body []byte, apiKeyMissing bool) *APIError {
	return &APIError{
		StatusCode:    statusCode,
		Message:       parseAPIErrorMessage(body),
		Body:          body,
		apiKeyMissing: apiKeyMissing,
	}
}

// Error implements the error interface.
func (e *APIError) Error() string {
	msg := e.Message
	if msg == "" {
		// Degrade to the status text rather than dumping a body we already
		// know is not Piston's documented JSON shape. The raw body is still
		// available on Body.
		if text := http.StatusText(e.StatusCode); text != "" {
			msg = strings.ToLower(text)
		} else {
			msg = "unexpected response"
		}
	}
	if e.apiKeyMissing && isUnauthorizedStatus(e.StatusCode) {
		return fmt.Sprintf("piston: %s (HTTP %d; no API key configured, set one with WithAPIKey)", msg, e.StatusCode)
	}
	return fmt.Sprintf("piston: %s (HTTP %d)", msg, e.StatusCode)
}

// Is reports whether this error matches one of the package's sentinel errors,
// so callers can classify failures with errors.Is:
//
//	if errors.Is(err, piston.ErrRateLimited) { ... }
//
// Check ErrAPIKeyRequired before ErrUnauthorized: the former implies the latter.
func (e *APIError) Is(target error) bool {
	switch target {
	case ErrAPIKeyRequired:
		return e.apiKeyMissing && isUnauthorizedStatus(e.StatusCode)
	case ErrUnauthorized:
		return isUnauthorizedStatus(e.StatusCode)
	case ErrBadRequest:
		return e.StatusCode == http.StatusBadRequest
	case ErrNotFound:
		return e.StatusCode == http.StatusNotFound
	case ErrRateLimited:
		return e.StatusCode == http.StatusTooManyRequests
	case ErrServer:
		return e.StatusCode >= 500
	}
	return false
}

func isUnauthorizedStatus(code int) bool {
	return code == http.StatusUnauthorized || code == http.StatusForbidden
}

// parseAPIErrorMessage extracts Piston's documented {"message": "..."} error
// payload. It returns an empty string for any body that is not a JSON object
// carrying a string message, including HTML from a reverse proxy, an empty
// body, or an explicit JSON null.
func parseAPIErrorMessage(body []byte) string {
	// *string rather than string so an explicit null is treated as absent.
	var payload struct {
		Message *string `json:"message"`
	}
	if err := json.Unmarshal(body, &payload); err != nil || payload.Message == nil {
		return ""
	}
	return strings.TrimSpace(*payload.Message)
}

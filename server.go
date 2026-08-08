package gopiston

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

// apiPathSuffix is the path a Piston base URL ends in. It is trimmed to reach
// the instance root, which is the one endpoint outside the versioned API.
const apiPathSuffix = "api/v2/"

// ServerVersion returns the version an instance reports for itself, such as
// "3.1.1".
//
// This is a diagnostic, not a feature test. The endpoint it reads lives at the
// instance root rather than under the API base URL, so the base URL's trailing
// "api/v2/" is trimmed to find it — which means it fails with an error matching
// ErrNotFound behind a proxy that exposes only the API path, and against
// instances predating the endpoint, even though both are perfectly usable.
// Never gate a feature on it; use SupportsOperations, which asks the endpoint
// in question directly.
//
// The official API does not serve it, so a client targeting it fails with an
// error matching ErrUnsupportedByOfficialAPI without making a request.
func (client *Client) ServerVersion(ctx context.Context) (string, error) {
	if client.officialAPI {
		return "", fmt.Errorf("piston: server version: %w", ErrUnsupportedByOfficialAPI)
	}

	root := strings.TrimSuffix(client.baseURL, apiPathSuffix)

	body, err := client.doRequest(ctx, http.MethodGet, root, nil)
	if err != nil {
		return "", err
	}

	var payload struct {
		Message string `json:"message"`
	}
	if err := decodeJSON(body, &payload); err != nil {
		return "", err
	}

	// The body is a greeting, "Piston v3.1.1", not a bare version.
	version := strings.TrimPrefix(strings.TrimSpace(payload.Message), "Piston v")
	if version == "" || strings.ContainsAny(version, " \t") {
		return "", fmt.Errorf("piston: server version: unrecognized greeting %q", payload.Message)
	}

	return version, nil
}

package gopiston

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// maxResponseBytes caps how much of a response is buffered, so a misconfigured
// or hostile endpoint cannot exhaust memory.
const maxResponseBytes = 8 << 20 // 8 MiB

// doRequest sends a request to the Piston instance and returns the raw
// response body. Any non-2xx status is reported as an *APIError.
func (client *Client) doRequest(ctx context.Context, method string, url string, body []byte) ([]byte, error) {
	var reader io.Reader = http.NoBody
	if len(body) > 0 {
		reader = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reader)
	if err != nil {
		return nil, fmt.Errorf("piston: build request: %w", err)
	}

	if len(body) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}
	if client.apiKey != "" {
		// Piston expects the bare key, with no "Bearer " scheme. This is
		// deliberate, not an omission.
		req.Header.Set("Authorization", client.apiKey)
	}

	resp, err := client.httpClient.Do(req)
	if err != nil {
		// http.Client already wraps this in *url.Error, which unwraps to
		// context.DeadlineExceeded / context.Canceled, so errors.Is keeps
		// working through this wrap.
		return nil, fmt.Errorf("piston: %s: %w", method, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return nil, fmt.Errorf("piston: read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, newAPIError(resp.StatusCode, respBody, client.apiKey == "")
	}

	return respBody, nil
}

// decodeJSON unmarshals a response body, tagging failures so they are
// distinguishable from transport and API errors.
func decodeJSON(data []byte, v any) error {
	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("piston: decode response: %w", err)
	}
	return nil
}

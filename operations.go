package gopiston

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

// Operations are an addition to the Piston API, not part of the original v2
// surface. Every method here answers ErrOperationsUnsupported against an
// instance that predates them; see SupportsOperations.
//
// They exist because InstallPackage is synchronous — it holds the HTTP request
// open until the package is installed. That is fine for a runtime that is a
// tarball to fetch, and unworkable for one that is compiled from source, which
// can take an hour. Starting an operation returns immediately with an id to
// poll or stream instead.

// StartOperation begins an install or uninstall in the background and returns
// as soon as the instance has accepted it, without waiting for it to finish.
//
// Track it with GetOperation, read its output with GetOperationLog, or follow
// both live with ConnectOperation.
//
// version is a selector, so "5.x" and "*" work as they do for InstallPackage;
// the returned Operation carries the concrete version that was resolved.
//
// Errors worth distinguishing:
//
//   - ErrConflict — the same package already has an operation in flight. Only
//     one at a time is allowed.
//   - ErrNotFound — no such package. Note that an instance without the
//     operations API answers 404 for the endpoint itself, which is reported as
//     ErrOperationsUnsupported instead.
//   - ErrUnsupportedByOfficialAPI — returned without making a request, since
//     package management needs a self-hosted instance.
func (client *Client) StartOperation(ctx context.Context, kind OperationKind, language string, version string) (*Operation, error) {
	if client.officialAPI {
		return nil, fmt.Errorf("piston: start operation: %w", ErrUnsupportedByOfficialAPI)
	}

	// The instance reads anything other than "uninstall" as an install. Being
	// explicit here keeps a mistyped kind from silently installing something.
	if kind != OperationInstall && kind != OperationUninstall {
		return nil, fmt.Errorf("piston: start operation: unknown kind %q", kind)
	}

	body, err := json.Marshal(operationRequestBody{Kind: kind, Language: language, Version: version})
	if err != nil {
		return nil, fmt.Errorf("piston: encode request: %w", err)
	}

	respBody, err := client.doRequest(ctx, http.MethodPost, client.endpoint("operations"), body)
	if err != nil {
		return nil, operationsError(err)
	}

	operation := &Operation{}
	if err := decodeJSON(respBody, operation); err != nil {
		return nil, err
	}

	return operation, nil
}

// InstallPackageAsync starts installing a package in the background. It is
// StartOperation with OperationInstall, and is the asynchronous counterpart to
// InstallPackage.
func (client *Client) InstallPackageAsync(ctx context.Context, language string, version string) (*Operation, error) {
	return client.StartOperation(ctx, OperationInstall, language, version)
}

// UninstallPackageAsync starts uninstalling a package in the background. It is
// StartOperation with OperationUninstall.
func (client *Client) UninstallPackageAsync(ctx context.Context, language string, version string) (*Operation, error) {
	return client.StartOperation(ctx, OperationUninstall, language, version)
}

// GetOperations returns the operations the instance still remembers, newest
// first. Completed ones are dropped after a while, and all of them are lost
// when the instance restarts.
func (client *Client) GetOperations(ctx context.Context) ([]Operation, error) {
	if client.officialAPI {
		return nil, fmt.Errorf("piston: list operations: %w", ErrUnsupportedByOfficialAPI)
	}

	body, err := client.doRequest(ctx, http.MethodGet, client.endpoint("operations"), nil)
	if err != nil {
		return nil, operationsError(err)
	}

	var operations []Operation
	if err := decodeJSON(body, &operations); err != nil {
		return nil, err
	}

	return operations, nil
}

// GetOperation returns one operation by id.
//
// An id the instance has forgotten — because the operation finished long
// enough ago to be dropped, or because the instance restarted — is reported as
// ErrNotFound, which is indistinguishable from an id that never existed. Treat
// a vanished running operation as finished with an unknown outcome and consult
// GetPackages for the truth.
func (client *Client) GetOperation(ctx context.Context, id string) (*Operation, error) {
	if client.officialAPI {
		return nil, fmt.Errorf("piston: get operation: %w", ErrUnsupportedByOfficialAPI)
	}

	body, err := client.doRequest(ctx, http.MethodGet, client.operationEndpoint(id, ""), nil)
	if err != nil {
		return nil, err
	}

	operation := &Operation{}
	if err := decodeJSON(body, operation); err != nil {
		return nil, err
	}

	return operation, nil
}

// GetOperationLog returns everything the operation has logged so far, as plain
// text. It can be called while the operation is still running, and again after
// it finishes.
//
// If the log has outgrown the instance's buffer, the first line records how
// many earlier lines were dropped.
func (client *Client) GetOperationLog(ctx context.Context, id string) (string, error) {
	if client.officialAPI {
		return "", fmt.Errorf("piston: get operation log: %w", ErrUnsupportedByOfficialAPI)
	}

	// text/plain, so there is nothing to decode — but a 404 is still JSON, and
	// doRequest has already turned that into an *APIError.
	body, err := client.doRequest(ctx, http.MethodGet, client.operationEndpoint(id, "log"), nil)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

// SupportsOperations reports whether the instance serves the operations API.
//
// The endpoints are an addition, so an older instance — including anything
// built from upstream Piston — does not have them. Probe once and remember the
// answer rather than calling this before every operation; it costs a request.
//
// It returns false, nil for an instance without the endpoints, including the
// official API, and an error only when the instance could not be reached or
// answered in some other unexpected way. A caller that cannot tell those apart
// should treat any error as "no" and fall back to InstallPackage.
func (client *Client) SupportsOperations(ctx context.Context) (bool, error) {
	if client.officialAPI {
		return false, nil
	}

	_, err := client.doRequest(ctx, http.MethodGet, client.endpoint("operations"), nil)
	if errors.Is(err, ErrNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return true, nil
}

// operationEndpoint builds the URL of one operation, optionally with a
// subresource such as "log" or "connect".
func (client *Client) operationEndpoint(id string, sub string) string {
	path := "operations/" + id
	if sub != "" {
		path += "/" + sub
	}
	return client.endpoint(path)
}

// operationsError re-reports a 404 from a collection endpoint as
// ErrOperationsUnsupported.
//
// This is only safe on /operations itself, which always exists on an instance
// that has the API — unlike /operations/<id>, where a 404 genuinely means the
// id is unknown and must keep saying so.
func operationsError(err error) error {
	if err == nil || !errors.Is(err, ErrNotFound) {
		return err
	}
	// Wrapping both keeps errors.Is(err, ErrNotFound) true for callers that
	// only classify by status, and preserves the *APIError for errors.As.
	return fmt.Errorf("piston: %w: %w", ErrOperationsUnsupported, err)
}

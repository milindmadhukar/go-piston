package gopiston

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// GetPackages returns every package the instance knows about, along with
// whether it is currently installed.
//
// The official Piston API does not serve this endpoint, so on a client
// targeting it this returns an error matching ErrUnsupportedByOfficialAPI
// without making a request.
//
// On a self-hosted instance this endpoint consults the upstream package index
// and can be slow; pass a context with a timeout.
func (client *Client) GetPackages(ctx context.Context) ([]Package, error) {
	if client.officialAPI {
		return nil, fmt.Errorf("piston: list packages: %w", ErrUnsupportedByOfficialAPI)
	}

	body, err := client.doRequest(ctx, http.MethodGet, client.endpoint("packages"), nil)
	if err != nil {
		return nil, err
	}

	var packages []Package
	if err := decodeJSON(body, &packages); err != nil {
		return nil, err
	}

	return packages, nil
}

// InstallPackage installs the package for the given language and version, as
// listed by GetPackages.
//
// Package management operates on a Piston instance's own runtime store, so it
// is only available on a self-hosted instance. On a client targeting the
// official API this returns an error matching ErrUnsupportedByOfficialAPI
// without making a request.
//
// Installing a package downloads and unpacks a runtime and can take minutes;
// pass a context with a generous timeout.
func (client *Client) InstallPackage(ctx context.Context, language string, version string) (*PackageInstallation, error) {
	if client.officialAPI {
		return nil, fmt.Errorf("piston: install package: %w", ErrUnsupportedByOfficialAPI)
	}
	return client.packageRequest(ctx, http.MethodPost, language, version)
}

// UninstallPackage uninstalls the package for the given language and version.
// Like InstallPackage, it is unavailable on the official API.
func (client *Client) UninstallPackage(ctx context.Context, language string, version string) (*PackageInstallation, error) {
	if client.officialAPI {
		return nil, fmt.Errorf("piston: uninstall package: %w", ErrUnsupportedByOfficialAPI)
	}
	return client.packageRequest(ctx, http.MethodDelete, language, version)
}

func (client *Client) packageRequest(ctx context.Context, method string, language string, version string) (*PackageInstallation, error) {
	body, err := json.Marshal(packageRequestBody{Language: language, Version: version})
	if err != nil {
		return nil, fmt.Errorf("piston: encode request: %w", err)
	}

	respBody, err := client.doRequest(ctx, method, client.endpoint("packages"), body)
	if err != nil {
		return nil, err
	}

	installation := &PackageInstallation{}
	if err := decodeJSON(respBody, installation); err != nil {
		return nil, err
	}

	return installation, nil
}

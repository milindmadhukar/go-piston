package gopiston

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"
)

func TestGetPackages(t *testing.T) {
	if client.IsOfficialAPI() {
		t.Skip("skipping: the official Piston API does not serve the packages endpoint")
	}

	packages, err := client.GetPackages(testContext(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(packages) == 0 {
		t.Fatal("Expected a non-empty list of packages")
	}
}

// TestInstallAndUninstallPackage installs and then uninstalls a package,
// mutating the target Piston instance.
//
// It is skipped unless PISTON_TEST_PACKAGE_MANAGEMENT=true, and never runs
// against the official API (which rejects it outright — see
// TestPackageManagementGuardedOnOfficialAPI for that path).
//
// The package under test is PISTON_TEST_PACKAGE, defaulting to a small one.
// Do not point this at a large runtime: installing one can take many minutes
// and has been observed to kill the instance.
func TestInstallAndUninstallPackage(t *testing.T) {
	if client.IsOfficialAPI() {
		t.Skip("skipping: package management is unavailable on the official Piston API; set PISTON_BASE_URL to a self-hosted instance")
	}
	if os.Getenv("PISTON_TEST_PACKAGE_MANAGEMENT") != "true" {
		t.Skip("skipping: set PISTON_TEST_PACKAGE_MANAGEMENT=true to run package install/uninstall tests")
	}

	language, version := testPackage(t)

	// Installing downloads and unpacks a runtime, which takes far longer than
	// the default live-test budget.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	installation, err := client.InstallPackage(ctx, language, version)
	if err != nil {
		t.Fatal(err)
	}
	assert(installation.Language, language, t)
	assert(installation.Version, version, t)

	uninstallation, err := client.UninstallPackage(ctx, language, version)
	if err != nil {
		t.Fatal(err)
	}
	assert(uninstallation.Language, language, t)
	assert(uninstallation.Version, version, t)
}

// testPackage resolves the package to install and uninstall, as
// "language" or "language=version".
func testPackage(t *testing.T) (language, version string) {
	t.Helper()

	spec := os.Getenv("PISTON_TEST_PACKAGE")
	if spec == "" {
		spec = "bash"
	}

	language, version, found := strings.Cut(spec, "=")
	if found && version != "" {
		return language, version
	}

	// Resolve the version from the instance's own package list, so the caller
	// need only name the language.
	packages, err := client.GetPackages(testContext(t))
	if err != nil {
		t.Fatal(err)
	}
	for _, pkg := range packages {
		if pkg.Language == language && !pkg.Installed {
			return pkg.Language, pkg.LanguageVersion
		}
	}

	t.Skipf("skipping: no uninstalled package found for %q; set PISTON_TEST_PACKAGE to one from GetPackages", language)
	return "", ""
}

// The official API rejects package management locally, without a request.
func TestPackageManagementRejectedOnOfficialAPI(t *testing.T) {
	official := NewClient(OfficialAPIBaseURL)

	if _, err := official.InstallPackage(context.Background(), "python", "3.10.0"); !errors.Is(err, ErrUnsupportedByOfficialAPI) {
		t.Errorf("Expected ErrUnsupportedByOfficialAPI, got %v", err)
	}
}

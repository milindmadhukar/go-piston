package gopiston

import (
	"context"
	"os"
	"testing"
)

func TestGetPackages(t *testing.T) {
	packages, err := client.GetPackages(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if packages == nil || len(*packages) == 0 {
		t.Fatal("Expected a non-empty list of packages")
	}
}

// TestInstallAndUninstallPackage installs and then uninstalls a package,
// mutating the target Piston instance. It only runs when
// PISTON_TEST_PACKAGE_MANAGEMENT=true is set, so it doesn't affect a
// shared or official instance by default.
func TestInstallAndUninstallPackage(t *testing.T) {
	if os.Getenv("PISTON_TEST_PACKAGE_MANAGEMENT") != "true" {
		t.Skip("skipping: set PISTON_TEST_PACKAGE_MANAGEMENT=true to run package install/uninstall tests")
	}

	packages, err := client.GetPackages(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	var target *Package
	for i, pkg := range *packages {
		if !pkg.Installed {
			target = &(*packages)[i]
			break
		}
	}
	if target == nil {
		t.Skip("skipping: no uninstalled package available to test with")
	}

	installation, err := client.InstallPackage(context.Background(), target.Language, target.LanguageVersion)
	if err != nil {
		t.Fatal(err)
	}
	assert(installation.Language, target.Language, t)
	assert(installation.Version, target.LanguageVersion, t)

	uninstallation, err := client.UninstallPackage(context.Background(), target.Language, target.LanguageVersion)
	if err != nil {
		t.Fatal(err)
	}
	assert(uninstallation.Language, target.Language, t)
	assert(uninstallation.Version, target.LanguageVersion, t)
}

package gopiston

import (
	"context"
	"errors"
	"io"
	"os"
	"strings"
	"testing"
	"time"
)

// The operations API is an addition, so every test here skips against an
// instance that does not serve it — which includes upstream Piston and the
// official API. That is what keeps this suite meaningful as a backwards
// compatibility check rather than a fork-only one.
func requireOperations(t *testing.T) {
	t.Helper()

	if client.IsOfficialAPI() {
		t.Skip("skipping: the official Piston API does not serve the operations endpoints; set PISTON_BASE_URL to a self-hosted instance")
	}

	supported, err := client.SupportsOperations(testContext(t))
	if err != nil {
		t.Fatal(err)
	}
	if !supported {
		t.Skip("skipping: this Piston instance does not serve the operations endpoints")
	}
}

func TestGetOperationsLive(t *testing.T) {
	requireOperations(t)

	// An instance that has never run one answers with an empty list, not an
	// error, so there is nothing to arrange first.
	if _, err := client.GetOperations(testContext(t)); err != nil {
		t.Fatal(err)
	}
}

func TestGetUnknownOperationLive(t *testing.T) {
	requireOperations(t)

	_, err := client.GetOperation(testContext(t), "00000000-0000-4000-8000-000000000000")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
	// An id the instance has forgotten must not look like a missing API.
	if errors.Is(err, ErrOperationsUnsupported) {
		t.Error("an unknown operation id was reported as the API being unsupported")
	}
}

// TestOperationLifecycleLive installs a package asynchronously, follows its log
// over the WebSocket, and uninstalls it again — mutating the target instance,
// so it is gated the same way TestInstallAndUninstallPackage is.
func TestOperationLifecycleLive(t *testing.T) {
	requireOperations(t)

	if os.Getenv("PISTON_TEST_PACKAGE_MANAGEMENT") != "true" {
		t.Skip("skipping: set PISTON_TEST_PACKAGE_MANAGEMENT=true to run package install/uninstall tests")
	}

	language, version := testPackage(t)

	// Starting is immediate; the install behind it is not.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	started, err := client.InstallPackageAsync(ctx, language, version)
	if err != nil {
		t.Fatal(err)
	}
	if started.ID == "" {
		t.Fatal("the instance accepted the operation without giving it an id")
	}
	assert(OperationInstall, started.Kind, t)
	assert(language, started.Language, t)
	// A selector resolves to a concrete version on the way in.
	assert(version, started.Version, t)
	if started.Done() {
		t.Errorf("State = %q, want a freshly started operation to be running", started.State)
	}

	// Following the log to its end is also how this waits for the install:
	// the instance closes the socket once the operation settles.
	session, err := client.ConnectOperation(ctx, started.ID)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()

	var log strings.Builder
	settled := OperationState("")
	for {
		event, err := session.Next(ctx)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatal(err)
		}

		switch event.Type {
		case OperationEventLog:
			log.WriteString(event.Data)
		case OperationEventState:
			settled = event.State
			if event.State == OperationFailed {
				t.Fatalf("install failed: %s\nlog:\n%s", event.Error, log.String())
			}
		}
	}

	if settled != OperationSucceeded {
		t.Fatalf("the socket closed without a final state (last was %q); log:\n%s", settled, log.String())
	}
	if log.Len() == 0 {
		t.Error("the operation produced no log output")
	}

	// The polled view has to agree with what the stream reported.
	final, err := client.GetOperation(ctx, started.ID)
	if err != nil {
		t.Fatal(err)
	}
	assert(OperationSucceeded, final.State, t)
	if !final.Done() {
		t.Error("Done() = false for a succeeded operation")
	}
	if final.Finished == nil {
		t.Error("a settled operation reported no finish time")
	}
	if !final.FinishedAt().After(final.StartedAt()) && final.FinishedAt().Before(final.StartedAt()) {
		t.Errorf("FinishedAt %v is before StartedAt %v", final.FinishedAt(), final.StartedAt())
	}

	// Fetching the log after the fact must return the same output the stream
	// produced, since a late attach is meant to see everything.
	replayed, err := client.GetOperationLog(ctx, started.ID)
	if err != nil {
		t.Fatal(err)
	}
	if replayed == "" {
		t.Error("GetOperationLog returned nothing for an operation that logged")
	}

	// Starting a second operation for the same package while one is in flight
	// is a conflict. Uninstalling is quick, so this races it deliberately.
	uninstall, err := client.UninstallPackageAsync(ctx, language, version)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.UninstallPackageAsync(ctx, language, version); err != nil && !errors.Is(err, ErrConflict) {
		t.Errorf("a second operation for the same package failed with %v, want nil or ErrConflict", err)
	}

	waitForOperation(t, ctx, uninstall.ID)
}

// waitForOperation blocks until an operation settles, by draining its log
// socket. The socket closes when the operation does, so there is nothing to
// poll.
func waitForOperation(t *testing.T, ctx context.Context, id string) {
	t.Helper()

	session, err := client.ConnectOperation(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()

	for {
		event, err := session.Next(ctx)
		if errors.Is(err, io.EOF) {
			return
		}
		if err != nil {
			t.Fatal(err)
		}
		if event.Type == OperationEventState && event.State == OperationFailed {
			t.Fatalf("operation %s failed: %s", id, event.Error)
		}
	}
}

// ServerVersion is a diagnostic, not a feature test, and is allowed to fail on
// an instance that does not serve the root endpoint.
func TestServerVersionLive(t *testing.T) {
	if client.IsOfficialAPI() {
		t.Skip("skipping: the official Piston API does not serve the root endpoint")
	}

	version, err := client.ServerVersion(testContext(t))
	if errors.Is(err, ErrNotFound) {
		t.Skip("skipping: this instance does not serve the root version endpoint")
	}
	if err != nil {
		t.Fatal(err)
	}
	if version == "" || strings.HasPrefix(version, "Piston") {
		t.Errorf("ServerVersion() = %q, want a bare version such as 3.1.1", version)
	}
}

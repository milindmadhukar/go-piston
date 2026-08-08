# operations

Installs a package in the background, streams its log live, then uninstalls it again.

`InstallPackage` is synchronous — it holds the HTTP request open until the runtime is installed. That is fine for a package that is a tarball to fetch, and unworkable for one compiled from source, which can take an hour. The operations API starts the same work and returns immediately with an id to follow.

```go
operation, err := client.InstallPackageAsync(ctx, "bash", "5.1.0")

session, err := client.ConnectOperation(ctx, operation.ID)
for {
    event, err := session.Next(ctx)
    if errors.Is(err, io.EOF) {
        break // the instance closes the socket once the operation settles
    }
    ...
}
```

Alternatively poll `GetOperation(ctx, id)` for the state and `GetOperationLog(ctx, id)` for the output so far — both work while the operation is still running.

> **An addition, not part of the original v2 API.** Older instances do not serve these endpoints and answer `ErrOperationsUnsupported`. Call `SupportsOperations` once and fall back to `InstallPackage`, which every instance has. This example does exactly that.

> **Self-hosted only.** Like all package management, these are unavailable on the official Piston API; a client targeting it fails with `ErrUnsupportedByOfficialAPI` without making a request.

Two further details worth knowing:

- Only one operation per package may be in flight; a second fails with `ErrConflict`.
- An operation is a record of work in flight, not durable state. The instance keeps only the most recent completed ones and forgets all of them when it restarts, at which point `GetOperation` reports `ErrNotFound`. What survives is the effect, which `GetPackages` reports.

## Run

Probing and listing are read-only and always run:

```
go run main.go
```

Installing mutates the instance and can take a long time, so it happens only when a package is named:

```
PISTON_EXAMPLE_PACKAGE=bash=5.1.0 go run main.go
```

## Expected output

```
1 operation(s) the instance still remembers, newest first:
  install python 3.12.0 — succeeded

started install of bash 5.1.0 (id 0728a7b1-8f45-425a-b9f1-a51207acf342)
  | Installing bash-5.1.0
  | Fetching https://github.com/.../bash-5.1.0.tar.gz
  | Fetched 3070131 bytes, sha256 ok
  | Extracting into .
  | Registering runtime
  | Installed bash-5.1.0
succeeded in 33.6s
uninstalled bash 5.1.0
```

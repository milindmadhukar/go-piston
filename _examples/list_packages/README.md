# list_packages

Lists every package the instance knows about, and whether it is currently installed.

```go
packages, err := client.GetPackages(context.Background())
```

`InstallPackage(ctx, language, version)` and `UninstallPackage(ctx, language, version)` manage packages, taking a language/version pair from this list.

> **Self-hosted only.** All three methods operate on an instance's own runtime store and are unavailable on the official Piston API. A client targeting it fails with `ErrUnsupportedByOfficialAPI` without making a request.

Listing packages makes the instance consult the upstream package index, which can be slow — pass a context with a timeout. Installing a package downloads and unpacks a runtime and can take minutes.

## Run

```
go run main.go
```

## Expected output

```
python 3.10.0 (installed: true)
node 15.10.0 (installed: true)
bash 5.2.0 (installed: false)
...
```

# list_packages

Lists every package the Piston instance knows about, and whether it is currently installed, via `GetPackages`.

```go
packages, err := client.GetPackages(context.Background())
```

`InstallPackage(ctx, language, version)` and `UninstallPackage(ctx, language, version)` manage packages the same way, taking a language/version pair from this list and returning the installed/uninstalled `language`/`version`.

## Run

```
go run main.go
```

## Expected output

```
python 3.10.0 (installed: true)
node 15.10.0 (installed: true)
bash 5.1.0 (installed: false)
...
```

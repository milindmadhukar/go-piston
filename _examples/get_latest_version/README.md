# get_latest_version

Resolves the latest version of a language installed on the target Piston instance, without executing any code.

```go
latestVersion, err := client.GetLatestVersion(context.Background(), "python")
```

## Run

```
go run main.go
```

## Expected output

```
The latest version of python  supported by Piston API is 3.10.0
```

# get_latest_version

Resolves the highest installed version of a language on the target instance, comparing versions numerically rather than taking whichever the instance happens to list first.

```go
latestVersion, err := client.GetLatestVersion(context.Background(), "python")
```

Calling this before `Execute` is only necessary when the version itself is of interest — passing an empty version to `Execute` lets the instance resolve the latest itself, without the extra request.

## Run

```
go run main.go
```

## Expected output

```
The latest version of python  supported by Piston API is 3.10.0
```

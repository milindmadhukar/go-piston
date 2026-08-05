# get_runtimes

Lists every language, version, and set of aliases supported by the target Piston instance.

```go
runtimes, err := client.GetRuntimes(context.Background())
```

## Run

```
go run main.go
```

## Expected output

```
Runtimes supported by the Piston API are:  [{python 3.10.0 [py py3 python3] ...} ...]
```

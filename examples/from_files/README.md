# from_files

Reads code from files on disk (`main.py`, `test.py`) using `piston.Files(...)` instead of writing source inline, then executes them.

```go
files, err := piston.Files("main.py", "test.py")
response, err := client.Execute(ctx, "python", "", files)
```

## Run

```
go run main.go
```

## Expected output

```
Sentence from main is:  Running Multiple Files with Go-Piston
```

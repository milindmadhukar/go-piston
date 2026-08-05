# multi_file

Executes multiple in-memory files together — a `main.py` entrypoint that imports a `helper.py` module. Contrast with [from_files](../from_files), which reads the same kind of multi-file program from disk via `piston.Files(...)`.

```go
output, err := client.Execute(ctx, "python", "",
	[]piston.File{
		{Name: "main.py", Content: "from helper import shout\nprint(shout('hi'))"},
		{Name: "helper.py", Content: "def shout(s):\n    return s.upper()"},
	})
```

## Run

```
go run main.go
```

## Expected output

```
HI
```

# with_memory_limits

Demonstrates `RunMemoryLimit` and `CompileMemoryLimit`. The program tries to allocate 200MB but is only allowed 16MB for its run stage, so it fails with a non-zero exit code.

```go
output, err := client.Execute(ctx, "python", "",
	[]piston.Code{
		{Content: "x = bytearray(200 * 1024 * 1024)\nprint('allocated')"},
	},
	piston.RunMemoryLimit(16*1024*1024),
	piston.CompileMemoryLimit(16*1024*1024),
)
```

## Run

```
go run main.go
```

## Expected output

```
exit code: 137
stderr: /piston/packages/python/3.10.0/run: line 3:     3 Killed                  python3.10 "$@"
```

The process is killed by the kernel rather than raising a catchable `MemoryError`, so the exit code is 137 (128 + SIGKILL) and the message comes from the runtime's wrapper script, not from Python.

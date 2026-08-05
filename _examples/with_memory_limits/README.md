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
exit code: 1
stderr: ...MemoryError...
```

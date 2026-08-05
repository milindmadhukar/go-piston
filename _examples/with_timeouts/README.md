# with_timeouts

Demonstrates the compile/run timeout and CPU-time options together (`RunTimeout`, `CompileTimeout`, `RunCpuTime`, `CompileCpuTime`). The program sleeps for 3 seconds but is given only a 2 second run timeout, so it is killed before finishing.

```go
output, err := client.Execute(ctx, "python", "",
	[]piston.Code{
		{Content: "import time\nprint('before sleep')\ntime.sleep(3)\nprint('after sleep')"},
	},
	piston.RunTimeout(2*time.Second),
	piston.CompileTimeout(10*time.Second),
	piston.RunCpuTime(2*time.Second),
	piston.CompileCpuTime(10*time.Second),
)
```

## Run

```
go run main.go
```

## Expected output

```
stdout: before sleep
signal: SIGKILL
```

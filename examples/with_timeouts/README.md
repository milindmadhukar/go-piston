# with_timeouts

Demonstrates the compile/run timeout and CPU-time options together (`RunTimeout`, `CompileTimeout`, `RunCpuTime`, `CompileCpuTime`). The program sleeps for 3 seconds but is given only a 2 second run timeout, so it is killed before finishing.

```go
output, err := client.Execute(ctx, "python", "",
	[]piston.File{
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
stdout: 
signal: SIGKILL
```

Note that `stdout` is empty even though the program printed before sleeping: Python buffers stdout when it is not attached to a terminal, and `SIGKILL` gives it no chance to flush. A stage that is killed reports the signal rather than a meaningful exit code, so check `Signal` first.

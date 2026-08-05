# interactive

Streams a running process's output while writing to its stdin, using the WebSocket endpoint at `/api/v2/connect`. This is what `Execute` cannot do: `Execute` sends all input up front and returns everything at once when the job ends.

`Connect` returns a `Session` producing a stream of events:

```go
session, err := client.Connect(ctx, "python", "",
	[]piston.File{{Name: "main.py", Content: source}})
defer session.Close()

for {
	event, err := session.Next(ctx)
	if errors.Is(err, io.EOF) {
		break // the job finished
	}
	switch event.Type {
	case piston.EventStdout:
		fmt.Print(event.Data)
	case piston.EventExit:
		// exactly one of event.Code and event.Signal is set
	}
}

session.SendStdin(ctx, "line\n")
```

> **Self-hosted only.** Interactive execution is not available on the official Piston API; a client targeting it fails with `ErrUnsupportedByOfficialAPI` without connecting.

Two things worth knowing:

- **`Data` is a chunk, not a line.** One write may arrive split across events, and one event may hold several lines.
- **Flush on the other side.** Most runtimes block-buffer stdout when it is not a terminal, so a program that does not flush will deliver nothing until it exits — which defeats the point. The Python here uses `print(..., flush=True)`.

`Session.SendSignal` exists and is protocol-correct, but Piston currently drops signals: its router emits an event named `signal` while the job handler listens for one named `kill`. Verified against a live instance — a process sent `SIGKILL` runs to completion and exits 0.

## Run

```
go run main.go
```

## Expected output

```
runtime: python 3.10.0
stage:   run
stdout:  echo: line 1   (after 37ms)
stdout:  echo: line 2   (after 37ms)
stdout:  echo: line 3   (after 38ms)
exit:    run exited 0
```

The elapsed times are the point: output arrives while the process is still running, not batched at exit.

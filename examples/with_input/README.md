# with_input

Passes text to a program's stdin using `piston.Stdin(...)`.

```go
output, err := client.Execute(ctx, "python", "",
	[]piston.Code{
		{Content: "inp = input()\nprint(inp[::-1])"},
	},
	piston.Stdin("hello world"),
)
```

## Run

```
go run main.go
```

## Expected output

```
dlrow olleh
```

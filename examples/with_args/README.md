# with_args

Passes command line arguments to the executed program using `piston.Args(...)`.

```go
output, err := client.Execute(ctx, "python", "",
	[]piston.File{
		{Content: "import sys\nprint(sys.argv[1:])"},
	},
	piston.Args([]string{"foo", "bar"}),
)
```

## Run

```
go run main.go
```

## Expected output

```
['foo', 'bar']
```

# hello_world

The simplest possible example: execute an inline snippet of Python and print its output.

```go
client := piston.NewClient("http://localhost:2000/api/v2/")

output, err := client.Execute(context.Background(), "python", "",
	[]piston.Code{
		{Content: "print('Hello World')"},
	})
```

## Run

```
go run main.go
```

## Expected output

```
Output is:  Hello World
 Language used is:  python
```

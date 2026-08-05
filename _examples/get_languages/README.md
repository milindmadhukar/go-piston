# get_languages

Returns just the language names supported by the target Piston instance (a thin wrapper over `GetRuntimes`).

```go
languages := client.GetLanguages(context.Background())
```

## Run

```
go run main.go
```

## Expected output

```
Supported Languages by Piston are:  [python javascript c++ ...]
```

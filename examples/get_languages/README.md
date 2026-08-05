# get_languages

Returns the names of the languages the target instance supports, deduplicated — an instance with several versions of one language reports it once.

```go
languages, err := client.GetLanguages(context.Background())
```

## Run

```
go run main.go
```

## Expected output

```
Supported Languages by Piston are:  [python javascript c++ ...]
```

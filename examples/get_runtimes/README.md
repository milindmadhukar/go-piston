# get_runtimes

Lists every language, version, and set of aliases installed on the target instance. Several versions of the same language may be installed at once.

```go
runtimes, err := client.GetRuntimes(context.Background())
for _, runtime := range runtimes {
	log.Printf("%s %s (aliases: %v)", runtime.Language, runtime.Version, runtime.Aliases)
}
```

## Run

```
go run main.go
```

## Expected output

```
python 3.10.0 (aliases: [py py3 python3 python3.10])
javascript 18.15.0 (aliases: [node-javascript node-js javascript js])
```

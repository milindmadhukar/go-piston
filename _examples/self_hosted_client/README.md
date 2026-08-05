# self_hosted_client

Shows the recommended way to construct a client for a self-hosted Piston instance, including overriding the `*http.Client` (here, to set a request timeout) with `piston.WithHTTPClient`.

```go
client := piston.NewClient(
	"http://localhost:2000/api/v2/",
	piston.WithHTTPClient(&http.Client{Timeout: 10 * time.Second}),
)
```

See [official_api_client](../official_api_client) for using the official, key-gated Piston API instead.

## Run

```
go run main.go
```

## Expected output

```
Hello from a self-hosted instance
```

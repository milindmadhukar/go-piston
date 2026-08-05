# official_api_client

Shows how to use the official, key-gated Piston API via `piston.OfficialAPIBaseURL` and `piston.WithAPIKey`. See the top-level README's "Piston API access" section for how (and whether) to obtain a key — most projects should use a self-hosted instance instead, see [self_hosted_client](../self_hosted_client).

```go
client := piston.NewClient(
	piston.OfficialAPIBaseURL,
	piston.WithAPIKey(os.Getenv("PISTON_API_KEY")),
)
```

## Run

```
PISTON_API_KEY=your-key go run main.go
```

## Expected output

```
Hello from the official API
```

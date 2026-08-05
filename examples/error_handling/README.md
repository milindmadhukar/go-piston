# error_handling

Shows how to classify failures. A non-2xx response is returned as an `*piston.APIError` carrying the status code and the instance's own message; sentinel errors let you branch on the cause without parsing text.

```go
switch {
case errors.Is(err, piston.ErrAPIKeyRequired):
	// this instance needs a key; pass piston.WithAPIKey(...)
case errors.Is(err, piston.ErrRateLimited):
	// back off and retry
}

var apiErr *piston.APIError
if errors.As(err, &apiErr) {
	log.Printf("status=%d message=%q", apiErr.StatusCode, apiErr.Message)
}
```

Available sentinels: `ErrBadRequest`, `ErrUnauthorized`, `ErrAPIKeyRequired`, `ErrNotFound`, `ErrRateLimited`, `ErrServer`, `ErrLanguageNotFound` and `ErrUnsupportedByOfficialAPI`.

`ErrAPIKeyRequired` implies `ErrUnauthorized`, so check it first. It is derived from the client's own configuration rather than from the server's wording, which keeps it accurate if that wording changes.

## Run

```
go run main.go
```

## Expected output

Against a reachable instance with Python installed:

```
hello
```

Against the official API without a key:

```
this instance needs an API key; pass piston.WithAPIKey(...)
status=401 message="Public Piston API is now whitelist only as of 2/15/2026. ..."
```

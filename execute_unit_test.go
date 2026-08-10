package gopiston

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// fakeExecute stands up an instance whose /execute answers with the given body.
func fakeExecute(t *testing.T, body string) *Client {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)

	return NewClient(server.URL, WithHTTPClient(server.Client()))
}

// A language served by several engines resolves by version number, so the
// engine that ran the job is only knowable if the instance names it.
func TestExecutionCarriesRuntime(t *testing.T) {
	client := fakeExecute(t, `{
		"language": "javascript",
		"version": "1.3.14",
		"runtime": "bun",
		"run": {"stdout": "ok\n", "stderr": "", "output": "ok\n", "code": 0, "signal": null}
	}`)

	execution, err := client.Execute(context.Background(), "bun-js", "*", []File{{Content: "x"}})
	if err != nil {
		t.Fatal(err)
	}

	assert(execution.Language, "javascript", t)
	assert(execution.Runtime, "bun", t)
}

// Absence is the common case and the only thing an upstream instance can
// report, so it must decode to empty rather than failing.
func TestExecutionWithoutRuntime(t *testing.T) {
	client := fakeExecute(t, `{
		"language": "bash",
		"version": "5.2.0",
		"run": {"stdout": "ok\n", "stderr": "", "output": "ok\n", "code": 0, "signal": null}
	}`)

	execution, err := client.Execute(context.Background(), "bash", "*", []File{{Content: "x"}})
	if err != nil {
		t.Fatal(err)
	}

	assert(execution.Runtime, "", t)
}

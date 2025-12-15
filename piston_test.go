package gopiston

import (
	"context"
	"testing"
	"time"
)

var client = CreateDefaultClient()

func assert(expected, got interface{}, t *testing.T) {
	if expected != got {
		t.Errorf("Expected - %v, but got %v!", expected, got)
	}
}

func TestRuntimes(t *testing.T) {
	runtimes, err := client.GetRuntimes(context.Background())
	if err != nil {
		t.Error(err.Error())
	}
	for _, runtime := range *runtimes {
		if runtime.Language == "python" {
			assert(runtime.Aliases[0], "py", t)
		}
	}
}

func TestExecutionCode(t *testing.T) {
	execution, err := client.Execute(
		context.Background(), "python", "",
		[]Code{{Content: "print([i for i in range(4)])"}},
	)
	if err != nil {
		t.Errorf(err.Error())
	}

	assert(execution.GetOutput(), "[0, 1, 2, 3]\n", t)
}

func TestTimeout(t *testing.T) {
	response, err := client.Execute(
		context.Background(), "python", "",
		[]Code{
			{
				Name:    "main.py",
				Content: "import time\nprint('before sleep')\ntime.sleep(3)\nprint('after sleep')",
			},
		},
		RunTimeout(2*time.Second),
	)
	if err != nil {
		t.Errorf(err.Error())
	}
	assert(response.Run.Signal, "SIGKILL", t)
}

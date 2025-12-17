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

func TestCompileStage(t *testing.T) {
	// C++ usually has a compile stage
	execution, err := client.Execute(
		context.Background(), "c++", "",
		[]Code{{Content: "#include <iostream>\nint main() { std::cout << \"Hello\"; return 0; }"}},
	)
	if err != nil {
		t.Errorf(err.Error())
	}

	// Verify we got some compile output or at least the stage is present (though it might be empty if successful and silent)
	// Usually Piston returns code 0 for success.
	if execution.Compile.Code != 0 {
		t.Errorf("Expected compile code 0, got %d", execution.Compile.Code)
	}

	assert(execution.Run.Stdout, "Hello", t)
}

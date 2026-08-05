package gopiston

import (
	"context"
	"testing"
	"time"
)

func TestExecutionCode(t *testing.T) {
	execution, err := client.Execute(
		context.Background(), "python", "",
		[]Code{{Content: "print([i for i in range(4)])"}},
	)
	if err != nil {
		t.Fatal(err)
	}

	assert(execution.GetOutput(), "[0, 1, 2, 3]\n", t)
}

func TestExecutionWithArgs(t *testing.T) {
	execution, err := client.Execute(
		context.Background(), "python", "",
		[]Code{{Content: "import sys\nprint(sys.argv[1])"}},
		Args([]string{"hello-args"}),
	)
	if err != nil {
		t.Fatal(err)
	}

	assert(execution.GetOutput(), "hello-args\n", t)
}

func TestExecutionWithEmptyStdinDefault(t *testing.T) {
	execution, err := client.Execute(
		context.Background(), "python", "",
		[]Code{{Content: "import sys\nprint(repr(sys.stdin.read()))"}},
	)
	if err != nil {
		t.Fatal(err)
	}

	assert(execution.GetOutput(), "''\n", t)
}

func TestExecutionInvalidLanguage(t *testing.T) {
	_, err := client.Execute(
		context.Background(), "not-a-real-language", "1.0.0",
		[]Code{{Content: "print('hi')"}},
	)
	if err == nil {
		t.Errorf("Expected an error for an unsupported language, got nil")
	}
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
		t.Fatal(err)
	}
	assert(response.Run.Signal, "SIGKILL", t)
}

func TestRunMemoryLimit(t *testing.T) {
	response, err := client.Execute(
		context.Background(), "python", "",
		[]Code{
			{
				Name:    "main.py",
				Content: "x = bytearray(200 * 1024 * 1024)\nprint('allocated')",
			},
		},
		RunMemoryLimit(16*1024*1024),
	)
	if err != nil {
		t.Fatal(err)
	}
	if response.Run.Code == 0 {
		t.Errorf("Expected a non-zero exit code when exceeding the run memory limit")
	}
}

func TestCompileStage(t *testing.T) {
	// C++ usually has a compile stage
	execution, err := client.Execute(
		context.Background(), "c++", "",
		[]Code{{Content: "#include <iostream>\nint main() { std::cout << \"Hello\"; return 0; }"}},
	)
	if err != nil {
		t.Fatal(err)
	}

	// Verify we got some compile output or at least the stage is present (though it might be empty if successful and silent)
	// Usually Piston returns code 0 for success.
	if execution.Compile.Code != 0 {
		t.Errorf("Expected compile code 0, got %d", execution.Compile.Code)
	}

	assert(execution.Run.Stdout, "Hello", t)
}

func TestExecutionMultiFile(t *testing.T) {
	execution, err := client.Execute(
		context.Background(), "python", "",
		[]Code{
			{Name: "main.py", Content: "from helper import shout\nprint(shout('hi'))"},
			{Name: "helper.py", Content: "def shout(s):\n    return s.upper()"},
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	assert(execution.GetOutput(), "HI\n", t)
}

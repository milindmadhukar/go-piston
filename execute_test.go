package gopiston

import (
	"errors"
	"testing"
	"time"
)

func TestExecutionCode(t *testing.T) {
	requireLanguage(t, "python")

	execution, err := client.Execute(
		testContext(t), "python", "",
		[]Code{{Content: "print([i for i in range(4)])"}},
	)
	if err != nil {
		t.Fatal(err)
	}

	assert(execution.GetOutput(), "[0, 1, 2, 3]\n", t)
}

// An empty version must resolve to the highest installed version, not simply
// whichever the instance lists first.
func TestExecutionEmptyVersionUsesLatest(t *testing.T) {
	requireLanguage(t, "python")

	latest, err := client.GetLatestVersion(testContext(t), "python")
	if err != nil {
		t.Fatal(err)
	}

	execution, err := client.Execute(
		testContext(t), "python", "",
		[]Code{{Content: "print('hi')"}},
	)
	if err != nil {
		t.Fatal(err)
	}

	assert(execution.Version, latest, t)
}

func TestExecutionWithArgs(t *testing.T) {
	requireLanguage(t, "python")

	execution, err := client.Execute(
		testContext(t), "python", "",
		[]Code{{Content: "import sys\nprint(sys.argv[1])"}},
		Args([]string{"hello-args"}),
	)
	if err != nil {
		t.Fatal(err)
	}

	assert(execution.GetOutput(), "hello-args\n", t)
}

func TestExecutionWithEmptyStdinDefault(t *testing.T) {
	requireLanguage(t, "python")

	// Piston appends a trailing newline to stdin server-side if one isn't
	// already present, so even blank stdin is read back as "\n".
	execution, err := client.Execute(
		testContext(t), "python", "",
		[]Code{{Content: "import sys\nprint(repr(sys.stdin.read()))"}},
	)
	if err != nil {
		t.Fatal(err)
	}

	assert(execution.GetOutput(), "'\\n'\n", t)
}

func TestExecutionInvalidLanguage(t *testing.T) {
	requireExecuteAccess(t)

	// A pinned version skips the runtime lookup, so the instance itself
	// rejects the request.
	_, err := client.Execute(
		testContext(t), "not-a-real-language", "1.0.0",
		[]Code{{Content: "print('hi')"}},
	)
	if !errors.Is(err, ErrBadRequest) {
		t.Errorf("Expected ErrBadRequest, got %v", err)
	}
}

func TestTimeout(t *testing.T) {
	requireLanguage(t, "python")

	execution, err := client.Execute(
		testContext(t), "python", "",
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
	assert(execution.Run.Signal, "SIGKILL", t)
}

func TestRunMemoryLimit(t *testing.T) {
	requireLanguage(t, "python")

	execution, err := client.Execute(
		testContext(t), "python", "",
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
	if execution.Run.Code == 0 && execution.Run.Signal == "" {
		t.Errorf("Expected a failure when exceeding the run memory limit, got %+v", execution.Run)
	}
}

func TestCompileStage(t *testing.T) {
	requireLanguage(t, "c++")

	execution, err := client.Execute(
		testContext(t), "c++", "",
		[]Code{{Content: "#include <iostream>\nint main() { std::cout << \"Hello\"; return 0; }"}},
	)
	if err != nil {
		t.Fatal(err)
	}

	if execution.Compile == nil {
		t.Fatal("Expected a compile stage for c++")
	}
	if execution.Compile.Code != 0 {
		t.Errorf("Expected compile code 0, got %d", execution.Compile.Code)
	}

	assert(execution.Run.Stdout, "Hello", t)
}

// An interpreted language reports no compile stage at all, which must be
// distinguishable from a compile stage that exited zero.
func TestNoCompileStageForInterpretedLanguage(t *testing.T) {
	requireLanguage(t, "python")

	execution, err := client.Execute(
		testContext(t), "python", "",
		[]Code{{Content: "print('hi')"}},
	)
	if err != nil {
		t.Fatal(err)
	}

	if execution.Compile != nil {
		t.Errorf("Expected no compile stage for python, got %+v", execution.Compile)
	}
}

func TestExecutionMultiFile(t *testing.T) {
	requireLanguage(t, "python")

	execution, err := client.Execute(
		testContext(t), "python", "",
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

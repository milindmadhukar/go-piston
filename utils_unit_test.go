package gopiston

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"
)

// Piston requires file names to carry no path, so Files must send the base
// name even when given a nested or absolute path.
func TestFilesUsesBaseName(t *testing.T) {
	dir := t.TempDir()
	nested := filepath.Join(dir, "src")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(nested, "main.py")
	if err := os.WriteFile(path, []byte("print('hi')"), 0o644); err != nil {
		t.Fatal(err)
	}

	files, err := Files(path)
	if err != nil {
		t.Fatal(err)
	}

	assert(len(files), 1, t)
	assert(files[0].Name, "main.py", t)
	assert(files[0].Content, "print('hi')", t)
	assert(files[0].Encoding, "", t)
}

func TestFilesPreservesOrder(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"main.py", "helper.py"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	files, err := Files(filepath.Join(dir, "main.py"), filepath.Join(dir, "helper.py"))
	if err != nil {
		t.Fatal(err)
	}

	assert(len(files), 2, t)
	assert(files[0].Name, "main.py", t)
	assert(files[1].Name, "helper.py", t)
}

// Content that is not valid UTF-8 cannot be sent as a JSON string, so it is
// base64-encoded and tagged, matching the upstream CLI.
func TestFilesEncodesBinaryContent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "prog.bin")
	binary := []byte{0x7f, 'E', 'L', 'F', 0x00, 0xff, 0xfe}
	if err := os.WriteFile(path, binary, 0o644); err != nil {
		t.Fatal(err)
	}

	files, err := Files(path)
	if err != nil {
		t.Fatal(err)
	}

	assert(files[0].Encoding, "base64", t)
	assert(files[0].Content, base64.StdEncoding.EncodeToString(binary), t)
}

func TestFilesReportsMissingPath(t *testing.T) {
	if _, err := Files(filepath.Join(t.TempDir(), "nope.py")); err == nil {
		t.Fatal("Expected an error for a missing file")
	}
}

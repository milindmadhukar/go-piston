package gopiston

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"unicode/utf8"
)

// GetOutput returns the run stage's stdout and stderr, interleaved in the
// order the process produced them.
func (resp *PistonExecution) GetOutput() string {
	return resp.Run.Output
}

// Files reads the given paths and returns them as execution files, so callers
// can run code from disk instead of embedding it in a string. The files are
// returned in the order given, so the first path is the job's entry point.
//
// Only the base name of each path is sent: Piston requires file names to
// carry no path, and would otherwise be asked to create subdirectories.
// Content that is not valid UTF-8 is base64-encoded, which allows binaries to
// be sent to runtimes that accept them.
func Files(paths ...string) ([]Code, error) {
	files := make([]Code, 0, len(paths))

	for _, path := range paths {
		content, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("piston: read %s: %w", path, err)
		}

		file := Code{Name: filepath.Base(path)}
		if utf8.Valid(content) {
			file.Content = string(content)
		} else {
			file.Content = base64.StdEncoding.EncodeToString(content)
			file.Encoding = "base64"
		}

		files = append(files, file)
	}

	return files, nil
}

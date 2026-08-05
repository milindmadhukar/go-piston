package gopiston

import (
	"context"
	"testing"
)

func TestRuntimes(t *testing.T) {
	runtimes, err := client.GetRuntimes(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, runtime := range *runtimes {
		if runtime.Language == "python" {
			assert(runtime.Aliases[0], "py", t)
		}
	}
}

func TestGetLanguages(t *testing.T) {
	languages := client.GetLanguages(context.Background())
	if languages == nil || len(*languages) == 0 {
		t.Fatal("Expected a non-empty list of languages")
	}

	found := false
	for _, language := range *languages {
		if language == "python" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected \"python\" to be in the list of supported languages")
	}
}

func TestGetLatestVersion(t *testing.T) {
	version, err := client.GetLatestVersion(context.Background(), "python")
	if err != nil {
		t.Error(err.Error())
	}
	if version == "" {
		t.Errorf("Expected a non-empty version string for python")
	}
}

func TestGetLatestVersionInvalidLanguage(t *testing.T) {
	_, err := client.GetLatestVersion(context.Background(), "not-a-real-language")
	if err == nil {
		t.Errorf("Expected an error for an unsupported language, got nil")
	}
}

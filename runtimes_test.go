package gopiston

import (
	"errors"
	"slices"
	"testing"
)

func TestRuntimes(t *testing.T) {
	runtimes, err := client.GetRuntimes(testContext(t))
	if err != nil {
		t.Fatal(err)
	}
	for _, runtime := range runtimes {
		if runtime.Language == "python" {
			assert(slices.Contains(runtime.Aliases, "py"), true, t)
		}
	}
}

func TestGetLanguages(t *testing.T) {
	requireLanguage(t, "python")

	languages, err := client.GetLanguages(testContext(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(languages) == 0 {
		t.Fatal("Expected a non-empty list of languages")
	}

	if !slices.Contains(languages, "python") {
		t.Errorf("Expected %q to be in the list of supported languages", "python")
	}

	// An instance with several versions of one language must report it once.
	seen := make(map[string]bool, len(languages))
	for _, language := range languages {
		if seen[language] {
			t.Errorf("Expected language %q to appear only once", language)
		}
		seen[language] = true
	}
}

func TestGetLatestVersion(t *testing.T) {
	requireLanguage(t, "python")

	version, err := client.GetLatestVersion(testContext(t), "python")
	if err != nil {
		t.Fatal(err)
	}
	if version == "" {
		t.Errorf("Expected a non-empty version string for python")
	}

	// Whatever it returns must genuinely be the highest installed version,
	// not merely the first one the instance happened to list.
	runtimes, err := client.GetRuntimes(testContext(t))
	if err != nil {
		t.Fatal(err)
	}
	for _, runtime := range runtimes {
		if runtime.Language != "python" {
			continue
		}
		if compareVersions(runtime.Version, version) > 0 {
			t.Errorf("GetLatestVersion returned %s, but %s is installed", version, runtime.Version)
		}
	}
}

func TestGetLatestVersionInvalidLanguage(t *testing.T) {
	_, err := client.GetLatestVersion(testContext(t), "not-a-real-language")
	if !errors.Is(err, ErrLanguageNotFound) {
		t.Errorf("Expected ErrLanguageNotFound, got %v", err)
	}
}

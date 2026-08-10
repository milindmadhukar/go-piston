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
		if CompareVersions(runtime.Version, version) > 0 {
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

// An engine's own alias is the only stable way to ask for it: a bare language
// name resolves by version number, and versions do not compare across engines.
//
// The check that works everywhere is the version that comes back — asking for
// deno must answer with deno's version, not with node's higher one — because
// naming the engine in an execute response is newer than the /runtimes field
// that distinguishes them, so it is asserted only where it is reported.
//
// Skipped unless the instance really has two engines for one language.
func TestExecuteSelectsAnEngineByAlias(t *testing.T) {
	requireExecuteAccess(t)

	runtimes, err := client.GetRuntimes(testContext(t))
	if err != nil {
		t.Fatal(err)
	}

	// language -> engine -> that engine's runtimes for it.
	shared := map[string]map[string][]Runtime{}
	for _, runtime := range runtimes {
		if runtime.Runtime == "" {
			continue
		}
		if shared[runtime.Language] == nil {
			shared[runtime.Language] = map[string][]Runtime{}
		}
		shared[runtime.Language][runtime.Runtime] = append(
			shared[runtime.Language][runtime.Runtime], runtime)
	}

	tested := 0
	for language, engines := range shared {
		if len(engines) < 2 {
			continue
		}

		for engine, installed := range engines {
			alias, latest := exclusiveAlias(installed, engine, engines)
			if alias == "" {
				t.Errorf("%s on %s has no alias of its own, so it cannot be asked for", language, engine)
				continue
			}

			// An empty program: the assertion is about which engine answered,
			// not about what it did, and every engine accepts an empty file.
			execution, err := client.Execute(testContext(t), alias, "*", []File{{Content: ""}})
			if err != nil {
				t.Fatal(err)
			}
			assert(execution.Language, language, t)
			assert(execution.Version, latest, t)
			if execution.Runtime != "" {
				assert(execution.Runtime, engine, t)
			}
			tested++
		}
	}

	if tested == 0 {
		t.Skip("skipping: no language on this instance is served by more than one engine")
	}
}

// exclusiveAlias returns an alias belonging to engine alone, along with the
// highest version installed for it — what asking by that alias should resolve
// to. The alias is empty when every name this engine answers to is also
// claimed by another engine of the same language.
func exclusiveAlias(installed []Runtime, engine string, engines map[string][]Runtime) (string, string) {
	others := map[string]bool{}
	for name, runtimes := range engines {
		if name == engine {
			continue
		}
		for _, runtime := range runtimes {
			for _, alias := range runtime.Aliases {
				others[alias] = true
			}
		}
	}

	latest := ""
	for _, runtime := range installed {
		if latest == "" || CompareVersions(runtime.Version, latest) > 0 {
			latest = runtime.Version
		}
	}

	for _, runtime := range installed {
		for _, alias := range runtime.Aliases {
			if !others[alias] {
				return alias, latest
			}
		}
	}
	return "", latest
}

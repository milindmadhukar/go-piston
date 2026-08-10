package gopiston

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"
)

// GetRuntimes returns the runtimes the instance supports, each with its
// language, version and aliases. To execute code for a particular language
// with Execute, pass either the language name or one of its aliases. Several
// versions of the same language may be installed at once.
func (client *Client) GetRuntimes(ctx context.Context) ([]Runtime, error) {
	body, err := client.doRequest(ctx, http.MethodGet, client.endpoint("runtimes"), nil)
	if err != nil {
		return nil, err
	}

	var runtimes []Runtime
	if err := decodeJSON(body, &runtimes); err != nil {
		return nil, err
	}

	return runtimes, nil
}

// GetLanguages returns the names of the languages the instance supports.
// Names are deduplicated: an instance with several versions of the same
// language reports that language once.
func (client *Client) GetLanguages(ctx context.Context) ([]string, error) {
	runtimes, err := client.GetRuntimes(ctx)
	if err != nil {
		return nil, err
	}

	languages := make([]string, 0, len(runtimes))
	for _, runtime := range runtimes {
		if !slices.Contains(languages, runtime.Language) {
			languages = append(languages, runtime.Language)
		}
	}

	return languages, nil
}

// GetLatestVersion returns the highest installed version of the given
// language, which may be referred to by name or by any of its aliases.
//
// Execute resolves an empty version server-side, so calling this first is only
// necessary when the version itself is of interest.
//
// It compares across engines, exactly as the instance does. Asking for the
// latest "javascript" on an instance running both node 20 and bun 1.3 answers
// 20.11.1 — the version of whichever engine numbers highest, not a judgement
// about which is newer. Pass an engine's own alias ("bun-js") to scope it.
func (client *Client) GetLatestVersion(ctx context.Context, language string) (string, error) {
	runtimes, err := client.GetRuntimes(ctx)
	if err != nil {
		return "", err
	}

	latest := ""
	for _, runtime := range runtimes {
		if language != runtime.Language && !slices.Contains(runtime.Aliases, language) {
			continue
		}
		// The API makes no ordering guarantee and instances routinely expose
		// several versions of one language, so compare rather than taking the
		// first match.
		if latest == "" || CompareVersions(runtime.Version, latest) > 0 {
			latest = runtime.Version
		}
	}

	if latest == "" {
		return "", fmt.Errorf("piston: no runtime installed for language %q: %w", language, ErrLanguageNotFound)
	}

	return latest, nil
}

// CompareVersions orders two dotted version strings numerically, returning a
// negative number if a sorts before b, zero if they are equal, and a positive
// number otherwise. Components that are not numeric fall back to a string
// comparison, which keeps the ordering total for unusual version strings.
//
// It is what GetLatestVersion uses to pick a winner, and is exported so that a
// caller sorting the versions of one language for display agrees with it.
// Piston versions are plain dotted numbers in practice, not full semver.
func CompareVersions(a, b string) int {
	aParts := strings.Split(a, ".")
	bParts := strings.Split(b, ".")

	for i := 0; i < len(aParts) || i < len(bParts); i++ {
		// A component past the end of one version is treated as zero, so
		// "1.0" and "1.0.0" compare equal.
		aPart, bPart := "0", "0"
		if i < len(aParts) {
			aPart = aParts[i]
		}
		if i < len(bParts) {
			bPart = bParts[i]
		}

		aNum, aErr := strconv.Atoi(aPart)
		bNum, bErr := strconv.Atoi(bPart)
		if aErr == nil && bErr == nil {
			if aNum != bNum {
				return aNum - bNum
			}
			continue
		}

		if cmp := strings.Compare(aPart, bPart); cmp != 0 {
			return cmp
		}
	}

	return 0
}

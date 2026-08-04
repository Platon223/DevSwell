package github

import (
	"regexp"
	"testing"
)

func TestParseVersion(t *testing.T) {
	cases := []struct {
		tag  string
		want []int
		ok   bool
	}{
		{"go1.23.4", []int{1, 23, 4}, true},
		{"v2.0.1-beta", []int{2, 0, 1}, true},
		{"1.2.3", []int{1, 2, 3}, true},
		{"weekly.2012-03-27", nil, false}, // a bare number ("2012") is rejected as too ambiguous to be a real version
		{"no-version-here", nil, false},
		{"jdk-28+9", []int{28, 9}, true},  // '+' separator (OpenJDK's scheme)
		{"REL_18_4", []int{18, 4}, true},  // '_' separator (Postgres's scheme)
	}
	for _, c := range cases {
		got, ok := parseVersion(c.tag)
		if ok != c.ok {
			t.Errorf("parseVersion(%q) ok = %v, want %v", c.tag, ok, c.ok)
			continue
		}
		if !ok {
			continue
		}
		if len(got) != len(c.want) {
			t.Errorf("parseVersion(%q) = %v, want %v", c.tag, got, c.want)
			continue
		}
		for i := range got {
			if got[i] != c.want[i] {
				t.Errorf("parseVersion(%q) = %v, want %v", c.tag, got, c.want)
				break
			}
		}
	}
}

func TestCompareVersionsIsNumericNotLexicographic(t *testing.T) {
	// The whole point of this fix: "go1.9" must be LESS than "go1.10" even
	// though the string "9" is lexicographically greater than "1".
	v9, ok := parseVersion("go1.9")
	if !ok {
		t.Fatal("expected go1.9 to parse")
	}
	v10, ok := parseVersion("go1.10")
	if !ok {
		t.Fatal("expected go1.10 to parse")
	}
	if compareVersions(v9, v10) >= 0 {
		t.Fatalf("expected go1.9 < go1.10 numerically, got compareVersions = %d", compareVersions(v9, v10))
	}
}

func TestCompareVersionsHandlesMissingTrailingSegments(t *testing.T) {
	v1, _ := parseVersion("v1.2")
	v2, _ := parseVersion("v1.2.0")
	if compareVersions(v1, v2) != 0 {
		t.Fatalf("expected 1.2 == 1.2.0, got %d", compareVersions(v1, v2))
	}
}

func TestFetchLatestTagPicksHighestVersionNotFirstInList(t *testing.T) {
	// Simulates the real golang/go scenario: the API returns an old
	// "weekly.*" tag before modern "go1.x" tags. The picker must not just
	// take the first result.
	tags := []tagEntry{
		{Name: "weekly.2012-03-27"},
		{Name: "go1.9.0"},
		{Name: "go1.23.4"},
		{Name: "go1.10.0"},
		{Name: "not-a-version"},
	}

	var best tagEntry
	var bestVersion []int
	found := false
	for _, tg := range tags {
		v, ok := parseVersion(tg.Name)
		if !ok {
			continue
		}
		if !found || compareVersions(v, bestVersion) > 0 {
			best = tg
			bestVersion = v
			found = true
		}
	}

	if !found {
		t.Fatal("expected to find a version-like tag")
	}
	if best.Name != "go1.23.4" {
		t.Fatalf("expected go1.23.4 to be picked as the highest version, got %q", best.Name)
	}
}

// pickBest duplicates FetchLatestTag's selection loop so it can be tested
// without a live network call.
func pickBest(tags []tagEntry, namePattern *regexp.Regexp) (tagEntry, bool) {
	var best tagEntry
	var bestVersion []int
	found := false
	for _, tg := range tags {
		if namePattern != nil && !namePattern.MatchString(tg.Name) {
			continue
		}
		v, ok := parseVersion(tg.Name)
		if !ok {
			continue
		}
		if !found || compareVersions(v, bestVersion) > 0 {
			best = tg
			bestVersion = v
			found = true
		}
	}
	return best, found
}

func TestFetchLatestTagWithoutPatternIsFooledByLegacyScheme(t *testing.T) {
	// The real bug found against golang/go: pre-1.0 Go used tags like
	// "release.r60.3", which parses to [60, 3] — numerically bigger than a
	// real modern "go1.27" ([1, 27]), so an unfiltered picker gets fooled
	// into "confidently" returning a 2011 tag as the latest.
	tags := []tagEntry{
		{Name: "release.r60.3"},
		{Name: "go1.27rc2"},
	}

	best, found := pickBest(tags, nil)
	if !found {
		t.Fatal("expected a match")
	}
	if best.Name != "release.r60.3" {
		t.Fatalf("expected this test to demonstrate the bug (picking the legacy tag), got %q instead — has the underlying comparison logic changed?", best.Name)
	}
}

func TestFetchLatestTagWithPatternAvoidsLegacyScheme(t *testing.T) {
	// Same tag set as above, but with the per-repo name pattern DevSwell
	// actually uses for golang/go — this is the fix.
	tags := []tagEntry{
		{Name: "release.r60.3"},
		{Name: "go1.27rc2"},
	}
	goPattern := regexp.MustCompile(`^go1\.`)

	best, found := pickBest(tags, goPattern)
	if !found {
		t.Fatal("expected a match")
	}
	if best.Name != "go1.27rc2" {
		t.Fatalf("expected go1.27rc2 to be picked with the go1. prefix filter, got %q", best.Name)
	}
}

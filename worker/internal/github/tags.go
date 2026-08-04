package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"time"
)

// versionPattern matches a run of digit groups separated by '.', '_', or '+'
// — the separators real projects actually use (e.g. "go1.23.4" uses '.',
// "jdk-28+9" uses '+', "REL_18_4" uses '_').
var versionPattern = regexp.MustCompile(`\d+(?:[._+]\d+)*`)
var separatorPattern = regexp.MustCompile(`[._+]`)

// parseVersion extracts the first run of separator-joined numbers from a tag
// name (e.g. "go1.23.4" -> [1,23,4], "jdk-28+9" -> [28,9]). Tags with no
// recognizable multi-segment version number are skipped by FetchLatestTag.
func parseVersion(tag string) ([]int, bool) {
	match := versionPattern.FindString(tag)
	if match == "" {
		return nil, false
	}
	parts := separatorPattern.Split(match, -1)
	if len(parts) < 2 {
		// Reject bare single numbers (e.g. a year embedded in a date-based
		// tag like "weekly.2012-03-27" would otherwise parse as [2012], which
		// numerically outranks a real version like [1,23,4] — too ambiguous
		// to trust as a genuine version identifier.
		return nil, false
	}
	nums := make([]int, len(parts))
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			return nil, false
		}
		nums[i] = n
	}
	return nums, true
}

// compareVersions returns -1, 0, or 1 as a compares less than, equal to, or
// greater than b, treating missing trailing segments as 0 (1.2 == 1.2.0).
func compareVersions(a, b []int) int {
	for i := 0; i < len(a) || i < len(b); i++ {
		var av, bv int
		if i < len(a) {
			av = a[i]
		}
		if i < len(b) {
			bv = b[i]
		}
		if av != bv {
			if av < bv {
				return -1
			}
			return 1
		}
	}
	return 0
}

type tagEntry struct {
	Name   string `json:"name"`
	Commit struct {
		SHA string `json:"sha"`
	} `json:"commit"`
}

// maxTagPages bounds how many pages of tags FetchLatestTag scans. GitHub's
// tags list order is not chronological and can vary wildly by repo (e.g.
// golang/go's first page is 100 decade-old "weekly.*" tags with zero real
// go1.x versions — those only start appearing on page 2). This is a
// best-effort bound, not a guarantee: a repo whose true latest tag sits
// beyond this many pages would still be missed.
const maxTagPages = 5

// FetchLatestTag is a fallback for repos that don't use GitHub's Releases
// feature at all (many major projects — Go, Python, Linux, Postgres,
// MongoDB, MySQL, Git, and others — version via plain git tags instead, with
// no formal Release entries). It scans up to maxTagPages of tags and picks
// the one with the highest parsed version number, since the tags API's list
// order is not chronological.
//
// namePattern restricts which tags are even considered, and is required for
// most repos: a purely generic "biggest number wins" comparison is not safe
// on its own — e.g. golang/go's pre-1.0 tags like "release.r60.3" parse to a
// numerically larger value than real "go1.27" tags, despite being from 2011.
// Pass nil only for repos empirically confirmed to have a single consistent
// tag scheme with no such legacy trap.
func FetchLatestTag(ctx context.Context, owner, repo string, namePattern *regexp.Regexp) (*Release, error) {
	var best tagEntry
	var bestVersion []int
	found := false

	for page := 1; page <= maxTagPages; page++ {
		url := fmt.Sprintf("https://api.github.com/repos/%s/%s/tags?per_page=100&page=%d", owner, repo, page)
		req, err := newRequest(ctx, url)
		if err != nil {
			return nil, err
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("requesting tags page %d: %w", page, err)
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			return nil, fmt.Errorf("github api returned %s for %s/%s tags page %d", resp.Status, owner, repo, page)
		}

		var tags []tagEntry
		err = json.NewDecoder(resp.Body).Decode(&tags)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("decoding tags response page %d: %w", page, err)
		}

		for _, t := range tags {
			if namePattern != nil && !namePattern.MatchString(t.Name) {
				continue
			}
			v, ok := parseVersion(t.Name)
			if !ok {
				continue
			}
			if !found || compareVersions(v, bestVersion) > 0 {
				best = t
				bestVersion = v
				found = true
			}
		}

		if len(tags) < 100 {
			break // reached the last page
		}
	}
	if !found {
		return nil, nil
	}

	publishedAt, err := fetchCommitDate(ctx, owner, repo, best.Commit.SHA)
	if err != nil {
		return nil, fmt.Errorf("fetching commit date for tag %s: %w", best.Name, err)
	}

	return &Release{
		Name:        best.Name,
		TagName:     best.Name,
		URL:         fmt.Sprintf("https://github.com/%s/%s/releases/tag/%s", owner, repo, best.Name),
		PublishedAt: publishedAt,
		Body:        "", // tags have no changelog text, unlike formal Releases
	}, nil
}

func fetchCommitDate(ctx context.Context, owner, repo, sha string) (time.Time, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/commits/%s", owner, repo, sha)
	req, err := newRequest(ctx, url)
	if err != nil {
		return time.Time{}, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return time.Time{}, fmt.Errorf("requesting commit: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return time.Time{}, fmt.Errorf("github api returned %s for commit %s", resp.Status, sha)
	}

	var raw struct {
		Commit struct {
			Committer struct {
				Date time.Time `json:"date"`
			} `json:"committer"`
		} `json:"commit"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return time.Time{}, fmt.Errorf("decoding commit response: %w", err)
	}
	return raw.Commit.Committer.Date, nil
}

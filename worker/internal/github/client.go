package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

type Release struct {
	Name        string
	TagName     string
	URL         string
	PublishedAt time.Time
	Body        string
}

// newRequest builds a GitHub API GET request with standard headers, applying
// GITHUB_TOKEN if set (optional — raises the rate limit, not required for
// public repos).
func newRequest(ctx context.Context, url string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "DevSwell-Worker")
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return req, nil
}

// FetchLatestRelease pulls the most recent published, non-prerelease, non-draft
// release for owner/repo. Returns nil, nil if the repo has no releases.
func FetchLatestRelease(ctx context.Context, owner, repo string) (*Release, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", owner, repo)

	req, err := newRequest(ctx, url)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("requesting latest release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github api returned %s for %s/%s", resp.Status, owner, repo)
	}

	var raw struct {
		Name        string    `json:"name"`
		TagName     string    `json:"tag_name"`
		HTMLURL     string    `json:"html_url"`
		PublishedAt time.Time `json:"published_at"`
		Body        string    `json:"body"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return &Release{
		Name:        raw.Name,
		TagName:     raw.TagName,
		URL:         raw.HTMLURL,
		PublishedAt: raw.PublishedAt,
		Body:        raw.Body,
	}, nil
}

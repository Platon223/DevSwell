package main

import (
	"testing"
	"time"

	"github.com/Platon223/DevSwell/worker/internal/classifier"
	"github.com/Platon223/DevSwell/worker/internal/github"
)

func TestNewsItemFromReleaseAttachesSeverity(t *testing.T) {
	breaking := github.Release{
		Name:        "v3.0.0",
		TagName:     "v3.0.0",
		URL:         "https://example.com/v3.0.0",
		PublishedAt: time.Now(),
		Body:        "BREAKING CHANGE: removed the old config format.",
	}
	item := newsItemFromRelease("example/project", "Go", breaking)
	if item.Severity != string(classifier.SeverityBreaking) {
		t.Fatalf("expected breaking severity, got %q", item.Severity)
	}

	patch := github.Release{
		Name:        "v3.0.1",
		TagName:     "v3.0.1",
		URL:         "https://example.com/v3.0.1",
		PublishedAt: time.Now(),
		Body:        "Fixed a typo in the docs.",
	}
	item = newsItemFromRelease("example/project", "Go", patch)
	if item.Severity != string(classifier.SeverityPatch) {
		t.Fatalf("expected patch severity, got %q", item.Severity)
	}
}

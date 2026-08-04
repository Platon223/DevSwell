package classifier

import (
	"regexp"
	"strings"
)

type Severity string

const (
	SeveritySecurity Severity = "security"
	SeverityBreaking Severity = "breaking"
	SeverityPatch    Severity = "patch"
)

var securityKeywords = []string{"cve-", "security", "vulnerability", "exploit"}

// breakingKeywords are deliberately specific phrases, not bare words like
// "removed" or "deprecated" — those caused false positives on any unrelated
// commit message that happened to contain them (e.g. a real release,
// nodejs/node v26.6.0, was flagged "breaking" purely because one routine
// internal fix in its (very long) commit list was titled "avoid retaining
// removed event names" — a memory-leak fix, not an API removal).
var breakingKeywords = []string{
	"breaking change",
	"breaking:",
	"no longer supported",
	"no longer works",
	"will no longer",
	"has been removed",
	"have been removed",
	"### breaking",
	"## breaking",
	"### removed",
	"## removed",
	"### deprecated",
	"## deprecated",
}

// highlightsHeading matches a release's curated summary section, if it has
// one (e.g. Node.js's "### Notable Changes").
var highlightsHeading = regexp.MustCompile(`(?i)#{1,4}\s*(notable changes|highlights)\b`)

// nextHeading matches the start of the next markdown section, used to find
// where a highlights section ends.
var nextHeading = regexp.MustCompile(`(?m)^#{1,4}\s+\S`)

// Classify assigns a severity based on keyword matches.
//
// Security keywords are checked against the full title+body: a false
// negative there (missing a real security issue) is worse than a false
// positive, so we stay maximally sensitive.
//
// Breaking keywords are checked only against a release's curated
// "highlights" section when one exists, falling back to the full body
// otherwise. Scanning every individual commit message in a large release is
// exactly what caused the false positive described above — a curated
// summary section is a much cleaner signal when the project provides one.
func Classify(title, body string) Severity {
	fullText := strings.ToLower(title + " " + body)

	for _, kw := range securityKeywords {
		if strings.Contains(fullText, kw) {
			return SeveritySecurity
		}
	}

	breakingText := strings.ToLower(title + " " + extractHighlights(body))
	for _, kw := range breakingKeywords {
		if strings.Contains(breakingText, kw) {
			return SeverityBreaking
		}
	}
	return SeverityPatch
}

// extractHighlights returns the text of a release's curated summary section
// (e.g. "### Notable Changes") up to the next heading, or the full body
// unchanged if no such section is found.
func extractHighlights(body string) string {
	loc := highlightsHeading.FindStringIndex(body)
	if loc == nil {
		return body
	}
	rest := body[loc[1]:]
	if end := nextHeading.FindStringIndex(rest); end != nil {
		return rest[:end[0]]
	}
	return rest
}

package classifier

import "strings"

type Severity string

const (
	SeveritySecurity Severity = "security"
	SeverityBreaking Severity = "breaking"
	SeverityPatch    Severity = "patch"
)

var securityKeywords = []string{"cve-", "security", "vulnerability", "exploit"}
var breakingKeywords = []string{"breaking change", "breaking:", "removed", "no longer supported", "deprecated"}

// Classify assigns a severity to a release based on keyword matches in its
// title and body. Security keywords take priority over breaking-change
// keywords; anything matching neither is treated as a routine patch.
func Classify(title, body string) Severity {
	text := strings.ToLower(title + " " + body)

	for _, kw := range securityKeywords {
		if strings.Contains(text, kw) {
			return SeveritySecurity
		}
	}
	for _, kw := range breakingKeywords {
		if strings.Contains(text, kw) {
			return SeverityBreaking
		}
	}
	return SeverityPatch
}

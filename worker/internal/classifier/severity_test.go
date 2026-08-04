package classifier

import "testing"

func TestClassify(t *testing.T) {
	cases := []struct {
		name  string
		title string
		body  string
		want  Severity
	}{
		{
			name:  "breaking change release note",
			title: "v3.0.0",
			body:  "BREAKING CHANGE: removed the old config format. Migrate before upgrading.",
			want:  SeverityBreaking,
		},
		{
			name:  "patch release note",
			title: "v1.0.1",
			body:  "Fixed a typo in the docs.",
			want:  SeverityPatch,
		},
		{
			name:  "security release note",
			title: "v1.2.4",
			body:  "This release fixes a security vulnerability (CVE-2026-1234).",
			want:  SeveritySecurity,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := Classify(c.title, c.body)
			if got != c.want {
				t.Fatalf("Classify(%q, %q) = %q, want %q", c.title, c.body, got, c.want)
			}
		})
	}
}

// TestClassifyIgnoresIncidentalKeywordsOutsideHighlights reproduces the real
// false positive found in production: nodejs/node v26.6.0 was flagged
// "breaking" solely because one unrelated internal commit in its (very
// long) flat commit list was titled "avoid retaining removed event names" —
// a memory-leak fix, not an API removal. The release's actual curated
// summary ("Notable Changes") only lists new backward-compatible additions.
func TestClassifyIgnoresIncidentalKeywordsOutsideHighlights(t *testing.T) {
	title := "2026-08-03, Version 26.6.0 (Current), @aduh95"
	body := `
### Notable Changes

* **ffi**: add ` + "`getCurrentEventLoop`" + ` (Paolo Insogna)
* **test_runner**: add ` + "`context.log()`" + ` and ` + "`test:log`" + ` event (Moshe Atlow)

### Commits

* **events**: avoid retaining removed event names (Matteo Collina)
* **buffer**: normalize lone "\r" in Blob native line endings (Daijiro Wachi)
* **crypto**: preserve RSA-PSS legacy pubkey DER (Filip Skokan)
`
	got := Classify(title, body)
	if got != SeverityPatch {
		t.Fatalf("Classify(...) = %q, want %q (the word \"removed\" in an unrelated commit outside Notable Changes should not trigger breaking)", got, SeverityPatch)
	}
}

// TestClassifyStillCatchesRealBreakingChangeInHighlights confirms the fix
// above doesn't just suppress the breaking category entirely — a genuine
// breaking change announced within the highlights section is still caught.
func TestClassifyStillCatchesRealBreakingChangeInHighlights(t *testing.T) {
	title := "v5.0.0"
	body := `
### Notable Changes

* BREAKING CHANGE: the old config format has been removed. Migrate before upgrading.

### Commits

* **docs**: fix typo (Jane Doe)
`
	got := Classify(title, body)
	if got != SeverityBreaking {
		t.Fatalf("Classify(...) = %q, want %q (a real breaking change inside Notable Changes should still be caught)", got, SeverityBreaking)
	}
}

// TestClassifyFallsBackToFullBodyWithoutHighlightsSection confirms repos
// with no curated summary section (most releases) still get scanned in full.
func TestClassifyFallsBackToFullBodyWithoutHighlightsSection(t *testing.T) {
	title := "v2.0.0"
	body := "This release: BREAKING CHANGE: the legacy API has been removed."
	got := Classify(title, body)
	if got != SeverityBreaking {
		t.Fatalf("Classify(...) = %q, want %q (no Notable Changes heading present, should fall back to scanning the full body)", got, SeverityBreaking)
	}
}

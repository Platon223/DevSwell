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

package main

import "regexp"

// trackedRepo describes one repo the worker ingests. TagPattern is only used
// as a fallback when the repo has no formal GitHub Releases (see
// github.FetchLatestTag) — most repos leave it nil since GitHub Releases
// works directly. It's required for repos whose tag history mixes multiple
// incompatible naming schemes (e.g. golang/go used "release.rXX.X" before
// switching to "go1.X" — a generic "biggest number wins" comparison would
// otherwise pick the ancient tag).
type trackedRepo struct {
	Owner      string
	Name       string
	Technology string
	TagPattern *regexp.Regexp
}

var trackedRepos = []trackedRepo{
	// Languages
	{"golang", "go", "Go", regexp.MustCompile(`^go1\.`)},
	{"nodejs", "node", "JavaScript", nil},
	{"microsoft", "TypeScript", "TypeScript", nil},
	{"python", "cpython", "Python", regexp.MustCompile(`^v3\.`)},
	{"rust-lang", "rust", "Rust", nil},
	{"openjdk", "jdk", "Java", regexp.MustCompile(`^jdk-`)},
	{"dotnet", "roslyn", "C#", nil},
	{"ruby", "ruby", "Ruby", nil},
	{"php", "php-src", "PHP", nil},
	{"swiftlang", "swift", "Swift", nil},
	{"JetBrains", "kotlin", "Kotlin", nil},
	{"dart-lang", "sdk", "Dart", regexp.MustCompile(`^\d`)},
	{"elixir-lang", "elixir", "Elixir", nil},
	{"scala", "scala", "Scala", nil},

	// Frameworks
	{"react", "react", "React", nil},
	{"vuejs", "core", "Vue.js", nil},
	{"angular", "angular", "Angular", nil},
	{"vercel", "next.js", "Next.js", nil},
	{"nuxt", "nuxt", "Nuxt", nil},
	{"sveltejs", "svelte", "Svelte", nil},
	{"django", "django", "Django", nil},
	{"pallets", "flask", "Flask", nil},
	{"fastapi", "fastapi", "FastAPI", nil},
	{"expressjs", "express", "Express", nil},
	{"spring-projects", "spring-boot", "Spring", nil},
	{"rails", "rails", "Ruby on Rails", nil},
	{"laravel", "laravel", "Laravel", nil},
	{"dotnet", "runtime", ".NET", nil},

	// Databases (MySQL skipped — its highest-numbered tag, mysql-26.7.0, is a
	// very recent tag whose commit message is about CI/automation tooling,
	// not a release; looks like test/automation noise rather than a real
	// version. Revisit with a more careful look later.)
	{"mongodb", "mongo", "MongoDB", regexp.MustCompile(`^r\d`)},
	{"postgres", "postgres", "PostgreSQL", regexp.MustCompile(`^REL_\d+_`)},
	{"MariaDB", "server", "MariaDB", nil},
	{"redis", "redis", "Redis", nil},
	{"sqlite", "sqlite", "SQLite", regexp.MustCompile(`^version-`)},
	{"elastic", "elasticsearch", "Elasticsearch", nil},
	{"apache", "cassandra", "Cassandra", regexp.MustCompile(`^cassandra-\d`)},
	{"firebase", "firebase-tools", "Firebase", nil},
	{"supabase", "supabase", "Supabase", nil},

	// Tools & Platforms (GitHub, Google Cloud, Cloudflare, GitLab skipped —
	// no clean single repo / tag data too unreliable, see roadmap memory)
	{"moby", "moby", "Docker", nil},
	{"kubernetes", "kubernetes", "Kubernetes", nil},
	{"git", "git", "Git", nil},
	{"hashicorp", "terraform", "Terraform", nil},
	{"digitalocean", "doctl", "DigitalOcean", nil},
	{"vercel", "vercel", "Vercel", nil},
	{"netlify", "cli", "Netlify", nil},
	{"nginx", "nginx", "Nginx", nil},
	{"torvalds", "linux", "Linux", regexp.MustCompile(`^v\d`)},
	{"webpack", "webpack", "Webpack", nil},
	{"vitejs", "vite", "Vite", nil},
	{"graphql", "graphql-js", "GraphQL", nil},
}

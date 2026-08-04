# DevSwell

A personalized daily digest for the technologies you actually use. Pick your
stack — languages, frameworks, databases, tools — and DevSwell tracks their
real GitHub releases, classifies each one (security / breaking / patch), and
emails you a digest when something changes. There's also a separate,
unpersonalized Hacker News feed for general browsing.

This is a solo commercial project. The code is public for portfolio and
transparency purposes — see [`LICENSE`](./LICENSE) for what that does and
doesn't permit.

## Architecture

Two independently deployable Go services share one module and one MongoDB
database, but no code:

- **`worker/`** — runs on its own schedule, pulls the latest release (or,
  for projects without formal GitHub Releases, the latest version tag) for
  every tracked repo, classifies severity via keyword rules, upserts into
  Mongo, and emails affected users when something in their stack changed.
  It never talks to the API and has no HTTP surface of its own.
- **`api/`** — the only thing the frontend talks to. Handles signup/login
  (JWT in an HttpOnly cookie), per-user stack selection, the personalized
  digest endpoint, a cached Hacker News feed, account deletion, and email
  preferences. Also serves the server-rendered frontend (landing page,
  signup/login, and the dashboard) via Go's `html/template` + `go:embed`.

`api/` and `worker/` each own a full copy of their infra clients (MongoDB,
email) and domain types, by design — see the comment in `worker/Dockerfile`
and each service's `internal/` tree. They are deployed and scaled
separately, so nothing is shared beyond the top-level Go module and the
Mongo database itself.

```
api/
  cmd/            entry point (HTTP server)
  domain/         User, NewsItem, GeneralFeedItem schemas
  internal/
    httpapi/      routes, handlers, embedded frontend (web/)
    mongodb/      Mongo-backed stores/readers
    email/        Resend client + branded HTML email templates
    hackernews/   Hacker News API client
worker/
  cmd/            entry point (ingestion loop) + tracked-repo list
  domain/         its own copy of User, NewsItem
  internal/
    github/       GitHub Releases + tag-fallback client
    classifier/   keyword-based severity classifier
    mongodb/      Mongo-backed store/reader
    email/        its own Resend client + email templates
```

## Running locally

Requires Go 1.26+, a MongoDB Atlas connection string, and a
[Resend](https://resend.com) API key.

```
cp api/.env.example api/.env       # fill in MONGODB_URI, RESEND_API_KEY, RESEND_FROM, JWT_SECRET, SUPPORT_EMAIL
cp worker/.env.example worker/.env # fill in MONGODB_URI, RESEND_API_KEY, RESEND_FROM, APP_BASE_URL

go run ./api/cmd     # listens on :8080 (PORT env var to override)
go run ./worker/cmd  # ingests every 5 minutes by default (INGEST_INTERVAL to override)
```

A `GITHUB_TOKEN` env var is optional locally but effectively required once
the worker is tracking its full repo list — GitHub's unauthenticated rate
limit (60/hr) is too low for a real ingestion run; a token raises it to
5,000/hr.

## Testing

```
go build ./...
go vet ./...
go test ./...
```

Most tests hit real infrastructure (MongoDB Atlas, the live GitHub API)
rather than mocks — set `MONGODB_URI` before running `go test ./...`.

## Deployment

The worker deploys to Railway as its own service from `worker/Dockerfile`
(build context is the repo root). The API is not yet deployed publicly —
it currently only runs locally.

package main

import (
	"context"
	"fmt"
	htmlpkg "html"
	"log"
	"os"
	"strings"
	"time"

	"github.com/Platon223/DevSwell/worker/domain"
	"github.com/Platon223/DevSwell/worker/internal/classifier"
	"github.com/Platon223/DevSwell/worker/internal/email"
	"github.com/Platon223/DevSwell/worker/internal/github"
	"github.com/Platon223/DevSwell/worker/internal/mongodb"
	"github.com/joho/godotenv"
)

const defaultIngestInterval = 5 * time.Minute

// interRepoDelay paces requests between repos in a run, since GitHub applies
// secondary "abuse detection" rate limiting to bursts of rapid requests
// independent of the hourly quota — observed firsthand while tracking ~50
// repos (several got 403s with a fresh, mostly-unused token).
const interRepoDelay = 200 * time.Millisecond

func main() {
	log.Println("DevSwell worker starting...")

	_ = godotenv.Load("worker/.env")

	interval := defaultIngestInterval
	if raw := os.Getenv("INGEST_INTERVAL"); raw != "" {
		parsed, err := time.ParseDuration(raw)
		if err != nil {
			log.Fatalf("parsing INGEST_INTERVAL %q: %v", raw, err)
		}
		interval = parsed
	}

	ctx := context.Background()

	client, err := mongodb.Connect(ctx)
	if err != nil {
		log.Fatalf("connecting to mongodb: %v", err)
	}
	defer client.Disconnect(ctx)

	store := mongodb.NewNewsItemStore(client)
	if err := store.EnsureIndexes(ctx); err != nil {
		log.Fatalf("ensuring indexes: %v", err)
	}

	users := mongodb.NewUserReader(client)
	mailer := email.NewClient(os.Getenv("RESEND_API_KEY"), os.Getenv("RESEND_FROM"))

	// Used to build the logo URL in emails — must be the API's real public
	// domain (it serves /static/logo-mark.png), since email clients load
	// images from a real hosted URL, not a local path.
	appBaseURL := os.Getenv("APP_BASE_URL")

	log.Printf("running ingestion every %s", interval)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		runIngestion(ctx, store, users, mailer, appBaseURL)
		<-ticker.C
	}
}

func runIngestion(ctx context.Context, store *mongodb.NewsItemStore, users *mongodb.UserReader, mailer *email.Client, appBaseURL string) {
	log.Println("ingestion run starting")

	var changedItems []domain.NewsItem

	for i, repo := range trackedRepos {
		if i > 0 {
			time.Sleep(interRepoDelay)
		}

		project := fmt.Sprintf("%s/%s", repo.Owner, repo.Name)

		release, err := github.FetchLatestRelease(ctx, repo.Owner, repo.Name)
		if err != nil {
			log.Printf("fetching latest release for %s: %v", project, err)
			continue
		}
		if release == nil {
			// No formal GitHub Release exists — fall back to tags (many
			// major projects, including Go itself, never use Releases).
			release, err = github.FetchLatestTag(ctx, repo.Owner, repo.Name, repo.TagPattern)
			if err != nil {
				log.Printf("fetching latest tag for %s: %v", project, err)
				continue
			}
			if release == nil {
				fmt.Printf("%s: no releases or version-like tags found\n", project)
				continue
			}
		}

		item := newsItemFromRelease(project, repo.Technology, *release)
		changed, err := store.Upsert(ctx, item)
		if err != nil {
			log.Printf("saving %s: %v", item.URL, err)
			continue
		}
		fmt.Printf("%s: saved %s [%s] (%s)\n", item.Project, item.Title, item.Severity, item.URL)
		if changed {
			changedItems = append(changedItems, item)
		}
	}

	if len(changedItems) > 0 {
		notifyUsersOfChanges(ctx, users, mailer, changedItems, appBaseURL)
	}

	log.Println("ingestion run finished")
}

// notifyUsersOfChanges emails each affected user a single batched summary of
// every changed item relevant to their stack from this run, rather than one
// email per changed item.
func notifyUsersOfChanges(ctx context.Context, users *mongodb.UserReader, mailer *email.Client, changedItems []domain.NewsItem, appBaseURL string) {
	byTechnology := map[string][]domain.NewsItem{}
	for _, item := range changedItems {
		byTechnology[item.Technology] = append(byTechnology[item.Technology], item)
	}

	userItems := map[string][]domain.NewsItem{}
	for technology, items := range byTechnology {
		matched, err := users.FindByTechnology(ctx, technology)
		if err != nil {
			log.Printf("finding users tracking %s: %v", technology, err)
			continue
		}
		for _, u := range matched {
			userItems[u.Email] = append(userItems[u.Email], items...)
		}
	}

	for recipient, items := range userItems {
		if err := mailer.Send(ctx, recipient, "Your DevSwell stack update", stackUpdateEmailHTML(appBaseURL, items)); err != nil {
			log.Printf("sending stack update email to %s: %v", recipient, err)
			continue
		}
		fmt.Printf("notified %s of %d change(s)\n", recipient, len(items))
	}
}

// severityColors mirrors the dashboard's badge colors (worker/dashboard both
// use the light-theme values here, since email defaults to a light
// background regardless of the recipient's web theme preference).
func severityColors(severity string) (bg, fg string) {
	switch severity {
	case string(classifier.SeveritySecurity):
		return "rgba(207,34,46,0.08)", "#cf222e"
	case string(classifier.SeverityBreaking):
		return "rgba(154,103,0,0.1)", "#9a6700"
	default:
		return "rgba(26,127,55,0.08)", "#1a7f37"
	}
}

func stackUpdateEmailHTML(appBaseURL string, items []domain.NewsItem) string {
	var rows strings.Builder
	for _, item := range items {
		bg, fg := severityColors(item.Severity)
		rows.WriteString(fmt.Sprintf(`
    <tr>
      <td style="padding:14px 0;border-bottom:1px solid #e5e5e5;">
        <span style="display:inline-block;font-size:11px;font-weight:700;text-transform:uppercase;letter-spacing:0.03em;padding:2px 8px;border-radius:4px;background-color:%s;color:%s;margin-bottom:6px;">%s</span><br/>
        <a href="%s" style="font-size:14px;font-weight:600;color:#171717;text-decoration:none;">%s</a>
        <div style="font-size:12px;color:#6b6b6b;margin-top:2px;">%s &middot; %s</div>
      </td>
    </tr>`,
			bg, fg, htmlpkg.EscapeString(strings.ToUpper(item.Severity)),
			htmlpkg.EscapeString(item.URL), htmlpkg.EscapeString(item.Title),
			htmlpkg.EscapeString(item.Project), htmlpkg.EscapeString(item.Technology)))
	}

	body := fmt.Sprintf(`
    <p style="font-size:14px;color:#3a3a3a;line-height:1.6;margin:0 0 16px;">Here's what changed in the technologies you track:</p>
    <table role="presentation" cellpadding="0" cellspacing="0" style="width:100%%;">%s</table>`, rows.String())

	return email.BrandedHTML(appBaseURL+"/static/logo-mark.png", "Your DevSwell stack update", body)
}

// newsItemFromRelease maps a fetched GitHub release to a storable NewsItem,
// classifying its severity automatically so no manual step is needed.
func newsItemFromRelease(project, technology string, release github.Release) domain.NewsItem {
	title := release.Name
	if title == "" {
		// Some maintainers publish a release without setting its display
		// name — fall back to the tag rather than storing a blank title.
		title = release.TagName
	}
	return domain.NewsItem{
		Title:       title,
		Source:      "github",
		Project:     project,
		URL:         release.URL,
		PublishedAt: release.PublishedAt,
		Type:        "release",
		Body:        release.Body,
		Severity:    string(classifier.Classify(title, release.Body)),
		Technology:  technology,
	}
}

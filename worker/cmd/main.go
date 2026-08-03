package main

import (
	"context"
	"fmt"
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

var trackedRepos = []struct {
	Owner      string
	Name       string
	Technology string
}{
	{"cli", "cli", "Go"},
	{"vitejs", "vite", "JavaScript"},
	{"denoland", "deno", "Rust"},
}

const defaultIngestInterval = 5 * time.Minute

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

	log.Printf("running ingestion every %s", interval)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		runIngestion(ctx, store, users, mailer)
		<-ticker.C
	}
}

func runIngestion(ctx context.Context, store *mongodb.NewsItemStore, users *mongodb.UserReader, mailer *email.Client) {
	log.Println("ingestion run starting")

	var changedItems []domain.NewsItem

	for _, repo := range trackedRepos {
		project := fmt.Sprintf("%s/%s", repo.Owner, repo.Name)

		release, err := github.FetchLatestRelease(ctx, repo.Owner, repo.Name)
		if err != nil {
			log.Printf("fetching latest release for %s: %v", project, err)
			continue
		}
		if release == nil {
			fmt.Printf("%s: no releases found\n", project)
			continue
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
		notifyUsersOfChanges(ctx, users, mailer, changedItems)
	}

	log.Println("ingestion run finished")
}

// notifyUsersOfChanges emails each affected user a single batched summary of
// every changed item relevant to their stack from this run, rather than one
// email per changed item.
func notifyUsersOfChanges(ctx context.Context, users *mongodb.UserReader, mailer *email.Client, changedItems []domain.NewsItem) {
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
		if err := mailer.Send(ctx, recipient, "Your DevSwell stack update", stackUpdateEmailHTML(items)); err != nil {
			log.Printf("sending stack update email to %s: %v", recipient, err)
			continue
		}
		fmt.Printf("notified %s of %d change(s)\n", recipient, len(items))
	}
}

func stackUpdateEmailHTML(items []domain.NewsItem) string {
	var rows strings.Builder
	for _, item := range items {
		rows.WriteString(fmt.Sprintf(`
  <li style="margin-bottom:12px;">
    <strong>%s</strong> (%s) &mdash; %s<br/>
    <a href="%s">%s</a>
  </li>`, item.Project, item.Severity, item.Title, item.URL, item.URL))
	}

	return fmt.Sprintf(`
<div style="font-family: sans-serif; padding: 24px;">
  <h2>Your DevSwell stack update</h2>
  <p>Here's what changed in the technologies you track:</p>
  <ul style="list-style:none;padding:0;">%s</ul>
</div>`, rows.String())
}

// newsItemFromRelease maps a fetched GitHub release to a storable NewsItem,
// classifying its severity automatically so no manual step is needed.
func newsItemFromRelease(project, technology string, release github.Release) domain.NewsItem {
	return domain.NewsItem{
		Title:       release.Name,
		Source:      "github",
		Project:     project,
		URL:         release.URL,
		PublishedAt: release.PublishedAt,
		Type:        "release",
		Body:        release.Body,
		Severity:    string(classifier.Classify(release.Name, release.Body)),
		Technology:  technology,
	}
}

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/Platon223/DevSwell/worker/domain"
	"github.com/Platon223/DevSwell/worker/internal/classifier"
	"github.com/Platon223/DevSwell/worker/internal/github"
	"github.com/Platon223/DevSwell/worker/internal/mongodb"
	"github.com/joho/godotenv"
)

var trackedRepos = []struct {
	Owner string
	Name  string
}{
	{"cli", "cli"},
	{"vitejs", "vite"},
	{"denoland", "deno"},
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

	log.Printf("running ingestion every %s", interval)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		runIngestion(ctx, store)
		<-ticker.C
	}
}

func runIngestion(ctx context.Context, store *mongodb.NewsItemStore) {
	log.Println("ingestion run starting")

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

		item := newsItemFromRelease(project, *release)
		if err := store.Upsert(ctx, item); err != nil {
			log.Printf("saving %s: %v", item.URL, err)
			continue
		}
		fmt.Printf("%s: saved %s [%s] (%s)\n", item.Project, item.Title, item.Severity, item.URL)
	}

	log.Println("ingestion run finished")
}

// newsItemFromRelease maps a fetched GitHub release to a storable NewsItem,
// classifying its severity automatically so no manual step is needed.
func newsItemFromRelease(project string, release github.Release) domain.NewsItem {
	return domain.NewsItem{
		Title:       release.Name,
		Source:      "github",
		Project:     project,
		URL:         release.URL,
		PublishedAt: release.PublishedAt,
		Type:        "release",
		Body:        release.Body,
		Severity:    string(classifier.Classify(release.Name, release.Body)),
	}
}

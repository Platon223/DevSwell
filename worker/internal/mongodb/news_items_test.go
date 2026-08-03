package mongodb

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/Platon223/DevSwell/worker/domain"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

func TestNewsItemStoreRejectsDuplicateURL(t *testing.T) {
	_ = godotenv.Load("../../.env")

	ctx := context.Background()
	client, err := Connect(ctx)
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer client.Disconnect(ctx)

	store := NewNewsItemStore(client)
	if err := store.EnsureIndexes(ctx); err != nil {
		t.Fatalf("EnsureIndexes: %v", err)
	}

	suffix := time.Now().UnixNano()
	url := fmt.Sprintf("https://example.com/day3-dedupe-test/%d", suffix)
	project := fmt.Sprintf("example/day3-dedupe-test-%d", suffix)
	defer store.collection.DeleteOne(ctx, bson.M{"project": project})

	first := domain.NewsItem{
		Title:       "Fake release",
		Source:      "github",
		Project:     project,
		URL:         url,
		PublishedAt: time.Now(),
		Type:        "release",
	}
	if err := store.Insert(ctx, first); err != nil {
		t.Fatalf("first insert should have succeeded: %v", err)
	}

	second := first
	second.Title = "Fake release, duplicate URL"
	err = store.Insert(ctx, second)
	if err == nil {
		t.Fatal("second insert with the same URL should have been rejected, but succeeded")
	}
	if !mongo.IsDuplicateKeyError(err) {
		t.Fatalf("expected a duplicate key error, got: %v", err)
	}
}

func TestNewsItemStoreUpsertDoesNotDuplicate(t *testing.T) {
	_ = godotenv.Load("../../.env")

	ctx := context.Background()
	client, err := Connect(ctx)
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer client.Disconnect(ctx)

	store := NewNewsItemStore(client)
	if err := store.EnsureIndexes(ctx); err != nil {
		t.Fatalf("EnsureIndexes: %v", err)
	}

	suffix := time.Now().UnixNano()
	url := fmt.Sprintf("https://example.com/day6-upsert-test/%d", suffix)
	project := fmt.Sprintf("example/day6-upsert-test-%d", suffix)
	defer store.collection.DeleteOne(ctx, bson.M{"project": project})

	item := domain.NewsItem{
		Title:       "Original title",
		Source:      "github",
		Project:     project,
		URL:         url,
		PublishedAt: time.Now(),
		Type:        "release",
		Body:        "original body",
	}
	changed, err := store.Upsert(ctx, item)
	if err != nil {
		t.Fatalf("first upsert: %v", err)
	}
	if !changed {
		t.Fatal("first upsert (brand new project) should report changed=true")
	}

	sameAgain, err := store.Upsert(ctx, item)
	if err != nil {
		t.Fatalf("re-upserting identical data: %v", err)
	}
	if sameAgain {
		t.Fatal("re-upserting identical data should report changed=false")
	}

	item.Title = "Updated title"
	changed, err = store.Upsert(ctx, item)
	if err != nil {
		t.Fatalf("second upsert: %v", err)
	}
	if !changed {
		t.Fatal("upsert with a different title should report changed=true")
	}

	count, err := store.collection.CountDocuments(ctx, bson.M{"project": project})
	if err != nil {
		t.Fatalf("CountDocuments: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected exactly 1 document for project %s after two upserts, got %d", project, count)
	}

	var got domain.NewsItem
	if err := store.collection.FindOne(ctx, bson.M{"project": project}).Decode(&got); err != nil {
		t.Fatalf("FindOne: %v", err)
	}
	if got.Title != "Updated title" {
		t.Fatalf("expected the second upsert to update the title in place, got %q", got.Title)
	}
}

func TestNewsItemStoreUpsertReplacesOldReleaseForSameProject(t *testing.T) {
	_ = godotenv.Load("../../.env")

	ctx := context.Background()
	client, err := Connect(ctx)
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer client.Disconnect(ctx)

	store := NewNewsItemStore(client)
	if err := store.EnsureIndexes(ctx); err != nil {
		t.Fatalf("EnsureIndexes: %v", err)
	}

	suffix := time.Now().UnixNano()
	project := fmt.Sprintf("example/day6-replace-test-%d", suffix)
	oldURL := fmt.Sprintf("https://example.com/day6-replace-test/old/%d", suffix)
	newURL := fmt.Sprintf("https://example.com/day6-replace-test/new/%d", suffix)
	defer store.collection.DeleteOne(ctx, bson.M{"project": project})

	old := domain.NewsItem{
		Title:       "v1.0.0",
		Source:      "github",
		Project:     project,
		URL:         oldURL,
		PublishedAt: time.Now(),
		Type:        "release",
		Body:        "first release notes",
	}
	if _, err := store.Upsert(ctx, old); err != nil {
		t.Fatalf("upserting old release: %v", err)
	}

	newer := old
	newer.Title = "v1.1.0"
	newer.URL = newURL
	newer.Body = "second release notes"
	if _, err := store.Upsert(ctx, newer); err != nil {
		t.Fatalf("upserting newer release: %v", err)
	}

	count, err := store.collection.CountDocuments(ctx, bson.M{"project": project})
	if err != nil {
		t.Fatalf("CountDocuments: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected the new release to replace the old one, got %d documents for project %s", count, project)
	}

	var got domain.NewsItem
	if err := store.collection.FindOne(ctx, bson.M{"project": project}).Decode(&got); err != nil {
		t.Fatalf("FindOne: %v", err)
	}
	if got.URL != newURL || got.Body != "second release notes" {
		t.Fatalf("expected the stored document to reflect the newer release, got url=%q body=%q", got.URL, got.Body)
	}

	oldStillExists, err := store.collection.CountDocuments(ctx, bson.M{"url": oldURL})
	if err != nil {
		t.Fatalf("CountDocuments for old url: %v", err)
	}
	if oldStillExists != 0 {
		t.Fatalf("expected the old release's url to no longer exist, but it does")
	}
}

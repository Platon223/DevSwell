package mongodb

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/Platon223/DevSwell/api/domain"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestFilterByStackReturnsOnlyMatchingTechnology(t *testing.T) {
	_ = godotenv.Load("../../.env")

	ctx := context.Background()
	client, err := Connect(ctx)
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer client.Disconnect(ctx)

	reader := NewNewsItemReader(client)

	suffix := time.Now().UnixNano()
	goProject := fmt.Sprintf("example/day16-go-test-%d", suffix)
	jsProject := fmt.Sprintf("example/day16-js-test-%d", suffix)
	rustProject := fmt.Sprintf("example/day16-rust-test-%d", suffix)
	defer reader.collection.DeleteMany(ctx, bson.M{"project": bson.M{"$in": []string{goProject, jsProject, rustProject}}})

	fakeItems := []any{
		domain.NewsItem{Title: "Go release", Source: "github", Project: goProject, URL: goProject, PublishedAt: time.Now(), Type: "release", Technology: "Go"},
		domain.NewsItem{Title: "JS release", Source: "github", Project: jsProject, URL: jsProject, PublishedAt: time.Now(), Type: "release", Technology: "JavaScript"},
		domain.NewsItem{Title: "Rust release", Source: "github", Project: rustProject, URL: rustProject, PublishedAt: time.Now(), Type: "release", Technology: "Rust"},
	}
	if _, err := reader.collection.InsertMany(ctx, fakeItems); err != nil {
		t.Fatalf("seeding fake items: %v", err)
	}

	items, err := reader.FilterByStack(ctx, []string{"Go"})
	if err != nil {
		t.Fatalf("FilterByStack: %v", err)
	}

	// Only assert on our own seeded test items, since other real items may
	// exist in the collection from the worker's actual ingestion.
	var matched []domain.NewsItem
	for _, item := range items {
		if item.Project == goProject || item.Project == jsProject || item.Project == rustProject {
			matched = append(matched, item)
		}
	}

	if len(matched) != 1 {
		t.Fatalf("expected exactly 1 matching item for stack [Go], got %d: %+v", len(matched), matched)
	}
	if matched[0].Project != goProject || matched[0].Technology != "Go" {
		t.Fatalf("expected the Go item to match, got %+v", matched[0])
	}
}

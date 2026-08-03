package mongodb

import (
	"context"
	"testing"
	"time"

	"github.com/Platon223/DevSwell/api/domain"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestGeneralFeedCacheFreshness(t *testing.T) {
	_ = godotenv.Load("../../.env")

	ctx := context.Background()
	client, err := Connect(ctx)
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer client.Disconnect(ctx)

	store := NewGeneralFeedStore(client)
	// This collection is a singleton (one cache record total), so clean up
	// the test data afterward rather than leaving fake data as "the" cache.
	defer store.collection.DeleteOne(ctx, bson.M{"key": generalFeedCacheKey})

	items := []domain.GeneralFeedItem{
		{Title: "Fake HN story", URL: "https://example.com/day18-freshness-test"},
	}
	if err := store.Save(ctx, items); err != nil {
		t.Fatalf("Save: %v", err)
	}

	fresh, err := store.Get(ctx)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !fresh.IsFresh(time.Hour) {
		t.Fatal("a cache entry saved just now should be fresh under a 1-hour max age")
	}

	// Simulate a stale cache by directly writing an old fetched_at.
	stale := fresh
	stale.FetchedAt = time.Now().Add(-2 * time.Hour)
	if stale.IsFresh(time.Hour) {
		t.Fatal("a cache entry from 2 hours ago should be stale under a 1-hour max age")
	}
}

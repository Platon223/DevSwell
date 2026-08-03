package hackernews

import (
	"context"
	"fmt"
	"testing"
)

func TestFetchTopStories(t *testing.T) {
	stories, err := FetchTopStories(context.Background(), 5)
	if err != nil {
		t.Fatalf("FetchTopStories: %v", err)
	}
	if len(stories) == 0 {
		t.Fatal("expected at least one story, got none")
	}

	for _, s := range stories {
		if s.Title == "" {
			t.Fatalf("story %d has an empty title, not real data", s.ID)
		}
		fmt.Printf("[%d] %s (score %d) - %s\n", s.ID, s.Title, s.Score, s.URL)
	}
}

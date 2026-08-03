package httpapi

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/Platon223/DevSwell/api/domain"
	"github.com/Platon223/DevSwell/api/internal/hackernews"
)

const generalFeedMaxAge = 15 * time.Minute
const generalFeedStoryCount = 10

// topStoriesFetcher matches hackernews.FetchTopStories's signature, injected
// so tests can simulate a failing live fetch without depending on the real API.
type topStoriesFetcher func(ctx context.Context, n int) ([]hackernews.Story, error)

// generalFeedCache is satisfied by *mongodb.GeneralFeedStore; declared here so
// tests can inject a fake with fully controllable freshness state.
type generalFeedCache interface {
	Get(ctx context.Context) (domain.GeneralFeedCache, error)
	Save(ctx context.Context, items []domain.GeneralFeedItem) error
}

type generalFeedItemResponse struct {
	Title string `json:"title"`
	URL   string `json:"url"`
}

type generalFeedResponse struct {
	Source string                    `json:"source"`
	Items  []generalFeedItemResponse `json:"items"`
}

// feedHandler serves the general (Hacker News) feed: cache-hit if fresh,
// live-fetch (and cache write-back) if stale/missing, and falls back to
// stale cached data rather than erroring if the live fetch fails.
func feedHandler(cache generalFeedCache, fetch topStoriesFetcher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cached, err := cache.Get(r.Context())
		if err != nil {
			http.Error(w, "could not load feed", http.StatusInternalServerError)
			return
		}

		if cached.IsFresh(generalFeedMaxAge) {
			writeFeedResponse(w, cached.Items, "cache")
			return
		}

		stories, err := fetch(r.Context(), generalFeedStoryCount)
		if err != nil {
			log.Printf("live fetch of hacker news failed: %v", err)
			if len(cached.Items) > 0 {
				writeFeedResponse(w, cached.Items, "stale-fallback")
				return
			}
			http.Error(w, "could not load feed", http.StatusServiceUnavailable)
			return
		}

		items := make([]domain.GeneralFeedItem, 0, len(stories))
		for _, s := range stories {
			items = append(items, domain.GeneralFeedItem{Title: s.Title, URL: s.URL})
		}
		if err := cache.Save(r.Context(), items); err != nil {
			log.Printf("saving general feed cache: %v", err)
		}

		writeFeedResponse(w, items, "live")
	}
}

func writeFeedResponse(w http.ResponseWriter, items []domain.GeneralFeedItem, source string) {
	resp := generalFeedResponse{Source: source, Items: make([]generalFeedItemResponse, 0, len(items))}
	for _, item := range items {
		resp.Items = append(resp.Items, generalFeedItemResponse{Title: item.Title, URL: item.URL})
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

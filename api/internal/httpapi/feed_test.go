package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Platon223/DevSwell/api/domain"
	"github.com/Platon223/DevSwell/api/internal/hackernews"
)

// fakeFeedCache is an in-memory stand-in for *mongodb.GeneralFeedStore,
// giving tests full control over the cache's freshness state.
type fakeFeedCache struct {
	cache domain.GeneralFeedCache
	saved []domain.GeneralFeedItem
}

func (f *fakeFeedCache) Get(ctx context.Context) (domain.GeneralFeedCache, error) {
	return f.cache, nil
}

func (f *fakeFeedCache) Save(ctx context.Context, items []domain.GeneralFeedItem) error {
	f.saved = items
	f.cache = domain.GeneralFeedCache{FetchedAt: time.Now(), Items: items}
	return nil
}

func doFeedRequest(t *testing.T, cache generalFeedCache, fetch topStoriesFetcher) (*httptest.ResponseRecorder, generalFeedResponse) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/feed", nil)
	w := httptest.NewRecorder()
	feedHandler(cache, fetch)(w, req)

	var resp generalFeedResponse
	if w.Code == http.StatusOK {
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("decoding response: %v", err)
		}
	}
	return w, resp
}

func TestFeedHandlerServesFromCacheWhenFresh(t *testing.T) {
	cache := &fakeFeedCache{
		cache: domain.GeneralFeedCache{
			FetchedAt: time.Now(),
			Items:     []domain.GeneralFeedItem{{Title: "Cached story", URL: "https://example.com/cached"}},
		},
	}
	fetchCalled := false
	fetch := func(ctx context.Context, n int) ([]hackernews.Story, error) {
		fetchCalled = true
		return nil, nil
	}

	w, resp := doFeedRequest(t, cache, fetch)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if resp.Source != "cache" {
		t.Fatalf("expected source=cache, got %q", resp.Source)
	}
	if fetchCalled {
		t.Fatal("expected the live fetch to be skipped when cache is fresh, but it was called")
	}
	if len(resp.Items) != 1 || resp.Items[0].Title != "Cached story" {
		t.Fatalf("expected the cached item back, got %+v", resp.Items)
	}
}

func TestFeedHandlerFetchesLiveAndWritesBackWhenStale(t *testing.T) {
	cache := &fakeFeedCache{
		cache: domain.GeneralFeedCache{
			FetchedAt: time.Now().Add(-1 * time.Hour), // stale
			Items:     []domain.GeneralFeedItem{{Title: "Old story", URL: "https://example.com/old"}},
		},
	}
	fetchCalled := false
	fetch := func(ctx context.Context, n int) ([]hackernews.Story, error) {
		fetchCalled = true
		return []hackernews.Story{{ID: 1, Title: "Fresh story", URL: "https://example.com/fresh"}}, nil
	}

	w, resp := doFeedRequest(t, cache, fetch)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if resp.Source != "live" {
		t.Fatalf("expected source=live, got %q", resp.Source)
	}
	if !fetchCalled {
		t.Fatal("expected the live fetch to be called when cache is stale")
	}
	if len(resp.Items) != 1 || resp.Items[0].Title != "Fresh story" {
		t.Fatalf("expected the freshly fetched item back, got %+v", resp.Items)
	}
	if len(cache.saved) != 1 || cache.saved[0].Title != "Fresh story" {
		t.Fatalf("expected the fresh result to be written back to the cache, got %+v", cache.saved)
	}
}

func TestFeedHandlerSecondRequestIsServedFromCacheWithoutRefetching(t *testing.T) {
	cache := &fakeFeedCache{
		cache: domain.GeneralFeedCache{
			FetchedAt: time.Now().Add(-1 * time.Hour), // stale on the first request
			Items:     []domain.GeneralFeedItem{{Title: "Old story", URL: "https://example.com/old"}},
		},
	}
	fetchCount := 0
	fetch := func(ctx context.Context, n int) ([]hackernews.Story, error) {
		fetchCount++
		return []hackernews.Story{{ID: 1, Title: "Fresh story", URL: "https://example.com/fresh"}}, nil
	}

	_, first := doFeedRequest(t, cache, fetch)
	if first.Source != "live" || fetchCount != 1 {
		t.Fatalf("expected first request to fetch live once, got source=%q fetchCount=%d", first.Source, fetchCount)
	}

	_, second := doFeedRequest(t, cache, fetch)
	if second.Source != "cache" {
		t.Fatalf("expected second request to be served from cache, got source=%q", second.Source)
	}
	if fetchCount != 1 {
		t.Fatalf("expected the live fetch NOT to be called again on the second request, but it was called %d times", fetchCount)
	}
}

func TestFeedHandlerFallsBackToStaleCacheWhenLiveFetchFails(t *testing.T) {
	cache := &fakeFeedCache{
		cache: domain.GeneralFeedCache{
			FetchedAt: time.Now().Add(-1 * time.Hour), // stale
			Items:     []domain.GeneralFeedItem{{Title: "Old story", URL: "https://example.com/old"}},
		},
	}
	fetch := func(ctx context.Context, n int) ([]hackernews.Story, error) {
		return nil, errors.New("simulated network failure")
	}

	w, resp := doFeedRequest(t, cache, fetch)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 (served stale data despite the live fetch failing), got %d", w.Code)
	}
	if resp.Source != "stale-fallback" {
		t.Fatalf("expected source=stale-fallback, got %q", resp.Source)
	}
	if len(resp.Items) != 1 || resp.Items[0].Title != "Old story" {
		t.Fatalf("expected the stale cached item back as a fallback, got %+v", resp.Items)
	}
}

func TestFeedHandlerReturnsErrorWhenNoCacheAndLiveFetchFails(t *testing.T) {
	cache := &fakeFeedCache{} // no cache at all (zero value)
	fetch := func(ctx context.Context, n int) ([]hackernews.Story, error) {
		return nil, errors.New("simulated network failure")
	}

	req := httptest.NewRequest(http.MethodGet, "/feed", nil)
	w := httptest.NewRecorder()
	feedHandler(cache, fetch)(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 when there's no cache and the live fetch fails, got %d", w.Code)
	}
}

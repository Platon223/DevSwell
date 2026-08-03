package hackernews

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

const baseURL = "https://hacker-news.firebaseio.com/v0"

type Story struct {
	ID    int    `json:"id"`
	Title string `json:"title"`
	URL   string `json:"url"`
	Score int    `json:"score"`
}

// FetchTopStories pulls the current top stories from Hacker News, capped at n.
func FetchTopStories(ctx context.Context, n int) ([]Story, error) {
	ids, err := fetchTopStoryIDs(ctx)
	if err != nil {
		return nil, err
	}
	if len(ids) > n {
		ids = ids[:n]
	}

	stories := make([]Story, 0, len(ids))
	for _, id := range ids {
		story, err := fetchItem(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("fetching item %d: %w", id, err)
		}
		stories = append(stories, story)
	}
	return stories, nil
}

func fetchTopStoryIDs(ctx context.Context) ([]int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/topstories.json", nil)
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("requesting top stories: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("hacker news api returned %s", resp.Status)
	}

	var ids []int
	if err := json.NewDecoder(resp.Body).Decode(&ids); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}
	return ids, nil
}

func fetchItem(ctx context.Context, id int) (Story, error) {
	url := fmt.Sprintf("%s/item/%d.json", baseURL, id)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return Story{}, fmt.Errorf("building request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return Story{}, fmt.Errorf("requesting item %d: %w", id, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return Story{}, fmt.Errorf("hacker news api returned %s for item %d", resp.Status, id)
	}

	var story Story
	if err := json.NewDecoder(resp.Body).Decode(&story); err != nil {
		return Story{}, fmt.Errorf("decoding item %d: %w", id, err)
	}
	return story, nil
}

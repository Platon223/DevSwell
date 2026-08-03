package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type GeneralFeedItem struct {
	Title string `bson:"title"`
	URL   string `bson:"url"`
}

// GeneralFeedCache is a single cached snapshot of the general (Hacker News)
// feed, refreshed as a whole rather than per-item.
type GeneralFeedCache struct {
	ID        bson.ObjectID     `bson:"_id,omitempty"`
	FetchedAt time.Time         `bson:"fetched_at"`
	Items     []GeneralFeedItem `bson:"items"`
}

// IsFresh reports whether this cache entry was fetched within maxAge.
func (c GeneralFeedCache) IsFresh(maxAge time.Duration) bool {
	return time.Since(c.FetchedAt) < maxAge
}

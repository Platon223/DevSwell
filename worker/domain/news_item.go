package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type NewsItem struct {
	ID          bson.ObjectID `bson:"_id,omitempty"`
	Title       string        `bson:"title"`
	Source      string        `bson:"source"`
	Project     string        `bson:"project"`
	URL         string        `bson:"url"`
	PublishedAt time.Time     `bson:"published_at"`
	Type        string        `bson:"type"`
	Body        string        `bson:"body"`
	Severity    string        `bson:"severity"`
}

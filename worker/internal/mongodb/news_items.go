package mongodb

import (
	"context"

	"github.com/Platon223/DevSwell/worker/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const newsItemsCollection = "news_items"

type NewsItemStore struct {
	collection *mongo.Collection
}

func NewNewsItemStore(client *mongo.Client) *NewsItemStore {
	return &NewsItemStore{
		collection: client.Database("devswell").Collection(newsItemsCollection),
	}
}

// EnsureIndexes creates the indexes news_items relies on: url must be globally
// unique (belt-and-suspenders against accidental exact-duplicate inserts), and
// project must be unique so each tracked project has at most one current item.
func (s *NewsItemStore) EnsureIndexes(ctx context.Context) error {
	_, err := s.collection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "url", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys:    bson.D{{Key: "project", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
	})
	return err
}

func (s *NewsItemStore) Insert(ctx context.Context, item domain.NewsItem) error {
	_, err := s.collection.InsertOne(ctx, item)
	return err
}

// Upsert saves item keyed by its project: a first call inserts it, and a later
// call for the same project (e.g. a new release) replaces the existing
// document in place instead of accumulating history.
func (s *NewsItemStore) Upsert(ctx context.Context, item domain.NewsItem) error {
	filter := bson.M{"project": item.Project}
	update := bson.M{"$set": bson.M{
		"title":        item.Title,
		"source":       item.Source,
		"project":      item.Project,
		"url":          item.URL,
		"published_at": item.PublishedAt,
		"type":         item.Type,
		"body":         item.Body,
		"severity":     item.Severity,
	}}
	_, err := s.collection.UpdateOne(ctx, filter, update, options.UpdateOne().SetUpsert(true))
	return err
}

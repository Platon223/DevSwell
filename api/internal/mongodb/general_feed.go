package mongodb

import (
	"context"
	"time"

	"github.com/Platon223/DevSwell/api/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const generalFeedCollection = "general_feed_cache"

// generalFeedCacheKey is a fixed filter value: this collection only ever
// holds one document, the current cached snapshot.
const generalFeedCacheKey = "singleton"

type GeneralFeedStore struct {
	collection *mongo.Collection
}

func NewGeneralFeedStore(client *mongo.Client) *GeneralFeedStore {
	return &GeneralFeedStore{
		collection: client.Database("devswell").Collection(generalFeedCollection),
	}
}

// Save replaces the cached snapshot with items, stamped with the current time.
func (s *GeneralFeedStore) Save(ctx context.Context, items []domain.GeneralFeedItem) error {
	filter := bson.M{"key": generalFeedCacheKey}
	update := bson.M{"$set": bson.M{
		"key":        generalFeedCacheKey,
		"fetched_at": time.Now(),
		"items":      items,
	}}
	_, err := s.collection.UpdateOne(ctx, filter, update, options.UpdateOne().SetUpsert(true))
	return err
}

// Get returns the cached snapshot. If no snapshot has ever been saved, it
// returns a zero-value GeneralFeedCache (whose IsFresh is always false)
// rather than an error, since "no cache yet" is an expected, ordinary state.
func (s *GeneralFeedStore) Get(ctx context.Context) (domain.GeneralFeedCache, error) {
	var cache domain.GeneralFeedCache
	err := s.collection.FindOne(ctx, bson.M{"key": generalFeedCacheKey}).Decode(&cache)
	if err == mongo.ErrNoDocuments {
		return domain.GeneralFeedCache{}, nil
	}
	return cache, err
}

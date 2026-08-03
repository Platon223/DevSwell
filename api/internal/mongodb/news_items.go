package mongodb

import (
	"context"

	"github.com/Platon223/DevSwell/api/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

const newsItemsCollection = "news_items"

type NewsItemReader struct {
	collection *mongo.Collection
}

func NewNewsItemReader(client *mongo.Client) *NewsItemReader {
	return &NewsItemReader{
		collection: client.Database("devswell").Collection(newsItemsCollection),
	}
}

// FilterByStack returns news items whose technology matches any entry in stack.
func (r *NewsItemReader) FilterByStack(ctx context.Context, stack []string) ([]domain.NewsItem, error) {
	if len(stack) == 0 {
		return []domain.NewsItem{}, nil
	}

	cursor, err := r.collection.Find(ctx, bson.M{"technology": bson.M{"$in": stack}})
	if err != nil {
		return nil, err
	}

	items := []domain.NewsItem{}
	if err := cursor.All(ctx, &items); err != nil {
		return nil, err
	}
	return items, nil
}

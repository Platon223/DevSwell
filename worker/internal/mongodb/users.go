package mongodb

import (
	"context"

	"github.com/Platon223/DevSwell/worker/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

const usersCollection = "users"

type UserReader struct {
	collection *mongo.Collection
}

func NewUserReader(client *mongo.Client) *UserReader {
	return &UserReader{
		collection: client.Database("devswell").Collection(usersCollection),
	}
}

// FindByTechnology returns verified users whose stack includes technology.
func (r *UserReader) FindByTechnology(ctx context.Context, technology string) ([]domain.User, error) {
	cursor, err := r.collection.Find(ctx, bson.M{"stack": technology, "verified": true})
	if err != nil {
		return nil, err
	}

	users := []domain.User{}
	if err := cursor.All(ctx, &users); err != nil {
		return nil, err
	}
	return users, nil
}

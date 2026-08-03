package mongodb

import (
	"context"

	"github.com/Platon223/DevSwell/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

const usersCollection = "users"

type UserStore struct {
	collection *mongo.Collection
}

func NewUserStore(client *mongo.Client) *UserStore {
	return &UserStore{
		collection: client.Database("devswell").Collection(usersCollection),
	}
}

func (s *UserStore) Insert(ctx context.Context, user domain.User) error {
	_, err := s.collection.InsertOne(ctx, user)
	return err
}

func (s *UserStore) FindByEmail(ctx context.Context, email string) (domain.User, error) {
	var user domain.User
	err := s.collection.FindOne(ctx, bson.M{"email": email}).Decode(&user)
	return user, err
}

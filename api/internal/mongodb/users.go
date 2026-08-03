package mongodb

import (
	"context"

	"github.com/Platon223/DevSwell/api/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
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

// EnsureIndexes creates a unique index on email so signup can't create two
// accounts with the same address.
func (s *UserStore) EnsureIndexes(ctx context.Context) error {
	_, err := s.collection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "email", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	return err
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

func (s *UserStore) FindByID(ctx context.Context, id bson.ObjectID) (domain.User, error) {
	var user domain.User
	err := s.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&user)
	return user, err
}

func (s *UserStore) FindByVerificationToken(ctx context.Context, token string) (domain.User, error) {
	var user domain.User
	err := s.collection.FindOne(ctx, bson.M{"verification_token": token}).Decode(&user)
	return user, err
}

func (s *UserStore) MarkVerified(ctx context.Context, id bson.ObjectID) error {
	_, err := s.collection.UpdateOne(ctx, bson.M{"_id": id}, bson.M{
		"$set": bson.M{"verified": true, "verification_token": ""},
	})
	return err
}

func (s *UserStore) UpdateStack(ctx context.Context, id bson.ObjectID, stack []string) error {
	_, err := s.collection.UpdateOne(ctx, bson.M{"_id": id}, bson.M{
		"$set": bson.M{"stack": stack},
	})
	return err
}

func (s *UserStore) SetDeletionToken(ctx context.Context, id bson.ObjectID, token string) error {
	_, err := s.collection.UpdateOne(ctx, bson.M{"_id": id}, bson.M{
		"$set": bson.M{"deletion_token": token},
	})
	return err
}

func (s *UserStore) FindByDeletionToken(ctx context.Context, token string) (domain.User, error) {
	var user domain.User
	err := s.collection.FindOne(ctx, bson.M{"deletion_token": token}).Decode(&user)
	return user, err
}

func (s *UserStore) Delete(ctx context.Context, id bson.ObjectID) error {
	_, err := s.collection.DeleteOne(ctx, bson.M{"_id": id})
	return err
}

package mongodb

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/Platon223/DevSwell/api/domain"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestUserStoreInsertAndReadBack(t *testing.T) {
	_ = godotenv.Load("../../.env")

	ctx := context.Background()
	client, err := Connect(ctx)
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer client.Disconnect(ctx)

	store := NewUserStore(client)

	email := fmt.Sprintf("day11-test-%d@example.com", time.Now().UnixNano())
	defer store.collection.DeleteOne(ctx, bson.M{"email": email})

	want := domain.User{
		Email:        email,
		PasswordHash: "fake-bcrypt-hash-for-testing",
		Stack:        []string{"Go", "MongoDB"},
		Plan:         "free",
	}
	if err := store.Insert(ctx, want); err != nil {
		t.Fatalf("Insert: %v", err)
	}

	got, err := store.FindByEmail(ctx, email)
	if err != nil {
		t.Fatalf("FindByEmail: %v", err)
	}

	got.ID = bson.ObjectID{} // ignore the DB-generated ID for comparison
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("read-back user does not match what was inserted:\ngot:  %+v\nwant: %+v", got, want)
	}
}

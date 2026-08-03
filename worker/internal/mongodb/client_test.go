package mongodb

import (
	"context"
	"testing"

	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type connectivityCheck struct {
	ID    bson.ObjectID `bson:"_id,omitempty"`
	Label string        `bson:"label"`
}

func TestConnectWriteRead(t *testing.T) {
	_ = godotenv.Load("../../.env")

	ctx := context.Background()
	client, err := Connect(ctx)
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer client.Disconnect(ctx)

	collection := client.Database("devswell").Collection("_connectivity_check")

	want := connectivityCheck{Label: "day2-connectivity-check"}
	result, err := collection.InsertOne(ctx, want)
	if err != nil {
		t.Fatalf("InsertOne: %v", err)
	}
	id := result.InsertedID
	defer collection.DeleteOne(ctx, bson.M{"_id": id})

	var got connectivityCheck
	if err := collection.FindOne(ctx, bson.M{"_id": id}).Decode(&got); err != nil {
		t.Fatalf("FindOne: %v", err)
	}

	if got.Label != want.Label {
		t.Fatalf("got label %q, want %q", got.Label, want.Label)
	}
}

package domain

import "go.mongodb.org/mongo-driver/v2/bson"

type User struct {
	ID           bson.ObjectID `bson:"_id,omitempty"`
	Email        string        `bson:"email"`
	PasswordHash string        `bson:"password_hash"`
	Stack        []string      `bson:"stack"`
	Plan         string        `bson:"plan"`
}

package domain

import "go.mongodb.org/mongo-driver/v2/bson"

type User struct {
	ID                        bson.ObjectID `bson:"_id,omitempty"`
	Email                     string        `bson:"email"`
	PasswordHash              string        `bson:"password_hash"`
	Stack                     []string      `bson:"stack"`
	Plan                      string        `bson:"plan"`
	Verified                  bool          `bson:"verified"`
	VerificationToken         string        `bson:"verification_token,omitempty"`
	DeletionToken             string        `bson:"deletion_token,omitempty"`
	EmailNotificationsEnabled bool          `bson:"email_notifications_enabled"`
	UnsubscribeToken          string        `bson:"unsubscribe_token,omitempty"`
}

package entities

import "go.mongodb.org/mongo-driver/v2/bson"

type User struct {
	ID         bson.ObjectID          `bson:"_id,omitempty"           json:"-"`
	UserID     string                 `bson:"user_id"                 json:"user_id" validate:"required"`
	Metadata   map[string]interface{} `bson:"metadata,omitempty"      json:"metadata,omitempty"`
	PublicKey  *string                `bson:"public_key,omitempty"    json:"public_key,omitempty"`
	SyncedAt   *int64                 `bson:"synced_at,omitempty"     json:"synced_at,omitempty"`
	CreatedAt  int64                  `bson:"created_at"              json:"created_at" validate:"required"`
	UpdatedAt  int64                  `bson:"updated_at"              json:"updated_at" validate:"required"`
	LastSeenAt int64                  `bson:"last_seen_at"            json:"last_seen_at" validate:"required"`
}

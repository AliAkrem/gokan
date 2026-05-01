package entities

import "go.mongodb.org/mongo-driver/v2/bson"

type Room struct {
	ID           bson.ObjectID          `bson:"_id,omitempty"          json:"-"`
	RoomID       string                 `bson:"room_id"                 json:"room_id"       validate:"required"`
	Participants []string               `bson:"participants"           json:"participants" validate:"required,len=2"`
	Type         string                 `bson:"type"                   json:"type"         validate:"required,oneof=1-to-1"`
	LastMessages map[string]interface{} `bson:"last_messages,omitempty" json:"last_messages,omitempty"`
	Metadata     map[string]interface{} `bson:"metadata,omitempty"     json:"metadata,omitempty"`
	CreatedAt    int64                  `bson:"created_at"              json:"created_at"    validate:"required"`
	UpdatedAt    int64                  `bson:"updated_at"              json:"updated_at"    validate:"required"`
	DeletedAt    *int64                 `bson:"deleted_at,omitempty"    json:"deleted_at,omitempty"`
}

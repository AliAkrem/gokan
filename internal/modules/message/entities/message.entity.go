package entities

import "go.mongodb.org/mongo-driver/v2/bson"

type MessageStatus string

const (
	MessageStatusSent      MessageStatus = "sent"
	MessageStatusDelivered MessageStatus = "delivered"
	MessageStatusRead      MessageStatus = "read"
)

type MessageType string

const (
	MessageTypeText   MessageType = "text"
	MessageTypeBinary MessageType = "binary"
)

type Message struct {
	ID             bson.ObjectID          `bson:"_id,omitempty"            json:"-"`
	MsgID          string                 `bson:"msg_id"                    json:"msg_id"          validate:"required,uuid"`
	ClientMsgID    string                 `bson:"client_msg_id"              json:"client_msg_id"    validate:"required"`
	RoomID         string                 `bson:"room_id"                   json:"room_id"         validate:"required"`
	AuthorID       string                 `bson:"author_id"                 json:"author_id"       validate:"required"`
	Text           *string                `bson:"text,omitempty"           json:"text,omitempty"`
	Type           MessageType            `bson:"type"                     json:"type"           validate:"required,oneof=text binary"`
	Status         MessageStatus          `bson:"status"                   json:"status"         validate:"required,oneof=sent delivered read"`
	RepliedMessage map[string]interface{} `bson:"replied_message,omitempty" json:"replied_message,omitempty"`
	Metadata       map[string]interface{} `bson:"metadata,omitempty"       json:"metadata,omitempty"`
	CreatedAt      int64                  `bson:"created_at"                json:"created_at"      validate:"required"`
	UpdatedAt      int64                  `bson:"updated_at"                json:"updated_at"      validate:"required"`
	DeletedAt      *int64                 `bson:"deleted_at,omitempty"      json:"deleted_at,omitempty"`
}

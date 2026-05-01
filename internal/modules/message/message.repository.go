package message

import (
	"context"
	"fmt"
	"time"

	"github.com/aliakrem/gokan/internal/modules/message/entities"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var (
	ErrNotFound = fmt.Errorf("resource not found")
)

type Repository struct {
	collection *mongo.Collection
}

func (r *Repository) Collection() *mongo.Collection {
	return r.collection
}

func NewRepository(db *mongo.Database) (*Repository, error) {
	collection := db.Collection("messages")

	_, err := collection.Indexes().CreateOne(context.Background(), mongo.IndexModel{
		Keys:    bson.D{{Key: "msg_id", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create msg_id index: %w", err)
	}

	_, err = collection.Indexes().CreateOne(context.Background(), mongo.IndexModel{
		Keys: bson.D{
			{Key: "room_id", Value: 1},
			{Key: "created_at", Value: -1},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create roomId+createdAt index: %w", err)
	}

	_, err = collection.Indexes().CreateOne(context.Background(), mongo.IndexModel{
		Keys: bson.D{
			{Key: "client_msg_id", Value: 1},
			{Key: "room_id", Value: 1},
		},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create client_msg_id+roomId index: %w", err)
	}

	return &Repository{collection: collection}, nil
}

func (r *Repository) Create(ctx context.Context, msg *entities.Message) error {
	now := time.Now().UnixMilli()

	existing, err := r.FindByClientMsgID(ctx, msg.RoomID, msg.ClientMsgID)
	if err == nil && existing != nil {
		*msg = *existing
		return nil
	}

	msg.MsgID = uuid.New().String()
	msg.CreatedAt = now
	msg.UpdatedAt = now

	if msg.Status == "" {
		msg.Status = entities.MessageStatusSent
	}

	_, err = r.collection.InsertOne(ctx, msg)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			existing, fetchErr := r.FindByClientMsgID(ctx, msg.RoomID, msg.ClientMsgID)
			if fetchErr == nil {
				*msg = *existing
				return nil
			}
		}
		return fmt.Errorf("failed to create message: %w", err)
	}

	return nil
}

func (r *Repository) FindByID(ctx context.Context, msgID string) (*entities.Message, error) {
	var msg entities.Message
	err := r.collection.FindOne(ctx, bson.M{"msg_id": msgID}).Decode(&msg)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("%w: message not found: %s", ErrNotFound, msgID)
		}
		return nil, fmt.Errorf("failed to find message: %w", err)
	}

	return &msg, nil
}

func (r *Repository) FindByClientMsgID(ctx context.Context, roomID, clientMsgID string) (*entities.Message, error) {
	var msg entities.Message
	err := r.collection.FindOne(ctx, bson.M{
		"room_id":       roomID,
		"client_msg_id": clientMsgID,
	}).Decode(&msg)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("%w: message not found", ErrNotFound)
		}
		return nil, fmt.Errorf("failed to find message: %w", err)
	}

	return &msg, nil
}

func (r *Repository) ListByRoom(ctx context.Context, roomID string, limit int, before, after string) ([]*entities.Message, error) {
	filter := bson.M{
		"room_id":    roomID,
		"deleted_at": nil,
		"text":       bson.M{"$ne": nil},
	}

	if before != "" || after != "" {
		msgIDFilter := bson.M{}
		if before != "" {
			msgIDFilter["$lt"] = before
		}
		if after != "" {
			msgIDFilter["$gt"] = after
		}
		filter["msg_id"] = msgIDFilter
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}).
		SetLimit(int64(limit))

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list messages: %w", err)
	}
	defer cursor.Close(ctx)

	var messages []*entities.Message
	if err := cursor.All(ctx, &messages); err != nil {
		return nil, fmt.Errorf("failed to decode messages: %w", err)
	}

	return messages, nil
}

func (r *Repository) UpdateStatus(ctx context.Context, msgID string, status entities.MessageStatus) error {
	now := time.Now().UnixMilli()

	filter := bson.M{"msg_id": msgID}
	update := bson.M{
		"$set": bson.M{
			"status":     status,
			"updated_at": now,
		},
	}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update message status: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("%w: message not found: %s", ErrNotFound, msgID)
	}

	return nil
}

func (r *Repository) SoftDelete(ctx context.Context, msgID string) error {
	now := time.Now().UnixMilli()

	filter := bson.M{"msg_id": msgID}
	update := bson.M{
		"$set": bson.M{
			"deleted_at": now,
			"updated_at": now,
			"text":       nil,
		},
	}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to soft delete message: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("%w: message not found: %s", ErrNotFound, msgID)
	}

	return nil
}

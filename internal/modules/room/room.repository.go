package room

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"time"

	"github.com/aliakrem/gokan/internal/modules/message/entities"
	roomEntities "github.com/aliakrem/gokan/internal/modules/room/entities"
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

func NewRepository(db *mongo.Database) (*Repository, error) {
	collection := db.Collection("rooms")

	_, err := collection.Indexes().CreateOne(context.Background(), mongo.IndexModel{
		Keys:    bson.D{{Key: "room_id", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create roomId index: %w", err)
	}

	_, err = collection.Indexes().CreateOne(context.Background(), mongo.IndexModel{
		Keys: bson.D{{Key: "participants", Value: 1}},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create participants index: %w", err)
	}

	_, err = collection.Indexes().CreateOne(context.Background(), mongo.IndexModel{
		Keys: bson.D{{Key: "updated_at", Value: -1}},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create updated_at index: %w", err)
	}

	return &Repository{collection: collection}, nil
}

func GenerateRoomID(participants []string) string {
	sorted := make([]string, len(participants))
	copy(sorted, participants)
	sort.Strings(sorted)

	hash := sha256.New()
	for i, p := range sorted {
		hash.Write([]byte(p))
		if i < len(sorted)-1 {
			hash.Write([]byte(":"))
		}
	}

	return hex.EncodeToString(hash.Sum(nil))
}

func (r *Repository) Create(ctx context.Context, room *roomEntities.Room) error {
	now := time.Now().UnixMilli()

	if len(room.Participants) != 2 {
		return fmt.Errorf("room must have exactly 2 participants")
	}

	if room.Participants[0] == room.Participants[1] {
		return fmt.Errorf("participants must be different users")
	}

	sort.Strings(room.Participants)
	room.RoomID = GenerateRoomID(room.Participants)

	existing, err := r.FindByID(ctx, room.RoomID)
	if err == nil && existing != nil {
		*room = *existing
		return nil
	}

	room.CreatedAt = now
	room.UpdatedAt = now
	room.Type = "1-to-1"

	_, err = r.collection.InsertOne(ctx, room)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			existing, fetchErr := r.FindByID(ctx, room.RoomID)
			if fetchErr == nil {
				*room = *existing
				return nil
			}
		}
		return fmt.Errorf("failed to create room: %w", err)
	}

	return nil
}

func (r *Repository) FindByID(ctx context.Context, roomID string) (*roomEntities.Room, error) {
	var room roomEntities.Room
	err := r.collection.FindOne(ctx, bson.M{"room_id": roomID, "deleted_at": nil}).Decode(&room)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("%w: room not found: %s", ErrNotFound, roomID)
		}
		return nil, fmt.Errorf("failed to find room: %w", err)
	}

	return &room, nil
}

func (r *Repository) FindByParticipants(ctx context.Context, userIDs []string) (*roomEntities.Room, error) {
	roomID := GenerateRoomID(userIDs)
	return r.FindByID(ctx, roomID)
}

func (r *Repository) ListByUser(ctx context.Context, userID string, limit int, cursor string) ([]*roomEntities.Room, error) {
	filter := bson.M{
		"participants": userID,
		"deleted_at":   nil,
	}

	if cursor != "" {
		filter["room_id"] = bson.M{"$lt": cursor}
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "updated_at", Value: -1}}).
		SetLimit(int64(limit))

	cursor2, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list rooms: %w", err)
	}
	defer cursor2.Close(ctx)

	var rooms []*roomEntities.Room
	if err := cursor2.All(ctx, &rooms); err != nil {
		return nil, fmt.Errorf("failed to decode rooms: %w", err)
	}

	return rooms, nil
}

func (r *Repository) SoftDelete(ctx context.Context, roomID string) error {
	now := time.Now().UnixMilli()

	filter := bson.M{"room_id": roomID}
	update := bson.M{
		"$set": bson.M{
			"deleted_at": now,
			"updated_at": now,
		},
	}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to soft delete room: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("%w: room not found: %s", ErrNotFound, roomID)
	}

	return nil
}

func (r *Repository) UpdateLastMessage(ctx context.Context, roomID string, msg *entities.Message) error {
	now := time.Now().UnixMilli()

	lastMessage := map[string]interface{}{
		"msg_id":     msg.MsgID,
		"author_id":  msg.AuthorID,
		"text":       msg.Text,
		"type":       msg.Type,
		"created_at": msg.CreatedAt,
	}

	filter := bson.M{"room_id": roomID}
	update := bson.M{
		"$set": bson.M{
			"last_messages": lastMessage,
			"updated_at":    now,
		},
	}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update lastMessage: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("%w: room not found: %s", ErrNotFound, roomID)
	}

	return nil
}

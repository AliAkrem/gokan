package user

import (
	"context"
	"fmt"
	"time"

	"github.com/aliakrem/gokan/internal/modules/user/entities"
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
	collection := db.Collection("users")

	_, err := collection.Indexes().CreateOne(context.Background(), mongo.IndexModel{
		Keys:    bson.D{{Key: "user_id", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create user_id index: %w", err)
	}

	_, err = collection.Indexes().CreateOne(context.Background(), mongo.IndexModel{
		Keys: bson.D{{Key: "synced_at", Value: 1}},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create synced_at index: %w", err)
	}

	return &Repository{collection: collection}, nil
}

func (r *Repository) Upsert(ctx context.Context, user *entities.User) error {
	now := time.Now().UnixMilli()

	if user.CreatedAt == 0 {
		user.CreatedAt = now
	}
	user.UpdatedAt = now

	filter := bson.M{"user_id": user.UserID}

	setFields := bson.M{
		"user_id":    user.UserID,
		"updated_at": user.UpdatedAt,
	}

	if user.SyncedAt != nil {
		setFields["synced_at"] = user.SyncedAt
	}
	if user.LastSeenAt != 0 {
		setFields["last_seen_at"] = user.LastSeenAt
	}
	if user.Metadata != nil {
		setFields["metadata"] = user.Metadata
	}
	if user.PublicKey != nil {
		setFields["public_key"] = user.PublicKey
	}

	update := bson.M{
		"$set": setFields,
		"$setOnInsert": bson.M{
			"created_at": user.CreatedAt,
		},
	}

	opts := options.UpdateOne().SetUpsert(true)
	_, err := r.collection.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return fmt.Errorf("failed to upsert user: %w", err)
	}

	return nil
}

func (r *Repository) FindByID(ctx context.Context, userID string) (*entities.User, error) {
	var user entities.User
	err := r.collection.FindOne(ctx, bson.M{"user_id": userID}).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find user: %w", err)
	}

	return &user, nil
}

func (r *Repository) UpdateLastSeen(ctx context.Context, userID string) error {
	now := time.Now().UnixMilli()

	filter := bson.M{"user_id": userID}
	update := bson.M{
		"$set": bson.M{
			"last_seen_at": now,
			"updated_at":   now,
		},
	}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update last_seen_at: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("%w: user not found: %s", ErrNotFound, userID)
	}

	return nil
}

func (r *Repository) UpdatePublicKey(ctx context.Context, userID string, publicKey string) error {
	now := time.Now().UnixMilli()

	filter := bson.M{"user_id": userID}
	update := bson.M{
		"$set": bson.M{
			"public_key": publicKey,
			"updated_at": now,
		},
	}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update publicKey: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("%w: user not found: %s", ErrNotFound, userID)
	}

	return nil
}

func (r *Repository) List(ctx context.Context, limit int, cursor string) ([]*entities.User, error) {
	filter := bson.M{}

	if cursor != "" {
		filter["user_id"] = bson.M{"$gt": cursor}
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "user_id", Value: 1}}).
		SetLimit(int64(limit))

	cursorMongo, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to query users: %w", err)
	}
	defer cursorMongo.Close(ctx)

	var users []*entities.User
	if err := cursorMongo.All(ctx, &users); err != nil {
		return nil, fmt.Errorf("failed to decode users: %w", err)
	}

	if users == nil {
		users = make([]*entities.User, 0)
	}

	return users, nil
}

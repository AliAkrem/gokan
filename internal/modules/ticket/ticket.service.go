package ticket

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

type RedisClient interface {
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd
	GetDel(ctx context.Context, key string) *redis.StringCmd
	Close() error
}

type TicketService struct {
	redis RedisClient
	ttl   time.Duration
}

func NewTicketService(redisClient RedisClient, ttl time.Duration) *TicketService {
	return &TicketService{
		redis: redisClient,
		ttl:   ttl,
	}
}

func (s *TicketService) Close() error {
	if s.redis != nil {
		return s.redis.Close()
	}
	return nil
}

func generateTicket() (string, error) {
	bytes := make([]byte, 32) // 256 bits
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

func (s *TicketService) GenerateTicket(ctx context.Context, userID string, jwt string) (string, error) {
	ticket, err := generateTicket()
	if err != nil {
		log.Error().Err(err).Msg("failed to generate ticket")
		return "", err
	}

	payload := TicketPayload{
		UserID: userID,
		JWT:    jwt,
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		log.Error().Err(err).Msg("failed to marshal ticket payload")
		return "", fmt.Errorf("failed to marshal payload: %w", err)
	}

	key := fmt.Sprintf("ws:ticket:%s", ticket)
	err = s.redis.Set(ctx, key, payloadJSON, s.ttl).Err()
	if err != nil {
		log.Error().Err(err).Str("key", key).Msg("failed to store ticket in Redis")
		return "", fmt.Errorf("failed to store ticket: %w", err)
	}

	log.Debug().
		Str("user_id", userID).
		Dur("ttl", s.ttl).
		Time("expiresAt", time.Now().Add(s.ttl)).
		Msg("ticket issued")

	return ticket, nil
}

func (s *TicketService) ValidateTicket(ctx context.Context, ticket string) (string, string, error) {
	key := fmt.Sprintf("ws:ticket:%s", ticket)
	payloadJSON, err := s.redis.GetDel(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			log.Warn().Msg("ticket not found or expired")
			return "", "", fmt.Errorf("ticket not found or expired")
		}
		log.Error().Err(err).Str("key", key).Msg("failed to retrieve ticket from Redis")
		return "", "", fmt.Errorf("failed to retrieve ticket: %w", err)
	}

	var payload TicketPayload
	err = json.Unmarshal([]byte(payloadJSON), &payload)
	if err != nil {
		log.Error().Err(err).Str("key", key).Msg("failed to unmarshal ticket payload")
		return "", "", fmt.Errorf("failed to parse ticket payload: %w", err)
	}

	return payload.UserID, payload.JWT, nil
}

package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/aliakrem/gokan/internal/modules/user/entities"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/singleflight"
)

type UserRepository interface {
	Upsert(ctx context.Context, user *entities.User) error
	FindByID(ctx context.Context, userID string) (*entities.User, error)
	UpdateLastSeen(ctx context.Context, userID string) error
}

type UserSyncConfig struct {
	UserInfoURL string
	SyncTTL     time.Duration
	HTTPClient  *http.Client
}

type UserSyncMiddleware struct {
	config     UserSyncConfig
	userRepo   UserRepository
	httpClient *http.Client
	sfGroup    singleflight.Group
}

func NewUserSyncMiddleware(config UserSyncConfig, userRepo UserRepository) *UserSyncMiddleware {

	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: 10 * time.Second,
		}
	}

	return &UserSyncMiddleware{
		config:     config,
		userRepo:   userRepo,
		httpClient: httpClient,
	}
}

func (m *UserSyncMiddleware) SyncUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()

		userIDVal, exists := c.Get("user_id")
		if !exists {
			log.Error().Msg("user_id not found in context")
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Internal Server Error",
				"code":    "INTERNAL_ERROR",
				"details": map[string]interface{}{"reason": "user_id not found in context"},
			})
			c.Abort()
			return
		}

		userID, ok := userIDVal.(string)
		if !ok || userID == "" {
			log.Error().Msg("invalid user_id in context")
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Internal Server Error",
				"code":    "INTERNAL_ERROR",
				"details": map[string]interface{}{"reason": "invalid user_id in context"},
			})
			c.Abort()
			return
		}

		jwtVal, _ := c.Get("jwt")
		originalJWT, _ := jwtVal.(string)

		user, err := m.userRepo.FindByID(ctx, userID)
		if err != nil {
			log.Error().Err(err).Str("user_id", userID).Msg("failed to query user from DB")
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Internal Server Error",
				"code":  "DATABASE_ERROR",
			})
			c.Abort()
			return
		}

		now := time.Now().UnixMilli()

		syncNeeded := false
		if user == nil {
			syncNeeded = true
			log.Debug().Str("user_id", userID).Msg("user not found, sync needed")
		} else if user.SyncedAt == nil {
			syncNeeded = true
			log.Debug().Str("user_id", userID).Msg("user never synced, sync needed")
		} else if m.config.SyncTTL > 0 && (now-*user.SyncedAt) > m.config.SyncTTL.Milliseconds() {
			syncNeeded = true
			log.Debug().Str("user_id", userID).Int64("syncAge", now-*user.SyncedAt).Msg("user sync stale, sync needed")
		}

		syncedSynchronously := false

		if syncNeeded {
			if m.config.UserInfoURL != "" {
				if user != nil {
					log.Debug().Str("user_id", userID).Msg("user sync stale, refreshing in background")
					bgCtx := context.WithoutCancel(ctx)
					go func() {
						syncErr := m.syncUserMetadata(bgCtx, userID, originalJWT)
						if syncErr != nil {
							log.Warn().
								Err(syncErr).
								Str("user_id", userID).
								Str("userInfoURL", m.config.UserInfoURL).
								Msg("background user sync failed")
						}
					}()
				} else {
					syncErr := m.syncUserMetadata(ctx, userID, originalJWT)
					if syncErr != nil {
						log.Error().
							Err(syncErr).
							Str("user_id", userID).
							Str("userInfoURL", m.config.UserInfoURL).
							Msg("user sync failed for new user")
						c.JSON(http.StatusServiceUnavailable, gin.H{
							"error":   "Service Unavailable",
							"code":    "USER_SYNC_FAILED",
							"details": map[string]interface{}{"user_id": userID},
						})
						c.Abort()
						return
					}
					syncedSynchronously = true
				}
			} else {
				// USER_INFO_URL not configured - create minimal user record
				log.Debug().Str("user_id", userID).Msg("USER_INFO_URL not configured, creating minimal user")
				minimalUser := &entities.User{
					UserID:     userID,
					SyncedAt:   &now,
					CreatedAt:  now,
					UpdatedAt:  now,
					LastSeenAt: now,
				}
				if err := m.userRepo.Upsert(ctx, minimalUser); err != nil {
					log.Error().Err(err).Str("user_id", userID).Msg("failed to create minimal user")
					c.JSON(http.StatusInternalServerError, gin.H{
						"error":   "Internal Server Error",
						"code":    "DATABASE_ERROR",
						"details": map[string]interface{}{"reason": "failed to create user"},
					})
					c.Abort()
					return
				}
				syncedSynchronously = true
			}
		}

		// Update last_seen_at on every request
		if !syncedSynchronously {
			if err := m.userRepo.UpdateLastSeen(ctx, userID); err != nil {
				log.Warn().Err(err).Str("user_id", userID).Msg("failed to update last_seen_at")
				// Don't abort request for last_seen_at update failure
			}
		}

		// Continue to next handler
		c.Next()
	}
}

func (m *UserSyncMiddleware) syncUserMetadata(ctx context.Context, userID string, jwt string) error {
	_, err, _ := m.sfGroup.Do(userID, func() (interface{}, error) {

		req, err := http.NewRequestWithContext(ctx, "GET", m.config.UserInfoURL, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Authorization", "Bearer "+jwt)

		resp, err := m.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to call USER_INFO_URL: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 400 {
			return nil, fmt.Errorf("USER_INFO_URL returned error status: %d", resp.StatusCode)
		}

		var metadata map[string]interface{}
		limitedBody := io.LimitReader(resp.Body, 1<<20) // 1 MB max
		if err := json.NewDecoder(limitedBody).Decode(&metadata); err != nil {
			return nil, fmt.Errorf("failed to decode USER_INFO_URL response: %w", err)
		}

		now := time.Now().UnixMilli()
		user := &entities.User{
			UserID:     userID,
			Metadata:   metadata,
			SyncedAt:   &now,
			UpdatedAt:  now,
			LastSeenAt: now,
		}

		if err := m.userRepo.Upsert(ctx, user); err != nil {
			return nil, fmt.Errorf("failed to upsert user: %w", err)
		}

		log.Info().Str("user_id", userID).Msg("user metadata synced successfully")
		return nil, nil
	})

	return err
}

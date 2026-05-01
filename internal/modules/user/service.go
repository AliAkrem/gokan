package user

import (
	"context"

	"github.com/aliakrem/gokan/internal/modules/user/entities"
	apperrors "github.com/aliakrem/gokan/internal/shared/errors"
	"github.com/go-playground/validator/v10"
	"github.com/rs/zerolog"
)

type Service interface {
	CreateUser(ctx context.Context, req CreateUserRequest, callerID string) (*entities.User, error)
	GetUser(ctx context.Context, userID string) (*entities.User, error)
	UpdateUser(ctx context.Context, userID string, req UpdateUserRequest, callerID string) (*entities.User, error)
	GetPublicKey(ctx context.Context, userID string) (string, error)
	ListUsers(ctx context.Context, limit int, cursor string) ([]*entities.User, error)
}

type UserService struct {
	userRepo *Repository
	validate *validator.Validate
	logger   *zerolog.Logger
}

func NewUserService(userRepo *Repository, logger *zerolog.Logger) Service {
	return &UserService{
		userRepo: userRepo,
		validate: validator.New(),
		logger:   logger,
	}
}

func (s *UserService) CreateUser(ctx context.Context, req CreateUserRequest, callerID string) (*entities.User, error) {
	if err := s.validate.Struct(req); err != nil {
		s.logger.Warn().Err(err).Msg("validation failed")
		return nil, apperrors.NewInvalidPayloadError("validation failed", map[string]interface{}{
			"reason": err.Error(),
		})
	}

	if callerID != req.UserID {
		return nil, apperrors.NewForbiddenError("forbidden: cannot modify other users")
	}

	user := &entities.User{
		UserID:    req.UserID,
		Metadata:  req.Metadata,
		PublicKey: req.PublicKey,
	}

	if err := s.userRepo.Upsert(ctx, user); err != nil {
		s.logger.Error().Err(err).Str("user_id", req.UserID).Msg("failed to upsert user")
		return nil, apperrors.NewInternalError("failed to create user").WithError(err)
	}

	createdUser, err := s.userRepo.FindByID(ctx, req.UserID)
	if err != nil || createdUser == nil {
		s.logger.Error().Err(err).Str("user_id", req.UserID).Msg("failed to fetch user after upsert")
		return nil, apperrors.NewInternalError("failed to fetch user").WithError(err)
	}

	return createdUser, nil
}

func (s *UserService) GetUser(ctx context.Context, userID string) (*entities.User, error) {
	if userID == "" {
		return nil, apperrors.NewInvalidPayloadError("user_id is required", nil)
	}

	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		s.logger.Error().Err(err).Str("user_id", userID).Msg("failed to query user")
		return nil, apperrors.NewInternalError("failed to query user").WithError(err)
	}
	if user == nil {
		s.logger.Warn().Str("user_id", userID).Msg("user not found")
		return nil, apperrors.NewUserNotFoundError(userID)
	}

	return user, nil
}

func (s *UserService) UpdateUser(ctx context.Context, userID string, req UpdateUserRequest, callerID string) (*entities.User, error) {
	if userID == "" {
		return nil, apperrors.NewInvalidPayloadError("user_id is required", nil)
	}

	if callerID != userID {
		return nil, apperrors.NewForbiddenError("forbidden: cannot modify other users")
	}

	if err := s.validate.Struct(req); err != nil {
		s.logger.Warn().Err(err).Msg("validation failed")
		return nil, apperrors.NewInvalidPayloadError("validation failed", map[string]interface{}{
			"reason": err.Error(),
		})
	}

	if err := s.userRepo.UpdatePublicKey(ctx, userID, req.PublicKey); err != nil {
		s.logger.Error().Err(err).Str("user_id", userID).Msg("failed to update publicKey")
		return nil, apperrors.NewInternalError("failed to update user").WithError(err)
	}

	updatedUser, err := s.userRepo.FindByID(ctx, userID)
	if err != nil || updatedUser == nil {
		s.logger.Error().Err(err).Str("user_id", userID).Msg("failed to fetch user after update")
		return nil, apperrors.NewInternalError("failed to fetch user").WithError(err)
	}

	return updatedUser, nil
}

func (s *UserService) GetPublicKey(ctx context.Context, userID string) (string, error) {
	if userID == "" {
		return "", apperrors.NewInvalidPayloadError("user_id is required", nil)
	}

	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		s.logger.Error().Err(err).Str("user_id", userID).Msg("failed to query user")
		return "", apperrors.NewInternalError("failed to query user").WithError(err)
	}
	if user == nil {
		s.logger.Warn().Str("user_id", userID).Msg("user not found")
		return "", apperrors.NewUserNotFoundError(userID)
	}

	publicKey := ""
	if user.PublicKey != nil {
		publicKey = *user.PublicKey
	}

	return publicKey, nil
}

func (s *UserService) ListUsers(ctx context.Context, limit int, cursor string) ([]*entities.User, error) {
	if limit <= 0 {
		return nil, apperrors.NewInvalidPayloadError("invalid limit parameter", nil)
	}

	users, err := s.userRepo.List(ctx, limit, cursor)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to list users")
		return nil, apperrors.NewInternalError("failed to list users").WithError(err)
	}

	return users, nil
}

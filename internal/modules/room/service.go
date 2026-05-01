package room

import (
	"context"

	"github.com/aliakrem/gokan/internal/modules/room/entities"
	userEntities "github.com/aliakrem/gokan/internal/modules/user/entities"
	apperrors "github.com/aliakrem/gokan/internal/shared/errors"
	"github.com/go-playground/validator/v10"
	"github.com/rs/zerolog"
)

type UserRepository interface {
	FindByID(ctx context.Context, userID string) (*userEntities.User, error)
}

type Service interface {
	CreateRoom(ctx context.Context, req CreateRoomRequest, callerID string) (*entities.Room, error)
	GetRoom(ctx context.Context, roomID string, callerID string) (*entities.Room, error)
	DeleteRoom(ctx context.Context, roomID string, callerID string) error
	GetUserRooms(ctx context.Context, userID string, callerID string, limit int, cursor string) ([]*entities.Room, error)
}

type RoomService struct {
	roomRepo *Repository
	userRepo UserRepository
	validate *validator.Validate
	logger   *zerolog.Logger
}

func NewRoomService(roomRepo *Repository, userRepo UserRepository, logger *zerolog.Logger) Service {
	return &RoomService{
		roomRepo: roomRepo,
		userRepo: userRepo,
		validate: validator.New(),
		logger:   logger,
	}
}

func (s *RoomService) CreateRoom(ctx context.Context, req CreateRoomRequest, callerID string) (*entities.Room, error) {
	if err := s.validate.Struct(req); err != nil {
		s.logger.Warn().Err(err).Msg("validation failed")
		return nil, apperrors.NewInvalidPayloadError("validation failed", map[string]interface{}{
			"reason": err.Error(),
		})
	}

	if len(req.Participants) != 2 {
		return nil, apperrors.NewInvalidPayloadError("room must have exactly 2 participants", nil)
	}

	if req.Participants[0] == req.Participants[1] {
		return nil, apperrors.NewInvalidPayloadError("participants must be different users", nil)
	}

	if req.Participants[0] != callerID && req.Participants[1] != callerID {
		return nil, apperrors.NewForbiddenError("forbidden: must be a participant to create a room")
	}

	for _, userID := range req.Participants {
		user, err := s.userRepo.FindByID(ctx, userID)
		if err != nil {
			s.logger.Error().Err(err).Str("user_id", userID).Msg("failed to verify user")
			return nil, apperrors.NewInternalError("failed to verify user").WithError(err)
		}
		if user == nil {
			s.logger.Warn().Str("user_id", userID).Msg("user not found")
			return nil, apperrors.NewUserNotFoundError(userID)
		}
	}

	room := &entities.Room{
		Participants: req.Participants,
		Metadata:     req.Metadata,
	}

	if err := s.roomRepo.Create(ctx, room); err != nil {
		s.logger.Error().Err(err).Strs("participants", req.Participants).Msg("failed to create room")
		return nil, apperrors.NewInternalError("failed to create room").WithError(err)
	}

	return room, nil
}

func (s *RoomService) GetRoom(ctx context.Context, roomID string, callerID string) (*entities.Room, error) {
	if roomID == "" {
		return nil, apperrors.NewInvalidPayloadError("roomId is required", nil)
	}

	room, err := s.roomRepo.FindByID(ctx, roomID)
	if err != nil {
		s.logger.Warn().Err(err).Str("room_id", roomID).Msg("room not found")
		return nil, apperrors.NewRoomNotFoundError(roomID)
	}

	isParticipant := false
	for _, p := range room.Participants {
		if p == callerID {
			isParticipant = true
			break
		}
	}
	if !isParticipant {
		return nil, apperrors.NewForbiddenError("forbidden: not a participant of this room")
	}

	return room, nil
}

func (s *RoomService) DeleteRoom(ctx context.Context, roomID string, callerID string) error {
	if roomID == "" {
		return apperrors.NewInvalidPayloadError("roomId is required", nil)
	}

	room, err := s.roomRepo.FindByID(ctx, roomID)
	if err != nil {
		s.logger.Warn().Err(err).Str("room_id", roomID).Msg("room not found for deletion")
		return apperrors.NewRoomNotFoundError(roomID)
	}

	isParticipant := false
	for _, p := range room.Participants {
		if p == callerID {
			isParticipant = true
			break
		}
	}
	if !isParticipant {
		return apperrors.NewForbiddenError("forbidden: not a participant of this room")
	}

	if err := s.roomRepo.SoftDelete(ctx, roomID); err != nil {
		s.logger.Error().Err(err).Str("room_id", roomID).Msg("failed to delete room")
		return apperrors.NewInternalError("failed to delete room").WithError(err)
	}

	return nil
}

func (s *RoomService) GetUserRooms(ctx context.Context, userID string, callerID string, limit int, cursor string) ([]*entities.Room, error) {
	if userID == "" {
		return nil, apperrors.NewInvalidPayloadError("user_id is required", nil)
	}

	if callerID != userID {
		return nil, apperrors.NewForbiddenError("forbidden: cannot view other users' rooms")
	}

	if limit <= 0 {
		return nil, apperrors.NewInvalidPayloadError("invalid limit parameter", nil)
	}

	rooms, err := s.roomRepo.ListByUser(ctx, userID, limit, cursor)
	if err != nil {
		s.logger.Error().Err(err).Str("user_id", userID).Msg("failed to list rooms")
		return nil, apperrors.NewInternalError("failed to list rooms").WithError(err)
	}

	return rooms, nil
}

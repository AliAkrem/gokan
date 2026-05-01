package message

import (
	"context"
	"errors"

	"github.com/aliakrem/gokan/internal/modules/message/entities"
	roomEntities "github.com/aliakrem/gokan/internal/modules/room/entities"
	apperrors "github.com/aliakrem/gokan/internal/shared/errors"
	"github.com/rs/zerolog"
)

type RoomRepository interface {
	FindByID(ctx context.Context, roomID string) (*roomEntities.Room, error)
}

type Service interface {
	GetRoomMessages(ctx context.Context, roomID string, callerID string, limit int, before, after string) ([]*entities.Message, error)
	DeleteMessage(ctx context.Context, msgID string, callerID string) error
}

type MessageService struct {
	messageRepo *Repository
	roomRepo    RoomRepository
	logger      *zerolog.Logger
}

type MessageWithRoom struct {
	*entities.Message
	Room *roomEntities.Room
}

func NewMessageService(messageRepo *Repository, roomRepo RoomRepository, logger *zerolog.Logger) Service {
	return &MessageService{
		messageRepo: messageRepo,
		roomRepo:    roomRepo,
		logger:      logger,
	}
}

func (s *MessageService) GetRoomMessages(ctx context.Context, roomID string, callerID string, limit int, before, after string) ([]*entities.Message, error) {
	if roomID == "" {
		return nil, apperrors.NewInvalidPayloadError("roomId is required", nil)
	}

	if limit <= 0 {
		return nil, apperrors.NewInvalidPayloadError("invalid limit parameter", nil)
	}

	room, err := s.roomRepo.FindByID(ctx, roomID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, apperrors.NewRoomNotFoundError(roomID)
		}
		s.logger.Warn().Err(err).Str("room_id", roomID).Msg("failed to find room")
		return nil, apperrors.NewInternalError("failed to retrieve room").WithError(err)
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

	messages, err := s.messageRepo.ListByRoom(ctx, roomID, limit, before, after)
	if err != nil {
		s.logger.Error().Err(err).Str("room_id", roomID).Msg("failed to list messages")
		return nil, apperrors.NewInternalError("failed to list messages").WithError(err)
	}

	if messages == nil {
		messages = []*entities.Message{}
	}

	return messages, nil
}

func (s *MessageService) DeleteMessage(ctx context.Context, msgID string, callerID string) error {
	if msgID == "" {
		return apperrors.NewInvalidPayloadError("msg_id is required", nil)
	}

	msg, err := s.messageRepo.FindByID(ctx, msgID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return apperrors.NewMessageNotFoundError(msgID)
		}
		s.logger.Error().Err(err).Str("msg_id", msgID).Msg("failed to fetch message")
		return apperrors.NewInternalError("failed to retrieve message").WithError(err)
	}

	if msg.AuthorID != callerID {
		return apperrors.NewForbiddenError("forbidden: cannot delete other user's message")
	}

	if err := s.messageRepo.SoftDelete(ctx, msgID); err != nil {
		s.logger.Error().Err(err).Str("msg_id", msgID).Msg("failed to delete message")
		return apperrors.NewInternalError("failed to delete message").WithError(err)
	}

	return nil
}

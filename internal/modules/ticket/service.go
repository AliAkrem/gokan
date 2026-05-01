package ticket

import (
	"context"

	apperrors "github.com/aliakrem/gokan/internal/shared/errors"
	"github.com/rs/zerolog"
)

type Service interface {
	IssueTicket(ctx context.Context, userID string, jwt string) (string, error)
}

type TicketModuleService struct {
	ticketSvc *TicketService
	logger    *zerolog.Logger
}

func NewService(ticketSvc *TicketService, logger *zerolog.Logger) Service {
	return &TicketModuleService{
		ticketSvc: ticketSvc,
		logger:    logger,
	}
}

func (s *TicketModuleService) IssueTicket(ctx context.Context, userID string, jwt string) (string, error) {
	if userID == "" {
		return "", apperrors.NewInternalError("failed to extract user information")
	}

	if jwt == "" {
		return "", apperrors.NewInternalError("failed to extract authentication token")
	}

	ticketValue, err := s.ticketSvc.GenerateTicket(ctx, userID, jwt)
	if err != nil {
		s.logger.Error().Err(err).Str("user_id", userID).Msg("failed to generate ticket")
		return "", apperrors.NewInternalError("failed to generate ticket").WithError(err)
	}

	return ticketValue, nil
}

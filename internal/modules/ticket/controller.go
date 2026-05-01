package ticket

import (
	"net/http"

	apperrors "github.com/aliakrem/gokan/internal/shared/errors"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

type Controller struct {
	ticketService Service
}

func NewController(ticketService Service) *Controller {
	return &Controller{
		ticketService: ticketService,
	}
}

func (c *Controller) RegisterRoutes(router *gin.RouterGroup) {
	router.POST("/auth/ws-ticket", c.IssueTicket)
}

func (c *Controller) IssueTicket(ctx *gin.Context) {
	userID, exists := ctx.Get("user_id")
	if !exists {
		log.Error().Msg("user_id not found in context")
		ctx.JSON(http.StatusInternalServerError, apperrors.ErrorResponse{
			Error: "failed to extract user information",
			Code:  "INTERNAL_ERROR",
		})
		return
	}

	userIDStr, ok := userID.(string)
	if !ok {
		log.Error().Msg("user_id is not a string")
		ctx.JSON(http.StatusInternalServerError, apperrors.ErrorResponse{
			Error: "failed to extract user information",
			Code:  "INTERNAL_ERROR",
		})
		return
	}

	jwt, exists := ctx.Get("jwt")
	if !exists {
		log.Error().Msg("jwt not found in context")
		ctx.JSON(http.StatusInternalServerError, apperrors.ErrorResponse{
			Error: "failed to extract authentication token",
			Code:  "INTERNAL_ERROR",
		})
		return
	}

	jwtStr, ok := jwt.(string)
	if !ok {
		log.Error().Msg("jwt is not a string")
		ctx.JSON(http.StatusInternalServerError, apperrors.ErrorResponse{
			Error: "failed to extract authentication token",
			Code:  "INTERNAL_ERROR",
		})
		return
	}

	ticketValue, err := c.ticketService.IssueTicket(ctx.Request.Context(), userIDStr, jwtStr)
	if err != nil {
		if appErr, ok := err.(*apperrors.AppError); ok {
			apperrors.RespondWithError(ctx, appErr)
		} else {
			ctx.JSON(http.StatusInternalServerError, apperrors.ErrorResponse{
				Error: "internal server error",
				Code:  "INTERNAL_ERROR",
			})
		}
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"ticket": ticketValue,
	})
}

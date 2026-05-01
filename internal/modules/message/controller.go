package message

import (
	"net/http"
	"strconv"

	apperrors "github.com/aliakrem/gokan/internal/shared/errors"
	"github.com/gin-gonic/gin"
)

type Controller struct {
	messageService Service
}

func NewController(messageService Service) *Controller {
	return &Controller{
		messageService: messageService,
	}
}

func (c *Controller) RegisterRoutes(router *gin.RouterGroup) {
	router.GET("/rooms/:room_id/messages", c.GetRoomMessages)
	router.DELETE("/messages/:msg_id", c.DeleteMessage)
}

func (c *Controller) GetRoomMessages(ctx *gin.Context) {
	roomID := ctx.Param("room_id")
	callerID := ctx.GetString("user_id")

	limitStr := ctx.DefaultQuery("limit", "20")
	before := ctx.Query("before")
	after := ctx.Query("after")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		ctx.JSON(http.StatusBadRequest, apperrors.ErrorResponse{
			Error: "invalid limit parameter",
			Code:  "INVALID_PAYLOAD",
		})
		return
	}

	if limit > 100 {
		limit = 100
	}

	messages, err := c.messageService.GetRoomMessages(ctx.Request.Context(), roomID, callerID, limit, before, after)
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
		"messages": messages,
	})
}

func (c *Controller) DeleteMessage(ctx *gin.Context) {
	msgID := ctx.Param("msg_id")
	callerID := ctx.GetString("user_id")

	err := c.messageService.DeleteMessage(ctx.Request.Context(), msgID, callerID)
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
		"success": true,
	})
}

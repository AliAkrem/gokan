package room

import (
	"net/http"
	"strconv"

	apperrors "github.com/aliakrem/gokan/internal/shared/errors"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

type Controller struct {
	roomService Service
}

func NewController(roomService Service) *Controller {
	return &Controller{
		roomService: roomService,
	}
}

func (c *Controller) RegisterRoutes(router *gin.RouterGroup) {
	router.POST("/rooms", c.CreateRoom)
	router.GET("/rooms/:room_id", c.GetRoom)
	router.DELETE("/rooms/:room_id", c.DeleteRoom)
	router.GET("/users/:user_id/rooms", c.GetUserRooms)
}

func (c *Controller) CreateRoom(ctx *gin.Context) {
	var req CreateRoomRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		log.Warn().Err(err).Msg("invalid request body")
		ctx.JSON(http.StatusBadRequest, apperrors.ErrorResponse{
			Error: "invalid request body",
			Code:  "INVALID_PAYLOAD",
			Details: map[string]interface{}{
				"reason": err.Error(),
			},
		})
		return
	}

	callerID := ctx.GetString("user_id")

	room, err := c.roomService.CreateRoom(ctx.Request.Context(), req, callerID)
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

	ctx.JSON(http.StatusCreated, room)
}

func (c *Controller) GetRoom(ctx *gin.Context) {
	roomID := ctx.Param("room_id")
	callerID := ctx.GetString("user_id")

	room, err := c.roomService.GetRoom(ctx.Request.Context(), roomID, callerID)
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

	ctx.JSON(http.StatusOK, room)
}

func (c *Controller) GetUserRooms(ctx *gin.Context) {
	userID := ctx.Param("user_id")
	callerID := ctx.GetString("user_id")

	limitStr := ctx.DefaultQuery("limit", "20")
	cursor := ctx.Query("cursor")

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

	rooms, err := c.roomService.GetUserRooms(ctx.Request.Context(), userID, callerID, limit, cursor)
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
		"rooms": rooms,
	})
}

func (c *Controller) DeleteRoom(ctx *gin.Context) {
	roomID := ctx.Param("room_id")
	callerID := ctx.GetString("user_id")

	err := c.roomService.DeleteRoom(ctx.Request.Context(), roomID, callerID)
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

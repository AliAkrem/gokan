package user

import (
	"net/http"
	"strconv"

	apperrors "github.com/aliakrem/gokan/internal/shared/errors"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

type Controller struct {
	userService Service
}

func NewController(userService Service) *Controller {
	return &Controller{
		userService: userService,
	}
}

func (c *Controller) RegisterRoutes(router *gin.RouterGroup) {
	router.POST("/users", c.CreateUser)
	router.GET("/users/:user_id", c.GetUser)
	router.PATCH("/users/:user_id", c.UpdateUser)
	router.GET("/users/:user_id/publicKey", c.GetPublicKey)
	router.GET("/users", c.ListUsers)
}

func (c *Controller) CreateUser(ctx *gin.Context) {
	var req CreateUserRequest

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

	user, err := c.userService.CreateUser(ctx.Request.Context(), req, callerID)
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

	ctx.JSON(http.StatusCreated, user)
}

func (c *Controller) GetUser(ctx *gin.Context) {
	userID := ctx.Param("user_id")

	user, err := c.userService.GetUser(ctx.Request.Context(), userID)
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

	ctx.JSON(http.StatusOK, user)
}

func (c *Controller) UpdateUser(ctx *gin.Context) {
	userID := ctx.Param("user_id")

	var req UpdateUserRequest

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

	user, err := c.userService.UpdateUser(ctx.Request.Context(), userID, req, callerID)
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

	ctx.JSON(http.StatusOK, user)
}

func (c *Controller) GetPublicKey(ctx *gin.Context) {
	userID := ctx.Param("user_id")

	publicKey, err := c.userService.GetPublicKey(ctx.Request.Context(), userID)
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
		"publicKey": publicKey,
	})
}

func (c *Controller) ListUsers(ctx *gin.Context) {
	limitStr := ctx.DefaultQuery("limit", "50")
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

	users, err := c.userService.ListUsers(ctx.Request.Context(), limit, cursor)
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
		"users": users,
	})
}

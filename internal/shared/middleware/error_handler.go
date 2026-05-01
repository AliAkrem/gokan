package middleware

import (
	"net/http"

	apperrors "github.com/aliakrem/gokan/internal/shared/errors"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

func ErrorHandlerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				log.Error().
					Interface("error", err).
					Str("path", c.Request.URL.Path).
					Str("method", c.Request.Method).
					Msg("panic recovered in error handler middleware")

				c.JSON(http.StatusInternalServerError, apperrors.ErrorResponse{
					Error: "Internal Server Error",
					Code:  string(apperrors.ErrorCodeInternalError),
				})
				c.Abort()
			}
		}()

		c.Next()

		if len(c.Errors) > 0 {
			err := c.Errors.Last()

			if appErr, ok := err.Err.(*apperrors.AppError); ok {
				log.Error().
					Err(appErr.Err).
					Str("code", string(appErr.Code)).
					Int("status", appErr.HTTPStatus).
					Str("path", c.Request.URL.Path).
					Str("method", c.Request.Method).
					Msg("application error")

				apperrors.RespondWithError(c, appErr)
				return
			}

			log.Error().
				Err(err.Err).
				Str("path", c.Request.URL.Path).
				Str("method", c.Request.Method).
				Msg("unhandled error")

			c.JSON(http.StatusInternalServerError, apperrors.ErrorResponse{
				Error: "Internal Server Error",
				Code:  string(apperrors.ErrorCodeInternalError),
			})
		}
	}
}

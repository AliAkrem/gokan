package errors

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type ErrorCode string

const (
	ErrorCodeInvalidJWT ErrorCode = "INVALID_JWT"
	ErrorCodeForbidden  ErrorCode = "FORBIDDEN"

	ErrorCodeUserSyncFailed ErrorCode = "USER_SYNC_FAILED"
	ErrorCodeUserNotFound   ErrorCode = "USER_NOT_FOUND"

	ErrorCodeRoomNotFound ErrorCode = "ROOM_NOT_FOUND"

	ErrorCodeMessageNotFound ErrorCode = "MESSAGE_NOT_FOUND"
	ErrorCodeInvalidPayload  ErrorCode = "INVALID_PAYLOAD"

	ErrorCodeInternalError ErrorCode = "INTERNAL_ERROR"
	ErrorCodeDatabaseError ErrorCode = "DATABASE_ERROR"
)

type ErrorResponse struct {
	Error   string                 `json:"error"`
	Code    string                 `json:"code"`
	Details map[string]interface{} `json:"details,omitempty"`
}

type AppError struct {
	HTTPStatus int
	Code       ErrorCode
	Message    string
	Details    map[string]interface{}
	Err        error
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return e.Message
}

func NewAppError(httpStatus int, code ErrorCode, message string, details map[string]interface{}) *AppError {
	return &AppError{
		HTTPStatus: httpStatus,
		Code:       code,
		Message:    message,
		Details:    details,
	}
}

func (e *AppError) WithError(err error) *AppError {
	e.Err = err
	return e
}

func NewInvalidJWTError(message string) *AppError {
	return NewAppError(http.StatusUnauthorized, ErrorCodeInvalidJWT, message, nil)
}

func NewUserSyncFailedError(userID string) *AppError {
	return NewAppError(http.StatusServiceUnavailable, ErrorCodeUserSyncFailed, "User sync failed", map[string]interface{}{
		"user_id": userID,
	})
}

func NewRoomNotFoundError(roomID string) *AppError {
	return NewAppError(http.StatusNotFound, ErrorCodeRoomNotFound, "Room not found", map[string]interface{}{
		"room_id": roomID,
	})
}

func NewForbiddenError(message string) *AppError {
	return NewAppError(http.StatusForbidden, ErrorCodeForbidden, message, nil)
}

func NewInvalidPayloadError(message string, details map[string]interface{}) *AppError {
	return NewAppError(http.StatusBadRequest, ErrorCodeInvalidPayload, message, details)
}

func NewUserNotFoundError(userID string) *AppError {
	return NewAppError(http.StatusNotFound, ErrorCodeUserNotFound, "User not found", map[string]interface{}{
		"user_id": userID,
	})
}

func NewMessageNotFoundError(msgID string) *AppError {
	return NewAppError(http.StatusNotFound, ErrorCodeMessageNotFound, "Message not found", map[string]interface{}{
		"msg_id": msgID,
	})
}

func NewInternalError(message string) *AppError {
	return NewAppError(http.StatusInternalServerError, ErrorCodeInternalError, message, nil)
}

func NewDatabaseError(message string) *AppError {
	return NewAppError(http.StatusInternalServerError, ErrorCodeDatabaseError, message, nil)
}

func RespondWithError(c *gin.Context, err *AppError) {
	c.JSON(err.HTTPStatus, ErrorResponse{
		Error:   err.Message,
		Code:    string(err.Code), // Convert ErrorCode to string
		Details: err.Details,
	})
}

package api

import (
	"errors"
	"net/http"

	room "github.com/Taiga2022/final-task/internal/room/core"
	"github.com/labstack/echo/v5"
)

var ErrInvalidRequest = errors.New("invalid request")

type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

func WriteError(c *echo.Context, status int, code, message string) error {
	return c.JSON(status, ErrorResponse{Error: ErrorDetail{Code: code, Message: message}})
}
func HandleError(c *echo.Context, err error) error {
	if e, ok := errors.AsType[*echo.HTTPError](err); ok && e.Code == http.StatusUnsupportedMediaType {
		return WriteError(c, 415, "UNSUPPORTED_MEDIA_TYPE", "Content-Type must be application/json")
	}
	switch {
	case errors.Is(err, room.ErrRoomNotFound):
		return WriteError(c, 404, "ROOM_NOT_FOUND", "room was not found")
	case errors.Is(err, ErrInvalidRequest):
		return WriteError(c, 400, "INVALID_REQUEST", "request is invalid")
	default:
		c.Logger().Error("request failed", "error", err)
		return WriteError(c, 500, "INTERNAL_ERROR", "an unexpected error occurred")
	}
}

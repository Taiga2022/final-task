package http

import (
	"errors"
	"net/http"

	"github.com/Taiga2022/final-task/internal/room/core"
	"github.com/labstack/echo/v5"
)

func handleServiceError(c *echo.Context, err error) error {
	if httpErr, ok := errors.AsType[*echo.HTTPError](err); ok {
		switch httpErr.Code {
		case http.StatusUnsupportedMediaType:
			return writeError(c, http.StatusUnsupportedMediaType, "UNSUPPORTED_MEDIA_TYPE", "Content-Type must be application/json")
		case http.StatusBadRequest:
			return writeError(c, http.StatusBadRequest, "INVALID_REQUEST", "request body must be valid JSON")
		}
	}
	switch {
	case errors.Is(err, core.ErrInvalidRoomName):
		return writeError(c, http.StatusBadRequest, "INVALID_ROOM_NAME", "room_name must contain 1 to 255 characters")
	case errors.Is(err, core.ErrInvalidPassword):
		return writeError(c, http.StatusBadRequest, "INVALID_PASSWORD", "password must contain 1 to 72 bytes")
	case errors.Is(err, core.ErrRoomNameAlreadyExists):
		return writeError(c, http.StatusConflict, "ROOM_NAME_ALREADY_EXISTS", "room_name is already in use")
	case errors.Is(err, core.ErrIncorrectPassword):
		return writeError(c, http.StatusForbidden, "INCORRECT_PASSWORD", "password is incorrect")
	case errors.Is(err, core.ErrRoomNotFound):
		return writeError(c, http.StatusNotFound, "ROOM_NOT_FOUND", "room was not found")
	case errors.Is(err, core.ErrRoomFull):
		return writeError(c, http.StatusConflict, "ROOM_FULL", "room already has two members")
	default:
		return writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "an unexpected error occurred")
	}
}

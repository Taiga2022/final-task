package http

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/Taiga2022/final-task/internal/room/core"
	"github.com/Taiga2022/final-task/internal/shared/auth"
	"github.com/labstack/echo/v5"
)

type RoomService interface {
	Create(ctx context.Context, userID string, roomName core.RoomName, password core.Password) (core.JoinedRoom, error)
	Join(ctx context.Context, userID string, roomName core.RoomName, password core.Password) (core.JoinedRoom, error)
	Leave(ctx context.Context, userID, roomID string) error
}

type Handler struct {
	service RoomService
}

func NewHandler(service RoomService) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Create(c *echo.Context) error {
	userID, ok := auth.UserID(c)
	if !ok {
		return writeError(c, http.StatusUnauthorized, "UNAUTHENTICATED", "authentication is required")
	}

	request, ok := bindRoomCredentials(c)
	if !ok {
		return nil
	}
	roomName, password, err := newRoomCredentials(request)
	if err != nil {
		return handleServiceError(c, err)
	}

	room, err := h.service.Create(c.Request().Context(), userID, roomName, password)
	if err != nil {
		return handleServiceError(c, err)
	}

	return c.JSON(http.StatusCreated, newRoomJoinedResponse(room))
}

func (h *Handler) Join(c *echo.Context) error {
	userID, ok := auth.UserID(c)
	if !ok {
		return writeError(c, http.StatusUnauthorized, "UNAUTHENTICATED", "authentication is required")
	}

	request, ok := bindRoomCredentials(c)
	if !ok {
		return nil
	}
	roomName, password, err := newRoomCredentials(request)
	if err != nil {
		return handleServiceError(c, err)
	}

	room, err := h.service.Join(c.Request().Context(), userID, roomName, password)
	if err != nil {
		return handleServiceError(c, err)
	}

	return c.JSON(http.StatusOK, newRoomJoinedResponse(room))
}

func (h *Handler) Leave(c *echo.Context) error {
	userID, ok := auth.UserID(c)
	if !ok {
		return writeError(c, http.StatusUnauthorized, "UNAUTHENTICATED", "authentication is required")
	}

	roomID := strings.TrimSpace(c.Param("room_id"))
	if roomID == "" {
		return writeError(c, http.StatusBadRequest, "INVALID_REQUEST", "room_id is required")
	}

	if err := h.service.Leave(c.Request().Context(), userID, roomID); err != nil {
		return handleServiceError(c, err)
	}

	return c.NoContent(http.StatusNoContent)
}

func bindRoomCredentials(c *echo.Context) (RoomCredentials, bool) {
	var request RoomCredentials
	if err := c.Bind(&request); err != nil {
		_ = writeError(c, http.StatusBadRequest, "INVALID_REQUEST", "request body must be valid JSON")
		return RoomCredentials{}, false
	}

	return request, true
}

func newRoomCredentials(request RoomCredentials) (core.RoomName, core.Password, error) {
	roomName, err := core.NewRoomName(request.RoomName)
	if err != nil {
		return core.RoomName{}, core.Password{}, err
	}
	password, err := core.NewPassword(request.Password)
	if err != nil {
		return core.RoomName{}, core.Password{}, err
	}
	return roomName, password, nil
}

func writeError(c *echo.Context, status int, code, message string) error {
	return c.JSON(status, ErrorResponse{Error: ErrorDetail{Code: code, Message: message}})
}

func newRoomJoinedResponse(room core.JoinedRoom) RoomJoined {
	return RoomJoined{RoomID: room.RoomID, Status: room.Status}
}

func handleServiceError(c *echo.Context, err error) error {
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

package http

import (
	"context"
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

	request, err := bindRoomCredentials(c)
	if err != nil {
		return handleServiceError(c, err)
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

	request, err := bindRoomCredentials(c)
	if err != nil {
		return handleServiceError(c, err)
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

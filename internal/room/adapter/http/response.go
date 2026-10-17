package http

import (
	"github.com/Taiga2022/final-task/internal/room/core"
	"github.com/labstack/echo/v5"
)

type RoomJoined struct {
	RoomID string `json:"room_id"`
	Status string `json:"status"`
}

type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func writeError(c *echo.Context, status int, code, message string) error {
	return c.JSON(status, ErrorResponse{Error: ErrorDetail{Code: code, Message: message}})
}

func newRoomJoinedResponse(room core.JoinedRoom) RoomJoined {
	return RoomJoined{RoomID: room.RoomID, Status: room.Status}
}

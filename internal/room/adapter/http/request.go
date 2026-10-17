package http

import (
	"mime"
	"net/http"

	"github.com/Taiga2022/final-task/internal/room/core"
	"github.com/labstack/echo/v5"
)

type RoomCredentials struct {
	RoomName string `json:"room_name"`
	Password string `json:"password"`
}

func bindRoomCredentials(c *echo.Context) (RoomCredentials, error) {
	mediaType, _, err := mime.ParseMediaType(c.Request().Header.Get(echo.HeaderContentType))
	if err != nil || mediaType != echo.MIMEApplicationJSON {
		return RoomCredentials{}, echo.NewHTTPError(http.StatusUnsupportedMediaType, "unsupported media type")
	}
	var request RoomCredentials
	if err := c.Bind(&request); err != nil {
		return RoomCredentials{}, echo.NewHTTPError(http.StatusBadRequest, "invalid JSON")
	}
	return request, nil
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

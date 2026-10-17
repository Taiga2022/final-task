package http

import "github.com/labstack/echo/v5"

func RegisterRoutes(e *echo.Echo, handler *Handler) {
	rooms := e.Group("/rooms")

	rooms.POST("", handler.Create)
	rooms.POST("/join", handler.Join)
	rooms.DELETE("/:room_id/membership", handler.Leave)
}

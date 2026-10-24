package http

import "github.com/labstack/echo/v5"

func RegisterRoutes(e *echo.Echo, h *Handler) {
	e.GET("/rooms/:room_id/members/:user_id/viewing-history", h.List)
}

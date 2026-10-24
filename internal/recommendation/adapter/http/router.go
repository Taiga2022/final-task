package http

import "github.com/labstack/echo/v5"

func RegisterRoutes(e *echo.Echo, h *Handler) {
	e.PUT("/rooms/:room_id/viewing-history", h.Submit)
	e.POST("/rooms/:room_id/preference", h.Preference)
	e.GET("/rooms/:room_id/ws", h.Connect)
}

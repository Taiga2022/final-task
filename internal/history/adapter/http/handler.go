package http

import (
	"errors"
	"strconv"

	"github.com/Taiga2022/final-task/internal/history/core"
	"github.com/Taiga2022/final-task/internal/shared/api"
	"github.com/Taiga2022/final-task/internal/shared/auth"
	"github.com/labstack/echo/v5"
)

type Handler struct {
	service *core.Service
}

func NewHandler(service *core.Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) List(c *echo.Context) error {
	user, ok := auth.UserID(c)
	if !ok {
		return api.WriteError(c, 401, "UNAUTHENTICATED", "authentication is required")
	}
	limit := 20
	if values, ok := c.QueryParams()["limit"]; ok {
		if len(values) != 1 {
			return api.HandleError(c, api.ErrInvalidRequest)
		}
		parsed, err := strconv.Atoi(values[0])
		if err != nil {
			return api.HandleError(c, api.ErrInvalidRequest)
		}
		limit = parsed
	}
	room, target := api.PathParam(c, "room_id"), api.PathParam(c, "user_id")
	after, err := decodeCursor(c.QueryParam("cursor"))
	if err != nil {
		return api.HandleError(c, err)
	}
	page, err := h.service.List(c.Request().Context(), room, user, target, after, limit)
	if errors.Is(err, core.ErrInvalidPage) {
		return api.HandleError(c, api.ErrInvalidRequest)
	}
	if err != nil {
		return api.HandleError(c, err)
	}
	response := HistoryResponse{Items: make([]SeriesResponse, 0, len(page.Items)), Pagination: Pagination{HasMore: page.HasMore}}
	for _, s := range page.Items {
		response.Items = append(response.Items, SeriesResponse{s.ID, s.Title})
	}
	if page.HasMore {
		token := encodeCursor(page.LastID)
		response.Pagination.NextCursor = &token
	}
	return c.JSON(200, response)
}

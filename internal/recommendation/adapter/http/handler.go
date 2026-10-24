package http

import (
	"context"

	"github.com/Taiga2022/final-task/internal/recommendation/core"
	"github.com/Taiga2022/final-task/internal/shared/api"
	"github.com/Taiga2022/final-task/internal/shared/auth"
	"github.com/labstack/echo/v5"
)

type RecommendationService interface {
	Submit(context.Context, string, string, core.Selection) (core.Status, error)
	Preference(context.Context, string, string, core.Weight) error
	Subscribe(context.Context, string, string) (*core.Subscription, error)
}
type Handler struct{ service RecommendationService }

func NewHandler(service RecommendationService) *Handler { return &Handler{service} }

func (h *Handler) Submit(c *echo.Context) error {
	user, ok := auth.UserID(c)
	if !ok {
		return api.WriteError(c, 401, "UNAUTHENTICATED", "authentication is required")
	}
	var request SelectionRequest
	if err := api.Decode(c, &request); err != nil {
		return handleError(c, err)
	}
	selection, err := core.NewSelection(request.IncludedSeriesIDs)
	if err != nil {
		return handleError(c, err)
	}
	status, err := h.service.Submit(c.Request().Context(), api.PathParam(c, "room_id"), user, selection)
	if err != nil {
		return handleError(c, err)
	}
	code := 200
	if status == core.Processing {
		code = 202
	}
	return c.JSON(code, SubmissionResponse{status})
}
func (h *Handler) Preference(c *echo.Context) error {
	user, ok := auth.UserID(c)
	if !ok {
		return api.WriteError(c, 401, "UNAUTHENTICATED", "authentication is required")
	}
	var request PreferenceRequest
	if err := api.Decode(c, &request); err != nil {
		return handleError(c, err)
	}
	if request.Weight == nil {
		return handleError(c, api.ErrInvalidRequest)
	}
	weight, err := core.NewWeight(*request.Weight)
	if err != nil {
		return handleError(c, err)
	}
	if err := h.service.Preference(c.Request().Context(), api.PathParam(c, "room_id"), user, weight); err != nil {
		return handleError(c, err)
	}
	return c.JSON(200, PreferenceResponse{"SUCCESS", weight.Value()})
}

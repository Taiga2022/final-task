package http

import (
	"errors"

	"github.com/Taiga2022/final-task/internal/recommendation/core"
	"github.com/Taiga2022/final-task/internal/shared/api"
	"github.com/labstack/echo/v5"
)

func handleError(c *echo.Context, err error) error {
	switch {
	case errors.Is(err, core.ErrInvalidWeight):
		return api.WriteError(c, 400, "INVALID_WEIGHT", "weight must be between 0 and 1")
	case errors.Is(err, core.ErrInvalidSelectionInput):
		return api.HandleError(c, api.ErrInvalidRequest)
	case errors.Is(err, core.ErrInvalidSelection):
		return api.WriteError(c, 422, "INVALID_SERIES_SELECTION", "select series from your own viewing history")
	case errors.Is(err, core.ErrRoomNotReady):
		return api.WriteError(c, 409, "ROOM_NOT_READY", "two members are required")
	case errors.Is(err, core.ErrInProgress):
		return api.WriteError(c, 409, "RECOMMENDATION_IN_PROGRESS", "recommendation is in progress")
	default:
		return api.HandleError(c, err)
	}
}

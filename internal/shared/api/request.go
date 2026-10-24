package api

import (
	"encoding/json/v2"
	"io"
	"mime"
	"net/http"

	"github.com/labstack/echo/v5"
)

func Decode(c *echo.Context, target any) error {
	mediaType, _, err := mime.ParseMediaType(c.Request().Header.Get(echo.HeaderContentType))
	if err != nil || mediaType != echo.MIMEApplicationJSON {
		return echo.NewHTTPError(http.StatusUnsupportedMediaType, "unsupported media type")
	}
	body, err := io.ReadAll(io.LimitReader(c.Request().Body, (1<<20)+1))
	if err != nil || len(body) > 1<<20 {
		return ErrInvalidRequest
	}
	if err := json.Unmarshal(body, target); err != nil {
		return ErrInvalidRequest
	}
	return nil
}

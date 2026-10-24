package api

import (
	"net/url"

	"github.com/labstack/echo/v5"
)

// PathParam returns the decoded value of a single path segment. Echo matches
// RawPath when present, and otherwise matches the already decoded URL.Path.
func PathParam(c *echo.Context, name string) string {
	value := c.Param(name)
	if c.Request().URL.RawPath != "" {
		if decoded, err := url.PathUnescape(value); err == nil {
			return decoded
		}
	}
	return value
}

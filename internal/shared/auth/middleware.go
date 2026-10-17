package auth

import (
	"context"
	"net/http"
	"strings"

	"github.com/labstack/echo/v5"
)

const (
	HeaderUserID     = "X-User-ID"
	contextUserIDKey = "user_id"
)

// UserStore checks whether a user ID supplied by a client belongs to an
// existing user. Authentication can later be replaced without changing the
// handlers that consume the verified user ID from the request context.
type UserStore interface {
	Exists(ctx context.Context, userID string) (bool, error)
}

type Middleware struct {
	users UserStore
}

func NewMiddleware(users UserStore) *Middleware {
	return &Middleware{users: users}
}

func (m *Middleware) Authenticate(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c *echo.Context) error {
		userID := strings.TrimSpace(c.Request().Header.Get(HeaderUserID))
		if userID == "" {
			return writeError(c, http.StatusUnauthorized, "UNAUTHENTICATED", "X-User-ID is required")
		}

		exists, err := m.users.Exists(c.Request().Context(), userID)
		if err != nil {
			return writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "an unexpected error occurred")
		}
		if !exists {
			return writeError(c, http.StatusUnauthorized, "UNAUTHENTICATED", "X-User-ID is invalid")
		}

		c.Set(contextUserIDKey, userID)
		return next(c)
	}
}

func UserID(c *echo.Context) (string, bool) {
	userID, ok := c.Get(contextUserIDKey).(string)
	userID = strings.TrimSpace(userID)
	return userID, ok && userID != ""
}

type errorResponse struct {
	Error errorDetail `json:"error"`
}

type errorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func writeError(c *echo.Context, status int, code, message string) error {
	return c.JSON(status, errorResponse{Error: errorDetail{Code: code, Message: message}})
}

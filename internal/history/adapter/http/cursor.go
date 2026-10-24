package http

import (
	"encoding/base64"
	"unicode/utf8"

	"github.com/Taiga2022/final-task/internal/shared/api"
)

// A cursor is the ID of the last item in the previous page, encoded for use in
// a query parameter. It is not a security boundary.
func encodeCursor(after string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(after))
}

func decodeCursor(token string) (string, error) {
	if token == "" {
		return "", nil
	}
	if len(token) > 2048 {
		return "", api.ErrInvalidRequest
	}
	data, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return "", api.ErrInvalidRequest
	}
	if len(data) == 0 || !utf8.Valid(data) {
		return "", api.ErrInvalidRequest
	}
	return string(data), nil
}

package auth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v5"
)

type userStoreStub struct {
	exists       bool
	err          error
	receivedID   string
	handlerCalls int
}

func (s *userStoreStub) Exists(_ context.Context, userID string) (bool, error) {
	s.receivedID = userID
	return s.exists, s.err
}

func serveWithAuthentication(store *userStoreStub, headerValue string) *httptest.ResponseRecorder {
	e := echo.New()
	middleware := NewMiddleware(store)
	e.GET("/protected", func(c *echo.Context) error {
		store.handlerCalls++
		userID, ok := UserID(c)
		if !ok {
			return c.NoContent(http.StatusInternalServerError)
		}
		return c.String(http.StatusOK, userID)
	}, middleware.Authenticate)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	if headerValue != "" {
		req.Header.Set(HeaderUserID, headerValue)
	}
	recorder := httptest.NewRecorder()
	e.ServeHTTP(recorder, req)
	return recorder
}

func TestAuthenticateStoresVerifiedTrimmedUserID(t *testing.T) {
	store := &userStoreStub{exists: true}
	recorder := serveWithAuthentication(store, "  user-1  ")

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if store.receivedID != "user-1" {
		t.Fatalf("looked-up user ID = %q, want %q", store.receivedID, "user-1")
	}
	if recorder.Body.String() != "user-1" {
		t.Fatalf("body = %q, want %q", recorder.Body.String(), "user-1")
	}
}

func TestAuthenticateRejectsMissingUserID(t *testing.T) {
	store := &userStoreStub{exists: true}
	recorder := serveWithAuthentication(store, "")

	assertErrorResponse(t, recorder, http.StatusUnauthorized, "UNAUTHENTICATED")
	if store.receivedID != "" {
		t.Fatalf("user store was called with %q", store.receivedID)
	}
	if store.handlerCalls != 0 {
		t.Fatalf("handler calls = %d, want 0", store.handlerCalls)
	}
}

func TestAuthenticateRejectsUnknownUser(t *testing.T) {
	store := &userStoreStub{exists: false}
	recorder := serveWithAuthentication(store, "unknown")

	assertErrorResponse(t, recorder, http.StatusUnauthorized, "UNAUTHENTICATED")
	if store.handlerCalls != 0 {
		t.Fatalf("handler calls = %d, want 0", store.handlerCalls)
	}
}

func TestAuthenticateReturnsInternalErrorWhenUserLookupFails(t *testing.T) {
	store := &userStoreStub{err: errors.New("database unavailable")}
	recorder := serveWithAuthentication(store, "user-1")

	assertErrorResponse(t, recorder, http.StatusInternalServerError, "INTERNAL_ERROR")
	if store.handlerCalls != 0 {
		t.Fatalf("handler calls = %d, want 0", store.handlerCalls)
	}
}

func assertErrorResponse(t *testing.T, recorder *httptest.ResponseRecorder, status int, code string) {
	t.Helper()
	if recorder.Code != status {
		t.Fatalf("status = %d, want %d; body = %s", recorder.Code, status, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"code":"`+code+`"`) {
		t.Fatalf("body does not contain code %q: %s", code, recorder.Body.String())
	}
}

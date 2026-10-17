package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Taiga2022/final-task/internal/room/core"
	"github.com/Taiga2022/final-task/internal/shared/auth"
	"github.com/labstack/echo/v5"
)

type existingUserStore struct{}

func (existingUserStore) Exists(context.Context, string) (bool, error) {
	return true, nil
}

type roomServiceStub struct {
	createdUserID string
	leftUserID    string
	leftRoomID    string
	joinErr       error
}

func (s *roomServiceStub) Create(_ context.Context, userID string, _ core.RoomName, _ core.Password) (core.JoinedRoom, error) {
	s.createdUserID = userID
	return core.JoinedRoom{RoomID: "room-1", Status: "waiting"}, nil
}

func (s *roomServiceStub) Join(_ context.Context, _ string, _ core.RoomName, _ core.Password) (core.JoinedRoom, error) {
	if s.joinErr != nil {
		return core.JoinedRoom{}, s.joinErr
	}
	return core.JoinedRoom{RoomID: "room-1", Status: "ready"}, nil
}

func TestJoinMapsDomainError(t *testing.T) {
	e := newTestServer(&roomServiceStub{joinErr: core.ErrRoomFull})
	req := httptest.NewRequest(http.MethodPost, "/rooms/join", strings.NewReader(`{"room_name":"movie night","password":"secret"}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set(auth.HeaderUserID, "user-3")
	recorder := httptest.NewRecorder()

	e.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusConflict)
	}
	if !strings.Contains(recorder.Body.String(), `"code":"ROOM_FULL"`) {
		t.Fatalf("body does not contain error code: %s", recorder.Body.String())
	}
}

func (s *roomServiceStub) Leave(_ context.Context, userID, roomID string) error {
	s.leftUserID = userID
	s.leftRoomID = roomID
	return nil
}

func newTestServer(service RoomService) *echo.Echo {
	e := echo.New()
	authentication := auth.NewMiddleware(existingUserStore{})
	e.Use(authentication.Authenticate)
	RegisterRoutes(e, NewHandler(service))
	return e
}

func TestCreateRoom(t *testing.T) {
	service := &roomServiceStub{}
	e := newTestServer(service)
	req := httptest.NewRequest(http.MethodPost, "/rooms", strings.NewReader(`{"room_name":"movie night","password":"secret"}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set(auth.HeaderUserID, "user-1")
	recorder := httptest.NewRecorder()

	e.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body = %s", recorder.Code, http.StatusCreated, recorder.Body.String())
	}
	if service.createdUserID != "user-1" {
		t.Fatalf("user ID = %q, want %q", service.createdUserID, "user-1")
	}
	if got := strings.TrimSpace(recorder.Body.String()); got != `{"room_id":"room-1","status":"waiting"}` {
		t.Fatalf("body = %s", got)
	}
}

func TestCreateRoomRejectsEmptyName(t *testing.T) {
	e := newTestServer(&roomServiceStub{})
	req := httptest.NewRequest(http.MethodPost, "/rooms", strings.NewReader(`{"room_name":" ","password":"secret"}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set(auth.HeaderUserID, "user-1")
	recorder := httptest.NewRecorder()

	e.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
	if !strings.Contains(recorder.Body.String(), `"code":"INVALID_ROOM_NAME"`) {
		t.Fatalf("body does not contain error code: %s", recorder.Body.String())
	}
}

func TestCreateRoomRejectsPasswordTooLongForBcrypt(t *testing.T) {
	e := newTestServer(&roomServiceStub{})
	body := `{"room_name":"movie night","password":"` + strings.Repeat("a", 73) + `"}`
	req := httptest.NewRequest(http.MethodPost, "/rooms", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set(auth.HeaderUserID, "user-1")
	recorder := httptest.NewRecorder()

	e.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
	if !strings.Contains(recorder.Body.String(), `"code":"INVALID_PASSWORD"`) {
		t.Fatalf("body does not contain error code: %s", recorder.Body.String())
	}
}

func TestLeaveRoom(t *testing.T) {
	service := &roomServiceStub{}
	e := newTestServer(service)
	req := httptest.NewRequest(http.MethodDelete, "/rooms/room-1/membership", nil)
	req.Header.Set(auth.HeaderUserID, "user-2")
	recorder := httptest.NewRecorder()

	e.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNoContent)
	}
	if service.leftUserID != "user-2" || service.leftRoomID != "room-1" {
		t.Fatalf("Leave called with userID=%q roomID=%q", service.leftUserID, service.leftRoomID)
	}
}

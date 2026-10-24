package http

import (
	"context"
	"encoding/json/v2"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Taiga2022/final-task/internal/recommendation/core"
	roomhttp "github.com/Taiga2022/final-task/internal/room/adapter/http"
	room "github.com/Taiga2022/final-task/internal/room/core"
	"github.com/Taiga2022/final-task/internal/shared/auth"
	"github.com/coder/websocket"
	"github.com/labstack/echo/v5"
)

type usersStub struct{}

func (usersStub) Exists(context.Context, string) (bool, error) { return true, nil }

type storeStub struct {
	mu      sync.Mutex
	members []string
	owner   string
	weight  core.Weight
}

func (s *storeStub) Snapshot(context.Context, string) (core.RoomSnapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return core.RoomSnapshot{ID: "r", OwnerID: s.owner, Weight: s.weight, Members: slices.Clone(s.members)}, nil
}
func (s *storeStub) ValidateSelection(_ context.Context, user string, ids []string) (bool, error) {
	for _, id := range ids {
		if id != "series-"+user {
			return false, nil
		}
	}
	return true, nil
}
func (s *storeStub) SetWeight(_ context.Context, _ string, w core.Weight) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.weight = w
	return nil
}
func (s *storeStub) Create(context.Context, string, string, room.RoomName, room.PasswordHash) error {
	return nil
}
func (s *storeStub) FindByName(context.Context, room.RoomName) (room.Room, error) {
	return room.Room{}, room.ErrRoomNotFound
}
func (s *storeStub) Join(context.Context, string, string) (int64, error) {
	return 0, room.ErrRoomNotFound
}
func (s *storeStub) Leave(_ context.Context, _ string, user string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !slices.Contains(s.members, user) {
		return room.ErrRoomNotFound
	}
	s.members = slices.DeleteFunc(s.members, func(id string) bool { return id == user })
	if user == s.owner && len(s.members) > 0 {
		s.owner = s.members[0]
		s.weight, _ = core.NewWeight(0.5)
	}
	return nil
}

type recommenderFunc func(context.Context, core.Input) ([]core.Series, error)

func (f recommenderFunc) Recommend(ctx context.Context, in core.Input) ([]core.Series, error) {
	return f(ctx, in)
}

func newServer(t *testing.T, f recommenderFunc) (*echo.Echo, *core.Service) {
	t.Helper()
	w, _ := core.NewWeight(0.5)
	store := &storeStub{members: []string{"a", "b"}, owner: "a", weight: w}
	service := core.NewService(store, f)
	t.Cleanup(service.Close)
	e := echo.New()
	e.Use(auth.NewMiddleware(usersStub{}).Authenticate)
	RegisterRoutes(e, NewHandler(service))
	roomhttp.RegisterRoutes(e, roomhttp.NewHandler(room.NewLiveService(store, service)))
	return e, service
}
func request(e *echo.Echo, method, path, user, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if user != "" {
		req.Header.Set(auth.HeaderUserID, user)
	}
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func TestRecommendationHTTPValidation(t *testing.T) {
	e, _ := newServer(t, func(context.Context, core.Input) ([]core.Series, error) { return nil, nil })
	for _, tc := range []struct {
		method, path, user, body string
		status                   int
		code                     string
	}{
		{"PUT", "/rooms/r/viewing-history", "a", "{}", 400, "INVALID_REQUEST"},
		{"PUT", "/rooms/r/viewing-history", "a", `{"included_series_ids":null}`, 400, "INVALID_REQUEST"},
		{"PUT", "/rooms/r/viewing-history", "a", `{"included_series_ids":["s","s"]}`, 400, "INVALID_REQUEST"},
		{"PUT", "/rooms/r/viewing-history", "a", `{"included_series_ids":["s"]}`, 422, "INVALID_SERIES_SELECTION"},
		{"PUT", "/rooms/r/viewing-history", "a", `{"included_series_ids":[1]}`, 400, "INVALID_REQUEST"},
		{"PUT", "/rooms/r/viewing-history", "outsider", `{"included_series_ids":[]}`, 404, "ROOM_NOT_FOUND"},
		{"PUT", "/rooms/r/viewing-history", "", `{"included_series_ids":[]}`, 401, "UNAUTHENTICATED"},
		{"PUT", "/rooms/r/viewing-history", "a", `{"included_series_ids":[],"included_series_ids":[]}`, 400, "INVALID_REQUEST"},
		{"POST", "/rooms/r/preference", "a", `{"weight":1.2}`, 400, "INVALID_WEIGHT"},
		{"POST", "/rooms/r/preference", "a", `{"weight":null}`, 400, "INVALID_REQUEST"},
		{"POST", "/rooms/r/preference", "a", `{"weight":"0.5"}`, 400, "INVALID_REQUEST"},
		{"POST", "/rooms/r/preference", "a", "{}", 400, "INVALID_REQUEST"},
		{"POST", "/rooms/r/preference", "outsider", `{"weight":0}`, 404, "ROOM_NOT_FOUND"},
		{"GET", "/rooms/r/ws", "outsider", "", 404, "ROOM_NOT_FOUND"},
	} {
		t.Run(tc.method+tc.path+tc.user+tc.body, func(t *testing.T) {
			res := request(e, tc.method, tc.path, tc.user, tc.body)
			if res.Code != tc.status || !strings.Contains(res.Body.String(), tc.code) {
				t.Fatalf("got %d %s", res.Code, res.Body.String())
			}
		})
	}
	zero := request(e, "POST", "/rooms/r/preference", "a", `{"weight":0}`)
	if zero.Code != 200 || !strings.Contains(zero.Body.String(), `"weight":0`) {
		t.Fatalf("zero: %d %s", zero.Code, zero.Body.String())
	}
}

type wireEvent struct {
	Type string `json:"type"`
	Data struct {
		RoomID  string           `json:"room_id"`
		Members []MemberResponse `json:"members"`
		Status  core.Status      `json:"status"`
		Items   []SeriesResponse `json:"items"`
	} `json:"data"`
}

func readEvent(t *testing.T, conn *websocket.Conn) wireEvent {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
	defer cancel()
	typ, data, err := conn.Read(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if typ != websocket.MessageText {
		t.Fatalf("message type %v", typ)
	}
	var event wireEvent
	if err := json.Unmarshal(data, &event); err != nil {
		t.Fatal(err)
	}
	return event
}
func awaitWire(t *testing.T, conn *websocket.Conn, typ string, status core.Status) wireEvent {
	t.Helper()
	for {
		event := readEvent(t, conn)
		if event.Type == typ && (status == "" || event.Data.Status == status) {
			return event
		}
	}
}
func dial(t *testing.T, url, user string) *websocket.Conn {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
	defer cancel()
	conn, _, err := websocket.Dial(ctx, url, &websocket.DialOptions{HTTPHeader: http.Header{auth.HeaderUserID: []string{user}}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := conn.CloseNow(); err != nil {
			t.Log(err)
		}
	})
	return conn
}
func TestWebSocketWorkflow(t *testing.T) {
	release := make(chan struct{})
	var once sync.Once
	e, _ := newServer(t, func(ctx context.Context, _ core.Input) ([]core.Series, error) {
		select {
		case <-release:
			return []core.Series{{ID: "recommended", Title: "Recommended"}}, nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	})
	server := httptest.NewServer(e)
	t.Cleanup(server.Close)
	t.Cleanup(func() { once.Do(func() { close(release) }) })
	socketURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/rooms/r/ws"
	a, b := dial(t, socketURL, "a"), dial(t, socketURL, "b")
	if readEvent(t, a).Data.Status != core.Waiting || readEvent(t, b).Data.Status != core.Waiting {
		t.Fatal("initial state")
	}
	send := func(method, path, user, body string, want int) {
		t.Helper()
		req, err := http.NewRequestWithContext(t.Context(), method, server.URL+path, strings.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set(auth.HeaderUserID, user)
		req.Header.Set("Content-Type", "application/json")
		res, err := server.Client().Do(req)
		if err != nil {
			t.Fatal(err)
		}
		data, readErr := io.ReadAll(res.Body)
		closeErr := res.Body.Close()
		if readErr != nil || closeErr != nil {
			t.Fatalf("body errors: %v %v", readErr, closeErr)
		}
		if res.StatusCode != want {
			t.Fatalf("%s: %d %s", path, res.StatusCode, data)
		}
	}
	send("PUT", "/rooms/r/viewing-history", "a", `{"included_series_ids":["series-a"]}`, 200)
	send("PUT", "/rooms/r/viewing-history", "b", `{"included_series_ids":["series-b"]}`, 202)
	awaitWire(t, a, "room.updated", core.Processing)
	awaitWire(t, b, "room.updated", core.Processing)
	send("POST", "/rooms/r/preference", "a", `{"weight":0.8}`, 409)
	once.Do(func() { close(release) })
	for _, conn := range []*websocket.Conn{a, b} {
		awaitWire(t, conn, "room.updated", core.Completed)
		event := awaitWire(t, conn, "recommendation.completed", "")
		if len(event.Data.Items) != 1 || event.Data.Items[0].ID != "recommended" {
			t.Fatalf("%+v", event)
		}
	}
	reconnect := dial(t, socketURL, "a")
	if readEvent(t, reconnect).Data.Status != core.Completed || readEvent(t, reconnect).Type != "recommendation.completed" {
		t.Fatal("reconnect missing result")
	}
	send("PUT", "/rooms/r/viewing-history", "a", `{"included_series_ids":["series-a"]}`, 200)
	send("DELETE", "/rooms/r/membership", "a", "", 204)
	remaining := awaitWire(t, b, "room.updated", core.Waiting)
	if len(remaining.Data.Members) != 1 || remaining.Data.Members[0].UserID != "b" {
		t.Fatalf("remaining: %+v", remaining)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
	defer cancel()
	if _, _, err := a.Read(ctx); err == nil {
		t.Fatal("departed connection remains open")
	}
	send("POST", "/rooms/r/preference", "b", `{"weight":0.5}`, 409)
	send("GET", "/rooms/r/ws", "a", "", 404)
	send("DELETE", "/rooms/r/membership", "b", "", 204)
}

func Example_eventResponse() {
	event := eventResponse(core.Event{Type: "recommendation.completed", State: core.State{RoomID: "r"}})
	data, err := json.Marshal(event)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(string(data))
	// Output: {"type":"recommendation.completed","data":{"room_id":"r","items":[]}}
}

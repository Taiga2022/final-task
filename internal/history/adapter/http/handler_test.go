package http

import (
	"context"
	"encoding/json/v2"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/Taiga2022/final-task/internal/history/core"
	room "github.com/Taiga2022/final-task/internal/room/core"
	"github.com/Taiga2022/final-task/internal/shared/auth"
	"github.com/labstack/echo/v5"
)

type usersStub struct{}

func (usersStub) Exists(context.Context, string) (bool, error) { return true, nil }

type readerStub struct {
	after         string
	limit         int
	actor, target string
	err           error
}

func (r *readerStub) Read(_ context.Context, _ string, actor, target, after string, limit int) ([]core.Series, error) {
	r.after, r.limit, r.actor, r.target = after, limit, actor, target
	if r.err != nil {
		return nil, r.err
	}
	if after == "" {
		return []core.Series{{ID: "s1", Title: "One"}, {ID: "s2", Title: "Two"}}, nil
	}
	return []core.Series{}, nil
}
func historyServer(t *testing.T, r *readerStub) *echo.Echo {
	t.Helper()
	e := echo.New()
	e.Use(auth.NewMiddleware(usersStub{}).Authenticate)
	RegisterRoutes(e, NewHandler(core.NewService(r)))
	return e
}
func get(e *echo.Echo, path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.Header.Set(auth.HeaderUserID, "actor")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}
func TestHistoryPagination(t *testing.T) {
	reader := &readerStub{}
	e := historyServer(t, reader)
	path := "/rooms/r/members/target/viewing-history"
	res := get(e, path+"?limit=1")
	if res.Code != 200 {
		t.Fatalf("%d %s", res.Code, res.Body.String())
	}
	var page HistoryResponse
	if err := json.Unmarshal(res.Body.Bytes(), &page); err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 1 || !page.Pagination.HasMore || page.Pagination.NextCursor == nil {
		t.Fatalf("%+v", page)
	}
	if reader.limit != 2 || reader.actor != "actor" || reader.target != "target" {
		t.Fatalf("reader: %+v", reader)
	}
	token := url.QueryEscape(*page.Pagination.NextCursor)
	next := get(e, path+"?limit=1&cursor="+token)
	if next.Code != 200 || reader.after != "s1" {
		t.Fatalf("%d %s after=%s", next.Code, next.Body.String(), reader.after)
	}
	if err := json.Unmarshal(next.Body.Bytes(), &page); err != nil {
		t.Fatal(err)
	}
	if page.Pagination.NextCursor != nil || page.Pagination.HasMore || len(page.Items) != 0 {
		t.Fatalf("%+v", page)
	}
	for _, p := range []string{
		path + "?cursor=garbage",
		path + "?limit=0", path + "?limit=101", path + "?limit=NaN", path + "?limit=",
	} {
		if got := get(e, p); got.Code != 400 {
			t.Fatalf("%s: %d", p, got.Code)
		}
	}
}
func TestHistoryMembershipError(t *testing.T) {
	e := historyServer(t, &readerStub{err: room.ErrRoomNotFound})
	got := get(e, "/rooms/r/members/target/viewing-history")
	if got.Code != 404 {
		t.Fatalf("status %d", got.Code)
	}
}

func TestHistoryEncodedPathParameters(t *testing.T) {
	for _, target := range []string{"user==", "/user+id==", "literal%2Fvalue"} {
		t.Run(target, func(t *testing.T) {
			reader := &readerStub{}
			e := historyServer(t, reader)
			res := get(e, "/rooms/r/members/"+url.PathEscape(target)+"/viewing-history")
			if res.Code != 200 || reader.target != target {
				t.Fatalf("status=%d target=%q, want %q", res.Code, reader.target, target)
			}
		})
	}
}

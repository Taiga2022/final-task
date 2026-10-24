package http

import (
	"context"
	"encoding/json/v2"
	"sync"
	"time"

	"github.com/Taiga2022/final-task/internal/shared/api"
	"github.com/Taiga2022/final-task/internal/shared/auth"
	"github.com/coder/websocket"
	"github.com/labstack/echo/v5"
)

func (h *Handler) Connect(c *echo.Context) error {
	user, ok := auth.UserID(c)
	if !ok {
		return api.WriteError(c, 401, "UNAUTHENTICATED", "authentication is required")
	}
	sub, err := h.service.Subscribe(c.Request().Context(), api.PathParam(c, "room_id"), user)
	if err != nil {
		return handleError(c, err)
	}
	defer sub.Close()
	conn, err := websocket.Accept(c.Response(), c.Request(), nil)
	if err != nil {
		return nil
	} // Accept writes handshake errors itself.
	defer func() {
		if err := conn.CloseNow(); err != nil {
			c.Logger().Debug("close websocket", "error", err)
		}
	}()
	lifetimeCtx, cancel := context.WithCancel(c.Request().Context())
	var workers sync.WaitGroup
	workers.Go(func() {
		select {
		case <-sub.Done:
			cancel()
		case <-lifetimeCtx.Done():
		}
	})
	defer func() { cancel(); workers.Wait() }()
	ctx := conn.CloseRead(lifetimeCtx)
	heartbeat := time.NewTicker(30 * time.Second)
	defer heartbeat.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-sub.Done:
			return nil
		case <-heartbeat.C:
			pingCtx, stop := context.WithTimeout(ctx, 5*time.Second)
			err := conn.Ping(pingCtx)
			stop()
			if err != nil {
				return nil
			}
		case event, ok := <-sub.Events:
			if !ok {
				return nil
			}
			data, err := json.Marshal(eventResponse(event))
			if err != nil {
				return nil
			}
			writeCtx, stop := context.WithTimeout(ctx, 5*time.Second)
			err = conn.Write(writeCtx, websocket.MessageText, data)
			stop()
			if err != nil {
				return nil
			}
		}
	}
}

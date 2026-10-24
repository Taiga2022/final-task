package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	historydb "github.com/Taiga2022/final-task/internal/history/adapter/db"
	historyhttp "github.com/Taiga2022/final-task/internal/history/adapter/http"
	historycore "github.com/Taiga2022/final-task/internal/history/core"
	recdb "github.com/Taiga2022/final-task/internal/recommendation/adapter/db"
	rechttp "github.com/Taiga2022/final-task/internal/recommendation/adapter/http"
	reccore "github.com/Taiga2022/final-task/internal/recommendation/core"
	roomdb "github.com/Taiga2022/final-task/internal/room/adapter/db"
	roomhttp "github.com/Taiga2022/final-task/internal/room/adapter/http"
	"github.com/Taiga2022/final-task/internal/room/core"
	"github.com/Taiga2022/final-task/internal/shared/auth"
	shareddb "github.com/Taiga2022/final-task/internal/shared/db"
	"github.com/labstack/echo/v5"
)

func main() {
	database, err := shareddb.Open()
	if err != nil {
		log.Fatalf("open database: %v", err)
	}

	sqlDB, err := database.DB()
	if err != nil {
		log.Fatalf("get database connection: %v", err)
	}
	defer func() {
		if err := sqlDB.Close(); err != nil {
			log.Printf("close database connection: %v", err)
		}
	}()

	e := echo.New()
	authentication := auth.NewMiddleware(auth.NewGORMUserStore(database))
	e.Use(authentication.Authenticate)
	recommendationRepository := recdb.NewRepository(database)
	recommendations := reccore.NewService(recommendationRepository, reccore.NewGenreRecommender(recommendationRepository))
	defer recommendations.Close()
	roomService := core.NewLiveService(roomdb.NewRepository(database), recommendations)
	roomhttp.RegisterRoutes(e, roomhttp.NewHandler(roomService))
	rechttp.RegisterRoutes(e, rechttp.NewHandler(recommendations))
	historyhttp.RegisterRoutes(e, historyhttp.NewHandler(historycore.NewService(historydb.NewHistoryReader(database))))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	server := &http.Server{Addr: ":8080", Handler: e, ReadHeaderTimeout: 5 * time.Second}
	stopped := make(chan error, 1)
	go func() { stopped <- server.ListenAndServe() }()
	select {
	case err := <-stopped:
		if !errors.Is(err, http.ErrServerClosed) {
			e.Logger.Error("server stopped", "error", err)
		}
	case <-ctx.Done():
		recommendations.Close()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			e.Logger.Error("shutdown server", "error", err)
		}
	}
}

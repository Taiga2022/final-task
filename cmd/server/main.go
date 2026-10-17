package main

import (
	"log"

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
	roomService := core.NewService(roomdb.NewRepository(database))
	roomhttp.RegisterRoutes(e, roomhttp.NewHandler(roomService))

	if err := e.Start(":8080"); err != nil {
		e.Logger.Error("server stopped", "error", err)
	}
}

package main

import (
	"log"

	historydb "github.com/Taiga2022/final-task/internal/history/adapter/db"
	"github.com/Taiga2022/final-task/internal/shared/db"
)

func main() {
	database, err := db.Open()
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

	if err := run(historydb.NewHistoryWriter(database)); err != nil {
		log.Fatalf("seed database: %v", err)
	}
}

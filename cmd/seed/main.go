package main

import (
	"log"

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
	defer sqlDB.Close()

	if err := run(database); err != nil {
		log.Fatalf("seed database: %v", err)
	}
}

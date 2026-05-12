package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"v2ray-dash/backend/internal/config"
	"v2ray-dash/backend/internal/handler"
	"v2ray-dash/backend/pkg/database"
)

func main() {
	cfg := config.Load()

	db, err := database.NewPostgres(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := database.InitSchema(db); err != nil {
		log.Fatalf("Failed to init schema: %v", err)
	}

	r := gin.Default()
	handler.SetupRoutes(r, db)

	log.Println("Server starting on :8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
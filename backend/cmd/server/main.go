package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"v2ray-dash/backend/internal/config"
	"v2ray-dash/backend/internal/handler"
	"v2ray-dash/backend/pkg/database"
)

func startCleanupWorker(db *database.DB) {
	go func() {
		cleanOldData(db)
		ticker := time.NewTicker(6 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			cleanOldData(db)
		}
	}()
}

func cleanOldData(db *database.DB) {
	queries := []struct {
		sql  string
		name string
	}{
		{`DELETE FROM account_traffic_logs      WHERE recorded_at < NOW() - INTERVAL '30 days'`, "account_traffic_logs"},
		{`DELETE FROM node_status               WHERE reported_at < NOW() - INTERVAL '30 days'`, "node_status"},
		{`DELETE FROM subscription_traffic_logs WHERE recorded_at < NOW() - INTERVAL '30 days'`, "subscription_traffic_logs"},
	}
	for _, q := range queries {
		if _, err := db.Exec(q.sql); err != nil {
			log.Printf("cleanup: failed to purge %s: %v", q.name, err)
		}
	}
}

func main() {
	cfg := config.Load()
	cfg.Validate()

	db, err := database.NewPostgres(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := database.InitSchema(db); err != nil {
		log.Fatalf("Failed to init schema: %v", err)
	}

	startCleanupWorker(db)

	r := gin.Default()
	handler.SetupRoutes(r, db, cfg)

	srv := &http.Server{
		Addr:    ":" + cfg.ServerPort,
		Handler: r,
	}

	go func() {
		log.Printf("Server starting on :%s", cfg.ServerPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}
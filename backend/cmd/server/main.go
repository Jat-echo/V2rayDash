package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"v2ray-dash/backend/internal/config"
	"v2ray-dash/backend/internal/handler"
	"v2ray-dash/backend/internal/service"
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
	// An interrupted publish is never reported as successful after a controller restart.
	if _, err := db.Exec(`UPDATE managed_nodes SET state='failed',last_error='控制中心在发布期间重启，请重新核对节点配置',updated_at=CURRENT_TIMESTAMP WHERE state='publishing'`); err != nil {
		log.Fatalf("Failed to recover publish state: %v", err)
	}

	startCleanupWorker(db)
	autoCtx, stopAutoSync := context.WithCancel(context.Background())
	defer stopAutoSync()
	autoDone := make(chan struct{})
	// Disable only for isolated rehearsals; ordinary deployments sync automatically.
	if os.Getenv("NODE_AUTO_SYNC") != "false" {
		go func() { defer close(autoDone); service.RunNodeAutoSync(autoCtx, db.DB, 5*time.Second, time.Second) }()
	} else {
		close(autoDone)
	}

	r := gin.Default()
	handler.SetupRoutes(r, db, cfg)

	srv := &http.Server{
		Addr:    net.JoinHostPort(cfg.ServerHost, cfg.ServerPort),
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
	stopAutoSync()
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}
	select {
	case <-autoDone:
	case <-ctx.Done():
		log.Println("Auto sync is still finishing; interrupted node state will require verification after restart")
	}

	log.Println("Server exited")
}

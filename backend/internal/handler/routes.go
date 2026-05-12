package handler

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
	"v2ray-dash/backend/internal/config"
	"v2ray-dash/backend/internal/repository"
)

func SetupRoutes(r *gin.Engine, db *sql.DB, cfg *config.Config) {
	// CORS
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Next()
	})

	logRepo := repository.NewLogRepository(db)

	// API 路由组
	api := r.Group("/api")
	{
		// 服务器管理
		serverHandler := NewServerHandler(db)
		api.GET("/servers", serverHandler.List)
		api.POST("/servers", serverHandler.Create)
		api.GET("/servers/:id", serverHandler.Get)
		api.PUT("/servers/:id", serverHandler.Update)
		api.DELETE("/servers/:id", serverHandler.Delete)

		// 订阅管理
		subHandler := NewSubscriptionHandler(db)
		api.GET("/subscriptions", subHandler.List)
		api.POST("/subscriptions", subHandler.Create)
		api.GET("/subscriptions/:id", subHandler.Get)
		api.PUT("/subscriptions/:id", subHandler.Update)
		api.DELETE("/subscriptions/:id", subHandler.Delete)
		api.GET("/subscriptions/:id/link", subHandler.GetLink)

		// Agent 通信
		agentHandler := NewAgentHandler(logRepo)
		api.POST("/agent/heartbeat", agentHandler.Heartbeat)
		api.GET("/agent/config/:server_id", agentHandler.GetConfig)

		// 日志
		logHandler := NewLogHandler(logRepo)
		api.GET("/logs/operation", logHandler.ListOperationLogs)
	}

	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
}

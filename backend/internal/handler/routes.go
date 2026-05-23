package handler

import (
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"v2ray-dash/backend/internal/config"
	"v2ray-dash/backend/internal/repository"
	"v2ray-dash/backend/pkg/database"
)

func SetupRoutes(r *gin.Engine, db *database.DB, cfg *config.Config) {
	// 获取安装脚本路径
	// 优先级：环境变量 > 开发目录检查 > 生产目录
	installScriptPath := cfg.InstallScriptPath
	if installScriptPath == "" {
		cwd, _ := os.Getwd()
		possiblePaths := []string{
			filepath.Join(cwd, "scripts", "install-agent.sh"),
			filepath.Join(cwd, "..", "scripts", "install-agent.sh"),
			"/opt/v2ray-dash/scripts/install-agent.sh",
		}
		for _, p := range possiblePaths {
			if _, err := os.Stat(p); err == nil {
				installScriptPath = p
				break
			}
		}
		if installScriptPath == "" {
			installScriptPath = possiblePaths[0]
			log.Printf("[WARN] 安装脚本路径不存在，使用开发路径: %s", installScriptPath)
		} else {
			log.Printf("[INFO] 使用安装脚本路径: %s", installScriptPath)
		}
	} else {
		log.Printf("[INFO] 使用环境变量配置的安装脚本路径: %s", installScriptPath)
	}
	// CORS
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Next()
	})

	logRepo := repository.NewLogRepository(db.DB)
	settingRepo := repository.NewSettingRepository(db.DB)
	settingHandler := NewSettingHandler(settingRepo)

	// 公开订阅接口 (无需认证)
	subHandler := NewSubscriptionHandler(db.DB)
	r.GET("/api/subscribe/:uuid", subHandler.ServeSubscription)

	// API 路由组
	api := r.Group("/api")
	{
		// 服务器管理
		serverHandler := NewServerHandler(db.DB)
		api.GET("/servers", serverHandler.List)
		api.POST("/servers", serverHandler.Create)
		api.GET("/servers/:id", serverHandler.Get)
		api.PUT("/servers/:id", serverHandler.Update)
		api.DELETE("/servers/:id", serverHandler.Delete)

		// 账号管理
		accountHandler := NewAccountHandler(db.DB)
		accountHandler.RegisterRoutes(api)

		// 订阅管理
		api.GET("/subscriptions", subHandler.List)
		api.GET("/subscriptions/full", subHandler.ListWithAccounts)
		api.POST("/subscriptions", subHandler.Create)
		api.GET("/subscriptions/:id", subHandler.Get)
		api.PUT("/subscriptions/:id", subHandler.Update)
		api.DELETE("/subscriptions/:id", subHandler.Delete)
		api.GET("/subscriptions/:id/link", subHandler.GetLink)
		api.POST("/subscriptions/:id/accounts", subHandler.AddAccount)
		api.DELETE("/subscriptions/:id/accounts/:accountId", subHandler.RemoveAccount)

		// Agent 通信
		agentHandler := NewAgentHandler(logRepo, settingRepo)
		api.POST("/agent/heartbeat", agentHandler.Heartbeat)
		api.GET("/agent/config/:server_id", agentHandler.GetConfig)

		// 日志
		logHandler := NewLogHandler(logRepo)
		api.GET("/logs/operation", logHandler.ListOperationLogs)
		api.GET("/logs/node-status", logHandler.ListNodeStatuses)

		// Agent 安装脚本
		r.GET("/install-agent.sh", func(c *gin.Context) {
			c.Header("Content-Type", "text/plain")
			c.File(installScriptPath)
		})

		// 模板管理
		templateHandler := NewTemplateHandler(db)
		templateHandler.RegisterRoutes(api)

		// 安装管理
		installHandler := NewInstallHandler(installScriptPath, NewServerHandler(db.DB).repo, NewAccountHandler(db.DB).Repo())
		api.POST("/servers/:id/install", installHandler.StartInstall)

		// 系统设置
		api.GET("/settings/public-url", settingHandler.GetPublicURL)
		api.PUT("/settings/public-url", settingHandler.UpdatePublicURL)
		api.GET("/settings/public-ip", settingHandler.GetPublicIP)
	}

	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
}

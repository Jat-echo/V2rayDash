package handler

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

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

	// 认证路由（无需 token）
	authRepo := repository.NewAuthRepository(db.DB)
	authHandler := NewAuthHandler(authRepo, settingRepo)
	r.GET("/api/auth/status", authHandler.Status)
	r.POST("/api/auth/setup", authHandler.Setup)
	r.POST("/api/auth/login", authHandler.Login)

	// Agent 心跳（使用 PSK，不需要用户 token）
	accountRepo0 := NewAccountHandler(db.DB).Repo()
	subRepo0 := repository.NewSubscriptionRepository(db.DB)
	serverHandler0 := NewServerHandler(db.DB)
	agentHandlerPub := NewAgentHandlerFull(logRepo, settingRepo, serverHandler0.repo, accountRepo0, subRepo0)
	r.POST("/api/agent/heartbeat", agentHandlerPub.Heartbeat)
	r.GET("/api/agent/config/:server_id", agentHandlerPub.GetConfig)

	// 公开订阅接口 (无需认证)
	subHandler := NewSubscriptionHandler(db.DB)
	r.GET("/api/subscribe/:uuid", subHandler.ServeSubscription)

	// API 路由组（需要认证）
	api := r.Group("/api")
	api.Use(AuthMiddleware(settingRepo))
	{
		// 服务器管理
		serverHandler := NewServerHandler(db.DB)
		api.GET("/servers", serverHandler.List)
		api.POST("/servers", serverHandler.Create)
		api.GET("/servers/:id", serverHandler.Get)
		api.PUT("/servers/:id", serverHandler.Update)
		api.DELETE("/servers/:id", serverHandler.Delete)
		api.POST("/servers/:id/restart-xray", serverHandler.RestartXray)

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
		api.GET("/subscriptions/:id/accounts", subHandler.GetAccounts)
		api.DELETE("/subscriptions/:id/accounts/:accountId", subHandler.RemoveAccount)
		api.PUT("/subscriptions/:id/accounts/order", subHandler.UpdateAccountsOrder)

		// 订阅流量查询
		api.GET("/subscriptions/:id/traffic", agentHandlerPub.GetTrafficLogs)
		api.GET("/subscriptions/:id/accounts/traffic", subHandler.GetAccountTraffic)

		// 认证管理（需要 token）
		api.PUT("/auth/password", authHandler.ChangePassword)
		api.POST("/auth/logout", authHandler.Logout)

		// 日志
		logHandler := NewLogHandler(logRepo)
		api.GET("/logs/operation", logHandler.ListOperationLogs)
		api.GET("/logs/node-status", logHandler.ListNodeStatuses)

		// Agent 安装脚本
		r.GET("/install-agent.sh", func(c *gin.Context) {
			c.Header("Content-Type", "text/plain")
			c.File(installScriptPath)
		})

		// Agent 二进制文件下载
		// 支持格式:
		// /agents/latest/:os/:arch/agent (双段，如 linux/amd64)
		// /agents/latest/:os_arch/agent (单段，如 linux_x86_64)
		// 因为 Gin 不允许 catch-all 后还有路径，使用 *path 然后在 handler 中检查后缀
		r.GET("/agents/latest/*path", func(c *gin.Context) {
			path := c.Param("path")
			// 移除前缀的 /
			path = strings.TrimPrefix(path, "/")
			if !strings.HasSuffix(path, "agent") && !strings.HasSuffix(path, "agent/") {
				c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
				return
			}
			agentBinaryPath := "/tmp/v2ray-agent"
			if _, err := os.Stat(agentBinaryPath); err == nil {
				c.Header("Content-Type", "application/octet-stream")
				c.File(agentBinaryPath)
				return
			}
			c.JSON(http.StatusNotFound, gin.H{"error": "agent binary not found"})
		})

		// 安装管理
		installHandler := NewInstallHandler(installScriptPath, NewServerHandler(db.DB).repo, NewAccountHandler(db.DB).Repo(), settingRepo)
		api.POST("/servers/:id/install", installHandler.StartInstall)

		// 系统设置
		api.GET("/settings/public-url", settingHandler.GetPublicURL)
		api.PUT("/settings/public-url", settingHandler.UpdatePublicURL)
		api.GET("/settings/public-ip", settingHandler.GetPublicIP)
		api.GET("/settings/system-status", settingHandler.GetSystemStatus)
	}

	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
}

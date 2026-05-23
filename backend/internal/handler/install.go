package handler

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"v2ray-dash/backend/internal/model"
	"v2ray-dash/backend/internal/repository"
	"v2ray-dash/backend/internal/service"
	"v2ray-dash/backend/internal/ssh"
)

type InstallHandler struct {
	scriptPath  string
	serverRepo  *repository.ServerRepository
	accountRepo *repository.AccountRepository
}

type InstallRequest struct {
	Core       string   `json:"core"`
	UUID       string   `json:"uuid"`
	ServerName string   `json:"server_name"`
	Protocols  []string `json:"protocols"`
}

func NewInstallHandler(scriptPath string, serverRepo *repository.ServerRepository, accountRepo *repository.AccountRepository) *InstallHandler {
	return &InstallHandler{
		scriptPath:  scriptPath,
		serverRepo:  serverRepo,
		accountRepo: accountRepo,
	}
}

func (h *InstallHandler) StartInstall(c *gin.Context) {
	serverID := c.Param("id")
	if serverID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "server ID is required"})
		return
	}

	// 解析安装配置
	var req InstallRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// 如果没有body，使用默认配置
		req = InstallRequest{
			Core:      "xray-core",
			Protocols: []string{"vless_reality_vision"},
		}
	}

	// 从数据库获取服务器信息（包括敏感字段）
	server, err := h.serverRepo.GetByIDForInstall(serverID)
	if err != nil {
		log.Printf("[DEBUG] GetByIDForInstall failed: %v", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "server not found"})
		return
	}

	log.Printf("[DEBUG] Server from DB: ID=%s, Name=%s, IP=%s, SSHPort=%d, SSHUser=%s, SSHKeyType=%s",
		server.ID, server.Name, server.IP, server.SSHPort, server.SSHUser, server.SSHKeyType)
	log.Printf("[DEBUG] SSHPassword length: %d", len(server.SSHPassword))
	log.Printf("[DEBUG] SSHKey length: %d", len(server.SSHKey))

	// Create SSH auth based on stored credentials
	var auth ssh.SSHAuth
	if server.SSHKeyType == "password" {
		log.Printf("[DEBUG] Using PasswordAuth")
		auth = &ssh.PasswordAuth{Password: server.SSHPassword}
	} else {
		log.Printf("[DEBUG] Using KeyAuth")
		auth = &ssh.KeyAuth{PrivateKey: server.SSHKey}
	}

	// Set SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Transfer-Encoding", "chunked")
	c.Header("Access-Control-Allow-Origin", "*")

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "streaming not supported"})
		return
	}

	// Create install config
	installConfig := &service.InstallConfig{
		Core:       req.Core,
		UUID:       req.UUID,
		ServerName: req.ServerName,
		Protocols:  req.Protocols,
	}

	// Create installer
	installer := service.NewInstaller(serverID, server.IP, server.SSHPort, server.SSHUser, auth, h.scriptPath)

	// Execute installation with streaming output
	result := installer.InstallStreaming(flusher, installConfig)

	if !result.Success {
		fmt.Fprintf(c.Writer, "\n[ERROR] %s\n", result.Error)
		flusher.Flush()
		return
	}

	// 如果安装生成了新 UUID，同步到第一个账号
	if result.GeneratedUUID != "" && req.UUID == "" {
		accounts, err := h.accountRepo.ListByServerID(serverID)
		if err == nil && len(accounts) > 0 {
			firstAccount := accounts[0]
			h.accountRepo.Update(firstAccount.ID, &model.UpdateAccountRequest{UUID: &result.GeneratedUUID})
			fmt.Fprintf(c.Writer, "\n[OK] 已同步新UUID到账号: %s\n", result.GeneratedUUID)
			flusher.Flush()
		}
	}

	// 安装成功后，保存 Reality 配置到服务器记录
	if result.RealityConfig != nil {
		realityEnabled := true
		realityPort := result.RealityPort
		_, err := h.serverRepo.Update(serverID, &model.UpdateServerRequest{
			RealityEnabled:    &realityEnabled,
			RealityServerName: &result.RealityConfig.ServerName,
			RealityPublicKey:  &result.RealityConfig.PublicKey,
			RealityPort:       &realityPort,
		})
		if err != nil {
			fmt.Fprintf(c.Writer, "\n[WARN] 保存Reality配置失败: %v\n", err)
			flusher.Flush()
		} else {
			fmt.Fprintf(c.Writer, "\n[OK] Reality配置已保存到数据库\n")
			flusher.Flush()
		}
	}
}

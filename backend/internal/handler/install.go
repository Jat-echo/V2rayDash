package handler

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"v2ray-dash/backend/internal/repository"
	"v2ray-dash/backend/internal/service"
	"v2ray-dash/backend/internal/ssh"
)

type InstallHandler struct {
	scriptPath  string
	serverRepo  *repository.ServerRepository
}

func NewInstallHandler(scriptPath string, serverRepo *repository.ServerRepository) *InstallHandler {
	return &InstallHandler{
		scriptPath: scriptPath,
		serverRepo: serverRepo,
	}
}

func (h *InstallHandler) StartInstall(c *gin.Context) {
	serverID := c.Param("id")
	if serverID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "server ID is required"})
		return
	}

	// 从数据库获取服务器信息
	server, err := h.serverRepo.GetByID(serverID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "server not found"})
		return
	}

	// Create SSH auth based on stored credentials
	var auth ssh.SSHAuth
	if server.SSHKeyType == "password" {
		auth = &ssh.PasswordAuth{Password: server.SSHPassword}
	} else {
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

	// Create installer
	installer := service.NewInstaller(serverID, server.IP, server.SSHPort, server.SSHUser, auth, h.scriptPath)

	// Execute installation (output streams directly to HTTP)
	result := installer.Install(c.Writer)
	flusher.Flush()

	if !result.Success {
		fmt.Fprintf(c.Writer, "\n[ERROR] %s\n", result.Error)
		flusher.Flush()
	}
}
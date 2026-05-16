package handler

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"v2ray-dash/backend/internal/service"
	"v2ray-dash/backend/internal/ssh"
)

type InstallHandler struct {
	scriptPath string
}

func NewInstallHandler(scriptPath string) *InstallHandler {
	return &InstallHandler{scriptPath: scriptPath}
}

func (h *InstallHandler) StartInstall(c *gin.Context) {
	serverID := c.Param("id")

	// Parse request body for server credentials
	var req struct {
		IP          string `json:"ip"`
		SSHPort     int    `json:"ssh_port"`
		SSHUser     string `json:"ssh_user"`
		SSHKeyType  string `json:"ssh_key_type"`
		SSHKey      string `json:"ssh_key"`
		SSHPassword string `json:"ssh_password"`
		Template    string `json:"template"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Set defaults
	if req.SSHPort == 0 {
		req.SSHPort = 22
	}
	if req.SSHUser == "" {
		req.SSHUser = "root"
	}
	if req.SSHKeyType == "" {
		req.SSHKeyType = "key"
	}

	// Create SSH auth
	var auth ssh.SSHAuth
	if req.SSHKeyType == "password" {
		auth = &ssh.PasswordAuth{Password: req.SSHPassword}
	} else {
		auth = &ssh.KeyAuth{PrivateKey: req.SSHKey}
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
	installer := service.NewInstaller(serverID, req.IP, req.SSHPort, req.SSHUser, auth, h.scriptPath)

	// Execute installation (output streams directly to HTTP)
	result := installer.Install(c.Writer)
	flusher.Flush()

	if !result.Success {
		fmt.Fprintf(c.Writer, "\n[ERROR] %s\n", result.Error)
		flusher.Flush()
	}
}
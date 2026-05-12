package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"v2ray-dash/backend/internal/model"
	"v2ray-dash/backend/internal/repository"
)

type AgentHandler struct {
	logRepo *repository.LogRepository
}

func NewAgentHandler(logRepo *repository.LogRepository) *AgentHandler {
	return &AgentHandler{logRepo: logRepo}
}

func (h *AgentHandler) Heartbeat(c *gin.Context) {
	var req model.HeartbeatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 更新节点状态
	status := &model.NodeStatus{
		ServerID:      req.ServerID,
		CPUPercent:   req.CPUPercent,
		MemoryPercent: req.MemPercent,
		DiskPercent:  req.DiskPercent,
		BandwidthIn:  req.BandwidthIn,
		BandwidthOut: req.BandwidthOut,
		V2rayStatus:  req.V2rayStatus,
	}

	h.logRepo.CreateNodeStatus(status)

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *AgentHandler) GetConfig(c *gin.Context) {
	serverID := c.Param("server_id")
	// 返回该服务器的最新配置
	c.JSON(http.StatusOK, gin.H{
		"server_id":      serverID,
		"control_center": "http://your-control-center:8080",
	})
}

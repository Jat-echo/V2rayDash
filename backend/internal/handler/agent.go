package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"v2ray-dash/backend/internal/model"
	"v2ray-dash/backend/internal/repository"
)

type AgentHandler struct {
	logRepo    *repository.LogRepository
	settingRepo *repository.SettingRepository
}

func NewAgentHandler(logRepo *repository.LogRepository, settingRepo *repository.SettingRepository) *AgentHandler {
	return &AgentHandler{
		logRepo:    logRepo,
		settingRepo: settingRepo,
	}
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

	// 获取控制中心URL设置
	publicURL := "http://localhost:8080"
	if setting, err := h.settingRepo.Get("public_url"); err == nil && setting != nil {
		publicURL = setting.Value
	}

	// 返回该服务器的最新配置
	c.JSON(http.StatusOK, gin.H{
		"server_id":        serverID,
		"control_center":   publicURL,
		"report_interval":  30,
	})
}

package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"v2ray-dash/backend/internal/model"
	"v2ray-dash/backend/internal/repository"
)

type AgentHandler struct {
	logRepo     *repository.LogRepository
	settingRepo *repository.SettingRepository
	serverRepo  *repository.ServerRepository
	accountRepo *repository.AccountRepository
	subRepo     *repository.SubscriptionRepository
}

func NewAgentHandlerFull(logRepo *repository.LogRepository, settingRepo *repository.SettingRepository, serverRepo *repository.ServerRepository, accountRepo *repository.AccountRepository, subRepo *repository.SubscriptionRepository) *AgentHandler {
	return &AgentHandler{
		logRepo:     logRepo,
		settingRepo: settingRepo,
		serverRepo:  serverRepo,
		accountRepo: accountRepo,
		subRepo:     subRepo,
	}
}

func (h *AgentHandler) Heartbeat(c *gin.Context) {
	var req model.HeartbeatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 验证 ServerID 是否为已注册的服务器，防止伪造心跳
	if req.ServerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing server_id"})
		return
	}
	if _, err := h.serverRepo.GetByID(req.ServerID); err != nil {
		// 未知的 ServerID 直接忽略，不暴露错误信息
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
		return
	}

	status := &model.NodeStatus{
		ServerID:      req.ServerID,
		CPUPercent:    req.CPUPercent,
		MemoryPercent: req.MemoryPercent,
		DiskPercent:   req.DiskPercent,
		BandwidthIn:   req.BandwidthIn,
		BandwidthOut:  req.BandwidthOut,
		V2rayStatus:   req.V2rayStatus,
	}
	h.logRepo.CreateNodeStatus(status)

	if req.ServerID != "" {
		h.serverRepo.UpdateStatus(req.ServerID, "online")
	}

	// 处理用户流量统计（来自 xray stats API）
	if len(req.UserTrafficStats) > 0 && h.accountRepo != nil && h.subRepo != nil {
		affectedSubs := make(map[string]struct{})
		for _, ut := range req.UserTrafficStats {
			delta := ut.Upload + ut.Download
			if delta <= 0 {
				continue
			}
			acc, err := h.accountRepo.GetByServerIDAndEmail(req.ServerID, ut.Email)
			if err != nil {
				continue
			}
			h.accountRepo.AddTrafficUsed(acc.ID, delta)
			// 找到该账号所属的所有订阅，标记需要重新计算
			subIDs, _ := h.subRepo.GetByAccountID(acc.ID)
			for _, sid := range subIDs {
				affectedSubs[sid] = struct{}{}
			}
		}
		// 重新计算受影响订阅的 traffic_used，并记录快照
		for sid := range affectedSubs {
			h.subRepo.RecalcTrafficUsed(sid)
			// 获取最新 traffic_used 并记录快照
			if sub, err := h.subRepo.GetByID(sid); err == nil {
				h.subRepo.LogTraffic(sid, sub.TrafficUsed)
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *AgentHandler) GetTrafficLogs(c *gin.Context) {
	subID := c.Param("id")
	timeRange := c.DefaultQuery("range", "1d")
	if h.subRepo == nil {
		c.JSON(http.StatusOK, []interface{}{})
		return
	}
	points, err := h.subRepo.GetTrafficLogs(subID, timeRange)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if points == nil {
		points = []model.BandwidthPoint{}
	}
	c.JSON(http.StatusOK, points)
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

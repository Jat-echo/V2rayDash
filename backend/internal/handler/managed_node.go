package handler

import (
	"database/sql"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"os"
	"v2ray-dash/backend/internal/model"
	"v2ray-dash/backend/internal/repository"
	"v2ray-dash/backend/internal/service"
)

type ManagedNodeHandler struct {
	servers      *repository.ServerRepository
	accounts     *repository.AccountRepository
	nodes        *service.NodeService
	logs         *repository.LogRepository
	settings     *repository.SettingRepository
	combinations *repository.CombinationRepository
}

func NewManagedNodeHandler(db *sql.DB) *ManagedNodeHandler {
	s, a := repository.NewServerRepository(db), repository.NewAccountRepository(db)
	return &ManagedNodeHandler{s, a, service.NewNodeService(a, s), repository.NewLogRepository(db), repository.NewSettingRepository(db), repository.NewCombinationRepository(db)}
}

func agentBinaryPath() string {
	if path := os.Getenv("AGENT_BINARY_PATH"); path != "" {
		return path
	}
	return "/tmp/v2ray-agent"
}

func (h *ManagedNodeHandler) Combinations(c *gin.Context) {
	items, err := h.combinations.List()
	if err != nil {
		c.JSON(500, gin.H{"error": "读取链路失败"})
		return
	}
	c.JSON(200, items)
}

func (h *ManagedNodeHandler) CombinationConnection(c *gin.Context) {
	c.Header("Cache-Control", "private, no-store")
	connection, err := h.combinations.Connection(c.Param("id"))
	if err != nil {
		c.JSON(404, gin.H{"error": "未找到上游连接，请先接管落地并创建链路"})
		return
	}
	c.JSON(200, connection)
}

func (h *ManagedNodeHandler) SaveCombination(c *gin.Context) {
	var req model.CombinationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "填写链路名称，并选择中转节点和落地节点"})
		return
	}
	id, err := h.combinations.Save(c.Param("id"), &req)
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	h.logs.Create(&model.OperationLog{Operator: "admin", Action: "save_relay_combination", TargetType: "combination", TargetID: id, IP: c.ClientIP()})
	status := 200
	if c.Param("id") == "" {
		status = 201
	}
	c.JSON(status, gin.H{"id": id})
}

func (h *ManagedNodeHandler) DeleteCombination(c *gin.Context) {
	if err := h.combinations.Delete(c.Param("id")); err != nil {
		c.JSON(409, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"message": "链路已删除"})
}

func (h *ManagedNodeHandler) Adopt(c *gin.Context) {
	var req struct {
		Layout string `json:"layout"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "请选择已有服务类型"})
		return
	}
	n, err := h.nodes.Adopt(c.Param("id"), req.Layout)
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	h.logs.Create(&model.OperationLog{Operator: "admin", Action: "adopt_node", TargetType: "server", TargetID: n.ServerID, IP: c.ClientIP()})
	c.JSON(200, n)
}

func (h *ManagedNodeHandler) Publish(c *gin.Context) {
	id := c.Param("id")
	n, err := h.servers.GetNode(id)
	if err != nil {
		c.JSON(400, gin.H{"error": "请先接管已有服务"})
		return
	}
	if n.State == "publishing" {
		c.JSON(409, gin.H{"error": "节点正在发布"})
		return
	}
	started, err := h.servers.BeginPublish(id)
	if err != nil {
		c.JSON(500, gin.H{"error": "无法创建发布任务"})
		return
	}
	if !started {
		c.JSON(409, gin.H{"error": "节点正在发布"})
		return
	}
	operatorIP := c.ClientIP()
	go func() {
		checks, err := h.nodes.Publish(id)
		detail := map[string]any{"checks": checks}
		if err != nil {
			_ = h.servers.SetNodeState(id, "failed", err.Error(), "")
			detail["error"] = err.Error()
		}
		h.logs.Create(&model.OperationLog{Operator: "admin", Action: "publish_node", TargetType: "server", TargetID: id, Detail: detail, IP: operatorIP})
	}()
	c.JSON(http.StatusAccepted, gin.H{"message": "正在校验、发布并验证配置", "state": "publishing"})
}

func (h *ManagedNodeHandler) Probe(c *gin.Context) {
	checks, err := h.nodes.Probe(c.Param("id"))
	if err != nil {
		c.JSON(400, gin.H{"error": fmt.Sprint(err)})
		return
	}
	c.JSON(200, gin.H{"checks": checks, "viewpoint": "node"})
}

func (h *ManagedNodeHandler) Monitor(c *gin.Context) {
	setting, err := h.settings.Get("public_url")
	if err != nil || setting == nil {
		c.JSON(400, gin.H{"error": "请先设置控制中心公网地址"})
		return
	}
	if err = h.nodes.InstallMonitor(c.Param("id"), setting.Value, agentBinaryPath()); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"message": "监控 Agent 已启动，等待首次心跳"})
}

func (h *ManagedNodeHandler) Restart(c *gin.Context) {
	if err := h.nodes.Restart(c.Param("id")); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"message": "节点服务已重启"})
}

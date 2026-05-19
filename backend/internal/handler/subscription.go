package handler

import (
	"database/sql"
	"encoding/base64"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"v2ray-dash/backend/internal/model"
	"v2ray-dash/backend/internal/repository"
	"v2ray-dash/backend/internal/service"
)

type SubscriptionHandler struct {
	repo    *repository.SubscriptionRepository
	logRepo *repository.LogRepository
	accountRepo *repository.AccountRepository
	serverRepo *repository.ServerRepository
	accountSvc *service.AccountService
}

func NewSubscriptionHandler(db *sql.DB) *SubscriptionHandler {
	return &SubscriptionHandler{
		repo:         repository.NewSubscriptionRepository(db),
		logRepo:      repository.NewLogRepository(db),
		accountRepo:  repository.NewAccountRepository(db),
		serverRepo:   repository.NewServerRepository(db),
		accountSvc:   service.NewAccountService(
			repository.NewAccountRepository(db),
			repository.NewServerRepository(db),
		),
	}
}

func (h *SubscriptionHandler) List(c *gin.Context) {
	serverID := c.Query("server_id")
	var subs []*model.Subscription
	var err error

	if serverID != "" {
		subs, err = h.repo.ListByServerID(serverID)
	} else {
		subs, err = h.repo.List()
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, subs)
}

func (h *SubscriptionHandler) Get(c *gin.Context) {
	id := c.Param("id")
	sub, err := h.repo.GetByID(id)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "subscription not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, sub)
}

func (h *SubscriptionHandler) Create(c *gin.Context) {
	var req model.CreateSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sub, err := h.repo.Create(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	h.logRepo.Create(&model.OperationLog{
		Operator:   "admin",
		Action:     "create_subscription",
		TargetType: "subscription",
		TargetID:   sub.ID,
		Detail:     map[string]any{"name": sub.Name, "server_id": sub.ServerID},
		IP:         c.ClientIP(),
	})

	c.JSON(http.StatusCreated, sub)
}

func (h *SubscriptionHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var req model.UpdateSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.repo.Update(id, &req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "updated"})
}

func (h *SubscriptionHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if err := h.repo.Delete(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	h.logRepo.Create(&model.OperationLog{
		Operator:   "admin",
		Action:     "delete_subscription",
		TargetType: "subscription",
		TargetID:   id,
		IP:         c.ClientIP(),
	})

	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

// GetLink 生成订阅链接
func (h *SubscriptionHandler) GetLink(c *gin.Context) {
	id := c.Param("id")
	sub, err := h.repo.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}

	// 使用实际域名生成订阅链接
	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	baseURL := fmt.Sprintf("%s://%s", scheme, c.Request.Host)
	link := fmt.Sprintf("%s/api/subscribe/%s", baseURL, sub.UUID)
	encoded := base64.StdEncoding.EncodeToString([]byte(link))

	c.JSON(http.StatusOK, gin.H{
		"link":    link,
		"encoded": encoded,
	})
}

// ServeSubscription 处理订阅请求 - 通过 UUID 提供订阅内容
func (h *SubscriptionHandler) ServeSubscription(c *gin.Context) {
	uuid := c.Param("uuid")
	sub, err := h.repo.GetByUUID(uuid)
	if err != nil {
		if err == sql.ErrNoRows {
			c.String(http.StatusNotFound, "Subscription not found")
			return
		}
		c.String(http.StatusInternalServerError, "Internal server error")
		return
	}

	// 检查订阅是否启用
	if !sub.Enable {
		c.String(http.StatusForbidden, "Subscription is disabled")
		return
	}

	// 获取关联的服务器
	server, err := h.serverRepo.GetByID(sub.ServerID)
	if err != nil {
		c.String(http.StatusInternalServerError, "Server not found")
		return
	}

	// 获取该服务器下的所有启用账号
	accounts, err := h.accountRepo.ListByServerID(sub.ServerID)
	if err != nil || len(accounts) == 0 {
		c.String(http.StatusNotFound, "No accounts available")
		return
	}

	// 获取订阅格式类型 (默认 vless)
	subType := c.Query("format")
	if subType == "" {
		subType = "vless"
	}

	var content string
	switch subType {
	case "clash_meta":
		content, _ = h.accountSvc.GenerateClashMetaSubscription(accounts, server.IP)
		c.Header("Content-Type", "text/plain; charset=utf-8")
	case "singbox":
		content, _ = h.accountSvc.GenerateSingBoxSubscription(accounts, server.IP)
		c.Header("Content-Type", "application/json; charset=utf-8")
	default:
		content = h.accountSvc.GenerateVLESSSubscription(accounts, server.IP)
		c.Header("Content-Type", "text/plain; charset=utf-8")
	}

	// 设置订阅缓存头 (1小时)
	c.Header("Cache-Control", "public, max-age=3600")
	c.Header("Subscription-Userinfo", fmt.Sprintf("upload=0; download=%d; total=%d; left=%d",
		sub.TrafficUsed, sub.TrafficLimit, sub.TrafficLimit-sub.TrafficUsed))

	c.String(http.StatusOK, content)
}

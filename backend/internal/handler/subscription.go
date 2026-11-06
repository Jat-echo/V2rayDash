package handler

import (
	"database/sql"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"v2ray-dash/backend/internal/model"
	"v2ray-dash/backend/internal/repository"
	"v2ray-dash/backend/internal/service"
)

type SubscriptionHandler struct {
	repo        *repository.SubscriptionRepository
	subAccRepo  *repository.SubscriptionAccountRepository
	logRepo     *repository.LogRepository
	accountRepo *repository.AccountRepository
	serverRepo  *repository.ServerRepository
	settingRepo *repository.SettingRepository
	accountSvc  *service.AccountService
}

func NewSubscriptionHandler(db *sql.DB) *SubscriptionHandler {
	accountRepo := repository.NewAccountRepository(db)
	serverRepo := repository.NewServerRepository(db)
	h := &SubscriptionHandler{
		repo:        repository.NewSubscriptionRepository(db),
		subAccRepo:  repository.NewSubscriptionAccountRepository(db),
		logRepo:     repository.NewLogRepository(db),
		accountRepo: accountRepo,
		serverRepo:  serverRepo,
		settingRepo: repository.NewSettingRepository(db),
		accountSvc:  service.NewAccountService(accountRepo, serverRepo),
	}
	return h
}

func (h *SubscriptionHandler) getPublicURL() string {
	if setting, err := h.settingRepo.Get("public_url"); err == nil && setting != nil && setting.Value != "" {
		return setting.Value
	}
	return "http://localhost:8080"
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

func (h *SubscriptionHandler) ListWithAccounts(c *gin.Context) {
	subs, err := h.repo.GetSubscriptionsWithAccounts()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, subs)
}

func (h *SubscriptionHandler) Get(c *gin.Context) {
	id := c.Param("id")
	sub, err := h.repo.GetByIDWithAccounts(id)
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
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	id, servers, err := h.repo.CreateWithTargets(&req)
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	h.syncTargetServers(servers)
	h.logRepo.Create(&model.OperationLog{Operator: "admin", Action: "create_subscription", TargetType: "subscription", TargetID: id, Detail: map[string]any{"name": req.Name, "node_count": len(req.AccountMappings)}, IP: c.ClientIP()})
	result, err := h.repo.GetByIDWithAccounts(id)
	if err != nil {
		c.JSON(500, gin.H{"error": "读取新订阅失败"})
		return
	}
	c.JSON(201, result)
}

func (h *SubscriptionHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var req model.UpdateSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.AccountMappings != nil {
		servers, err := h.repo.ReplaceTargets(id, *req.AccountMappings)
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		h.syncTargetServers(servers)
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

func (h *SubscriptionHandler) GetLink(c *gin.Context) {
	id := c.Param("id")
	sub, err := h.repo.GetByIDWithAccounts(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}

	publicURL := h.getPublicURL()
	link := fmt.Sprintf("%s/api/subscribe/%s", publicURL, sub.UUID)
	encoded := base64.StdEncoding.EncodeToString([]byte(link))

	// 生成每个账号的订阅 URI (用于二维码)
	accountLinks := make([]string, 0)
	for _, acc := range sub.Accounts {
		accountLinks = append(accountLinks, fmt.Sprintf("%s/api/subscribe/%s?aid=%s", publicURL, sub.UUID, acc.ID))
	}

	c.JSON(http.StatusOK, gin.H{
		"link":          link,
		"encoded":       encoded,
		"account_links": accountLinks,
	})
}

func (h *SubscriptionHandler) AddAccount(c *gin.Context) {
	var req model.AccountMapping
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "节点参数无效"})
		return
	}
	added, servers, err := h.repo.AddTarget(c.Param("id"), req)
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	h.syncTargetServers(servers)
	c.JSON(200, gin.H{"message": "节点已添加", "added": added})
}

func (h *SubscriptionHandler) RemoveAccount(c *gin.Context) {
	id := c.Param("id")
	accountID := c.Param("accountId")

	if err := h.subAccRepo.RemoveAccount(id, accountID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "account removed"})
}

func (h *SubscriptionHandler) GetAccounts(c *gin.Context) {
	id := c.Param("id")

	accounts, err := h.subAccRepo.GetBySubscriptionOrdered(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, accounts)
}

func (h *SubscriptionHandler) UpdateAccountsOrder(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		Order []struct {
			ID        string `json:"id"`
			SortOrder int    `json:"sort_order"`
		} `json:"order"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	for _, item := range req.Order {
		err := h.subAccRepo.UpdateSortOrder(id, item.ID, item.SortOrder)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "order updated"})
}

func (h *SubscriptionHandler) GetAccountTraffic(c *gin.Context) {
	id := c.Param("id")
	timeRange := c.DefaultQuery("range", "1d")
	series, err := h.repo.GetAccountTrafficLogs(id, timeRange)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, series)
}

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

	if !sub.Enable {
		c.String(http.StatusForbidden, "Subscription is disabled")
		return
	}

	accounts, err := h.subAccRepo.ListBySubscriptionID(sub.ID)
	if err != nil || len(accounts) == 0 {
		c.String(http.StatusNotFound, "No accounts available")
		return
	}

	// 如果指定了账号 ID，只返回该账号
	if aid := c.Query("aid"); aid != "" {
		for _, acc := range accounts {
			if acc.ID == aid {
				accounts = []*model.Account{acc}
				break
			}
		}
		// 如果没找到匹配的账号，返回错误
		if len(accounts) == 0 || accounts[0].ID != aid {
			c.String(http.StatusNotFound, "Account not found in this subscription")
			return
		}
	}
	accounts, err = h.accountSvc.ReadyAccounts(accounts)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to load published accounts")
		return
	}
	if len(accounts) == 0 {
		c.String(http.StatusServiceUnavailable, "No published enabled accounts; publish the node configuration first")
		return
	}

	serverIDs := make([]string, 0)
	serverMap := make(map[string]*model.Server)
	for _, acc := range accounts {
		if _, exists := serverMap[acc.ServerID]; !exists {
			serverIDs = append(serverIDs, acc.ServerID)
		}
	}

	realityConfigs := make(map[string]*service.RealityConfig)
	for _, serverID := range serverIDs {
		server, err := h.serverRepo.GetByID(serverID)
		if err != nil {
			continue
		}
		serverMap[serverID] = server
		realityConfigs[serverID] = &service.RealityConfig{
			Enabled:           server.RealityEnabled,
			ServerName:        server.RealityServerName,
			PublicKey:         server.RealityPublicKey,
			ShortID:           service.RealityShortID(server),
			ShortIDConfigured: server.Node != nil,
			SupportMLKEM:      server.Node != nil && server.Node.Layout != "legacy",
			Port:              server.RealityPort,
		}
	}

	subType := c.Query("format")
	if subType == "" {
		userAgent := strings.ToLower(c.GetHeader("User-Agent"))
		switch {
		case strings.Contains(userAgent, "shadowrocket"):
			subType = "shadowrocket"
		case strings.Contains(userAgent, "clash"), strings.Contains(userAgent, "mihomo"):
			subType = "clash_meta"
		default:
			subType = "vless"
		}
	}
	c.Header("Vary", "User-Agent")

	var content string
	switch subType {
	case "shadowrocket":
		content, err = h.accountSvc.GenerateShadowrocketSubscriptionMulti(accounts, serverMap, realityConfigs)
		c.Header("Content-Type", "text/plain; charset=utf-8")
	case "clash_meta":
		content, _ = h.accountSvc.GenerateClashMetaSubscriptionMulti(accounts, serverMap, realityConfigs)
		c.Header("Content-Type", "text/plain; charset=utf-8")
	case "singbox":
		content, _ = h.accountSvc.GenerateSingBoxSubscriptionMulti(accounts, serverMap, realityConfigs)
		c.Header("Content-Type", "application/json; charset=utf-8")
	case "ss":
		c.String(http.StatusBadRequest, "ShadowSocks format is not supported for VLESS accounts")
		return
	default:
		content = h.accountSvc.GenerateVLESSSubscriptionMulti(accounts, serverMap, realityConfigs)
		c.Header("Content-Type", "text/plain; charset=utf-8")
	}
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to generate subscription")
		return
	}

	c.Header("Cache-Control", "private, no-store")
	c.Header("Subscription-Userinfo", fmt.Sprintf("upload=0; download=%d; total=%d; left=%d",
		sub.TrafficUsed, sub.TrafficLimit, sub.TrafficLimit-sub.TrafficUsed))

	c.String(http.StatusOK, content)
}

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
	repo          *repository.SubscriptionRepository
	subAccRepo    *repository.SubscriptionAccountRepository
	logRepo       *repository.LogRepository
	accountRepo   *repository.AccountRepository
	serverRepo    *repository.ServerRepository
	settingRepo   *repository.SettingRepository
	accountSvc    *service.AccountService
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
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sub, err := h.repo.Create(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	accountIDs := make([]string, 0)
	for _, mapping := range req.AccountMappings {
		var accountID string
		if mapping.AutoCreate {
			newAccount, err := h.accountRepo.Create(&model.CreateAccountRequest{
				ServerID:  mapping.ServerID,
				Email:    fmt.Sprintf("auto-%s", sub.ID[:8]),
				Protocols: []string{"vless_tcp"},
			})
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("auto create account failed: %v", err)})
				return
			}
			accountID = newAccount.ID
		} else {
			accountID = mapping.AccountID
		}

		if err := h.subAccRepo.AddAccount(sub.ID, accountID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("add account to subscription failed: %v", err)})
			return
		}
		accountIDs = append(accountIDs, accountID)
	}

	h.logRepo.Create(&model.OperationLog{
		Operator:   "admin",
		Action:     "create_subscription",
		TargetType: "subscription",
		TargetID:   sub.ID,
		Detail:     map[string]any{"name": sub.Name, "account_count": len(accountIDs)},
		IP:         c.ClientIP(),
	})

	result, _ := h.repo.GetByIDWithAccounts(sub.ID)
	c.JSON(http.StatusCreated, result)
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

	if req.AccountMappings != nil {
		accountIDs := make([]string, 0)
		sortOrders := make(map[string]int)

		for i, mapping := range *req.AccountMappings {
			var accountID string
			if mapping.AutoCreate {
				newAccount, err := h.accountRepo.Create(&model.CreateAccountRequest{
					ServerID:  mapping.ServerID,
					Email:    fmt.Sprintf("auto-%s", id[:8]),
					Protocols: []string{"vless_tcp"},
				})
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("auto create account failed: %v", err)})
					return
				}
				accountID = newAccount.ID
			} else {
				accountID = mapping.AccountID
			}
			accountIDs = append(accountIDs, accountID)
			sortOrders[accountID] = i
		}

		if len(accountIDs) > 0 {
			h.subAccRepo.ReplaceAccounts(id, accountIDs, sortOrders)
		}
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
		"link":         link,
		"encoded":      encoded,
		"account_links": accountLinks,
	})
}

func (h *SubscriptionHandler) AddAccount(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		ServerID  string `json:"server_id"`
		AccountID string `json:"account_id"`
		AutoCreate bool  `json:"auto_create"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.ServerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "server_id is required"})
		return
	}
	if !req.AutoCreate && req.AccountID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "account_id is required when auto_create is false"})
		return
	}

	var accountID string
	if req.AutoCreate {
		newAccount, err := h.accountRepo.Create(&model.CreateAccountRequest{
			ServerID:  req.ServerID,
			Email:    fmt.Sprintf("auto-%s", id[:8]),
			Protocols: []string{"vless_tcp"},
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("auto create account failed: %v", err)})
			return
		}
		accountID = newAccount.ID
	} else {
		accountID = req.AccountID
	}

	if err := h.subAccRepo.AddAccount(id, accountID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "account added"})
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
			Enabled:    server.RealityEnabled,
			ServerName: server.RealityServerName,
			PublicKey:  server.RealityPublicKey,
			Port:       server.RealityPort,
		}
	}

	subType := c.Query("format")
	if subType == "" {
		userAgent := c.GetHeader("User-Agent")
		if strings.Contains(strings.ToLower(userAgent), "clash") {
			subType = "clash_meta"
		} else {
			subType = "vless"
		}
	}

	var content string
	switch subType {
	case "clash_meta":
		content, _ = h.accountSvc.GenerateClashMetaSubscriptionMulti(accounts, serverMap, realityConfigs)
		c.Header("Content-Type", "text/plain; charset=utf-8")
	case "singbox":
		content, _ = h.accountSvc.GenerateSingBoxSubscriptionMulti(accounts, serverMap, realityConfigs)
		c.Header("Content-Type", "application/json; charset=utf-8")
	default:
		content = h.accountSvc.GenerateVLESSSubscriptionMulti(accounts, serverMap, realityConfigs)
		c.Header("Content-Type", "text/plain; charset=utf-8")
	}

	c.Header("Cache-Control", "public, max-age=3600")
	c.Header("Subscription-Userinfo", fmt.Sprintf("upload=0; download=%d; total=%d; left=%d",
		sub.TrafficUsed, sub.TrafficLimit, sub.TrafficLimit-sub.TrafficUsed))

	c.String(http.StatusOK, content)
}
package handler

import (
	"database/sql"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"v2ray-dash/backend/internal/model"
	"v2ray-dash/backend/internal/repository"
	"v2ray-dash/backend/internal/service"
	"v2ray-dash/backend/internal/ssh"
)

type AccountHandler struct {
	accountRepo *repository.AccountRepository
	serverRepo  *repository.ServerRepository
	accountSvc  *service.AccountService
}

func NewAccountHandler(db *sql.DB) *AccountHandler {
	accountRepo := repository.NewAccountRepository(db)
	serverRepo := repository.NewServerRepository(db)
	accountSvc := service.NewAccountService(accountRepo, serverRepo)
	return &AccountHandler{
		accountRepo: accountRepo,
		serverRepo:  serverRepo,
		accountSvc:  accountSvc,
	}
}

func (h *AccountHandler) Repo() *repository.AccountRepository {
	return h.accountRepo
}

func (h *AccountHandler) RegisterRoutes(r *gin.RouterGroup) {
	accounts := r.Group("/servers/:id/accounts")
	{
		accounts.GET("", h.List)
		accounts.POST("", h.Create)
		accounts.POST("/import", h.Import)
		accounts.POST("/sync", h.SyncAll)
	}

	accountRoutes := r.Group("/accounts")
	{
		accountRoutes.GET("/:id", h.Get)
		accountRoutes.PUT("/:id", h.Update)
		accountRoutes.DELETE("/:id", h.Delete)
		accountRoutes.GET("/:id/subscribe", h.Subscribe)
		accountRoutes.POST("/:id/sync", h.Sync)
	}
}

func (h *AccountHandler) List(c *gin.Context) {
	serverID := c.Param("id")
	accounts, err := h.accountRepo.ListByServerID(serverID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.JSON(http.StatusOK, customerAccounts(accounts))
}

func customerAccounts(accounts []*model.Account) []*model.Account {
	customers := make([]*model.Account, 0, len(accounts))
	for _, account := range accounts {
		if !account.Transport {
			customers = append(customers, account)
		}
	}
	return customers
}

func (h *AccountHandler) Get(c *gin.Context) {
	id := c.Param("id")
	account, err := h.accountRepo.GetByID(id)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "account not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.JSON(http.StatusOK, account)
}

func (h *AccountHandler) Create(c *gin.Context) {
	serverID := c.Param("id")
	var req model.CreateAccountRequest
	req.ServerID = serverID
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.ServerID = serverID
	owner, err := h.serverRepo.GetByID(serverID)
	if err != nil {
		c.JSON(404, gin.H{"error": "server not found"})
		return
	}
	if owner.Role == "relay" {
		c.JSON(400, gin.H{"error": "请在订阅中选择代理链路，平台会自动创建对应账号"})
		return
	}
	if owner.Node != nil && owner.Node.State == "publishing" {
		c.JSON(409, gin.H{"error": "节点正在发布，请稍后创建账号"})
		return
	}

	account, err := h.accountRepo.Create(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	// Sync to remote server (GetByIDForInstall includes ssh_key/ssh_password)
	server, err := h.serverRepo.GetByIDForInstall(serverID)
	if err == nil {
		var auth ssh.SSHAuth
		if server.SSHKeyType == "password" {
			auth = &ssh.PasswordAuth{Password: server.SSHPassword}
		} else {
			auth = &ssh.KeyAuth{PrivateKey: server.SSHKey}
		}
		h.accountSvc.RequestSync(serverID, auth)
	}

	c.JSON(http.StatusCreated, account)
}

func (h *AccountHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var req model.UpdateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	account, err := h.accountRepo.GetByID(id)
	if err != nil {
		c.JSON(404, gin.H{"error": "account not found"})
		return
	}
	if account.Transport {
		c.JSON(409, gin.H{"error": "上游账号由代理链路自动管理，不能单独修改"})
		return
	}
	server, err := h.serverRepo.GetByID(account.ServerID)
	if err != nil {
		c.JSON(500, gin.H{"error": "server not found"})
		return
	}
	if server.Node != nil && req.UUID != nil && *req.UUID != account.UUID {
		c.JSON(400, gin.H{"error": "已接管账号的 UUID 不能直接修改，请创建新账号"})
		return
	}
	if server.Node != nil && server.Node.State == "publishing" {
		c.JSON(409, gin.H{"error": "节点正在发布，请稍后修改账号"})
		return
	}

	if err := h.accountRepo.Update(id, &req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "updated"})
}

func (h *AccountHandler) Delete(c *gin.Context) {
	id := c.Param("id")

	// Get account info before deletion to know which server it belongs to
	account, err := h.accountRepo.GetByID(id)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "account not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	// Delete from local DB
	if used, err := h.serverRepo.AccountReferenced(id); err != nil {
		c.JSON(500, gin.H{"error": "无法检查账号引用"})
		return
	} else if used {
		c.JSON(409, gin.H{"error": "该账号仍被中转线路使用，请先切换落地并发布后再删除"})
		return
	}
	if server, err := h.serverRepo.GetByID(account.ServerID); err == nil && server.Node != nil && server.Node.State == "publishing" {
		c.JSON(409, gin.H{"error": "节点正在发布，请稍后删除账号"})
		return
	}
	if err := h.accountRepo.Delete(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	// Sync remaining accounts to remote server (GetByIDForInstall includes ssh_key/ssh_password)
	server, err := h.serverRepo.GetByIDForInstall(account.ServerID)
	if err == nil {
		var auth ssh.SSHAuth
		if server.SSHKeyType == "password" {
			auth = &ssh.PasswordAuth{Password: server.SSHPassword}
		} else {
			auth = &ssh.KeyAuth{PrivateKey: server.SSHKey}
		}
		h.accountSvc.RequestSync(account.ServerID, auth)
	}

	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

func (h *AccountHandler) Subscribe(c *gin.Context) {
	id := c.Param("id")
	subType := c.Query("type")
	if subType == "" {
		subType = "vless"
	}

	account, err := h.accountRepo.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "account not found"})
		return
	}
	ready, err := h.accountSvc.ReadyAccounts([]*model.Account{account})
	if err != nil {
		c.JSON(500, gin.H{"error": "无法读取发布状态"})
		return
	}
	if len(ready) == 0 {
		c.JSON(409, gin.H{"error": "账号已停用或尚未发布，请先发布节点配置"})
		return
	}

	// 获取服务器完整信息（包括 Reality 配置）
	server, err := h.serverRepo.GetByIDForInstall(account.ServerID)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "server not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	// 构造 Reality 配置
	var realityConfig *service.RealityConfig
	if server.RealityEnabled && server.RealityPublicKey != "" {
		realityConfig = &service.RealityConfig{
			Enabled:           true,
			ServerName:        server.RealityServerName,
			PublicKey:         server.RealityPublicKey,
			ShortID:           service.RealityShortID(server),
			ShortIDConfigured: server.Node != nil,
			SupportMLKEM:      server.Node != nil && server.Node.Layout != "legacy",
			Port:              server.RealityPort,
		}
	}

	var content string
	var subErr error
	servers := map[string]*model.Server{server.ID: server}
	realityConfigs := map[string]*service.RealityConfig{server.ID: realityConfig}
	switch subType {
	case "clash_meta":
		content, subErr = h.accountSvc.GenerateClashMetaSubscriptionMulti(ready, servers, realityConfigs)
	case "singbox":
		content, subErr = h.accountSvc.GenerateSingBoxSubscriptionMulti(ready, servers, realityConfigs)
	default:
		content = h.accountSvc.GenerateVLESSSubscriptionMulti(ready, servers, realityConfigs)
	}
	if subErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.Header("Content-Type", "text/plain; charset=utf-8")
	c.String(http.StatusOK, content)
}

func (h *AccountHandler) Import(c *gin.Context) {
	serverID := c.Param("id")

	// 获取服务器信息以便创建 SSH 连接（需要包含敏感字段）
	server, err := h.serverRepo.GetByIDForInstall(serverID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server not found"})
		return
	}
	if server.Node != nil {
		_, err := service.NewNodeService(h.accountRepo, h.serverRepo).Adopt(serverID, server.Node.Layout)
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		accounts, err := h.accountRepo.ListByServerID(serverID)
		if err != nil {
			c.JSON(500, gin.H{"error": "读取账号失败"})
			return
		}
		c.JSON(200, gin.H{"message": "imported", "accounts": customerAccounts(accounts)})
		return
	}

	// 根据认证类型创建 auth
	var auth ssh.SSHAuth
	if server.SSHKeyType == "password" {
		auth = &ssh.PasswordAuth{Password: server.SSHPassword}
	} else {
		auth = &ssh.KeyAuth{PrivateKey: server.SSHKey}
	}

	accounts, err := h.accountSvc.ImportFromRemote(serverID, auth)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to import accounts: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "imported",
		"accounts": customerAccounts(accounts),
	})
}
func (h *AccountHandler) Sync(c *gin.Context) {
	id := c.Param("id")

	account, err := h.accountRepo.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "account not found"})
		return
	}

	server, err := h.serverRepo.GetByIDForInstall(account.ServerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server not found"})
		return
	}

	var auth ssh.SSHAuth
	if server.SSHKeyType == "password" {
		auth = &ssh.PasswordAuth{Password: server.SSHPassword}
	} else {
		auth = &ssh.KeyAuth{PrivateKey: server.SSHKey}
	}

	if err := h.accountSvc.SyncAllToRemote(account.ServerID, auth); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "sync failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "synced"})
}

func (h *AccountHandler) SyncAll(c *gin.Context) {
	serverID := c.Param("id")

	server, err := h.serverRepo.GetByIDForInstall(serverID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server not found"})
		return
	}

	var auth ssh.SSHAuth
	if server.SSHKeyType == "password" {
		auth = &ssh.PasswordAuth{Password: server.SSHPassword}
	} else {
		auth = &ssh.KeyAuth{PrivateKey: server.SSHKey}
	}

	if err := h.accountSvc.SyncAllToRemote(serverID, auth); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "sync failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "synced"})
}

package handler

import (
	"database/sql"
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
	c.JSON(http.StatusOK, accounts)
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

	account, err := h.accountRepo.Create(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	// Sync to remote server
	server, err := h.serverRepo.GetByID(serverID)
	if err == nil {
		var auth ssh.SSHAuth
		if server.SSHKeyType == "password" {
			auth = &ssh.PasswordAuth{Password: server.SSHPassword}
		} else {
			auth = &ssh.KeyAuth{PrivateKey: server.SSHKey}
		}
		// Sync non-blocking
		go h.accountSvc.SyncAllToRemote(serverID, auth)
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
	if err := h.accountRepo.Delete(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	// Sync remaining accounts to remote server
	server, err := h.serverRepo.GetByID(account.ServerID)
	if err == nil {
		var auth ssh.SSHAuth
		if server.SSHKeyType == "password" {
			auth = &ssh.PasswordAuth{Password: server.SSHPassword}
		} else {
			auth = &ssh.KeyAuth{PrivateKey: server.SSHKey}
		}
		// Sync remaining accounts (non-blocking, ignore error)
		go h.accountSvc.SyncAllToRemote(account.ServerID, auth)
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

	server, err := h.serverRepo.GetByID(account.ServerID)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "server not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	var content string
	var subErr error
	switch subType {
	case "clash_meta":
		content, subErr = h.accountSvc.GenerateClashMetaSubscription([]*model.Account{account}, server.IP)
	case "singbox":
		content, subErr = h.accountSvc.GenerateSingBoxSubscription([]*model.Account{account}, server.IP)
	default:
		content = h.accountSvc.GenerateVLESSSubscription([]*model.Account{account}, server.IP)
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

	// 获取服务器信息以便创建 SSH 连接
	server, err := h.serverRepo.GetByID(serverID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server not found"})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to import accounts"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "imported",
		"accounts": accounts,
	})
}
func (h *AccountHandler) Sync(c *gin.Context) {
	id := c.Param("id")

	account, err := h.accountRepo.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "account not found"})
		return
	}

	server, err := h.serverRepo.GetByID(account.ServerID)
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

	server, err := h.serverRepo.GetByID(serverID)
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

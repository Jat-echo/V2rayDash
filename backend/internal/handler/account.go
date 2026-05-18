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
	}

	accountRoutes := r.Group("/accounts")
	{
		accountRoutes.GET("/:id", h.Get)
		accountRoutes.PUT("/:id", h.Update)
		accountRoutes.DELETE("/:id", h.Delete)
		accountRoutes.GET("/:id/subscribe", h.Subscribe)
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
	if err := h.accountRepo.Delete(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
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
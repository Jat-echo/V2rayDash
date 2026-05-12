package handler

import (
	"database/sql"
	"encoding/base64"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"v2ray-dash/backend/internal/model"
	"v2ray-dash/backend/internal/repository"
)

type SubscriptionHandler struct {
	repo    *repository.SubscriptionRepository
	logRepo *repository.LogRepository
}

func NewSubscriptionHandler(db *sql.DB) *SubscriptionHandler {
	return &SubscriptionHandler{
		repo:    repository.NewSubscriptionRepository(db),
		logRepo: repository.NewLogRepository(db),
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

	// 生成 Base64 编码的订阅链接
	link := fmt.Sprintf("https://your-domain.com/api/subscribe/%s", sub.UUID)
	encoded := base64.StdEncoding.EncodeToString([]byte(link))

	c.JSON(http.StatusOK, gin.H{
		"link":    link,
		"encoded": encoded,
	})
}

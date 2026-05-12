package handler

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
	"v2ray-dash/backend/internal/model"
	"v2ray-dash/backend/internal/repository"
)

type ServerHandler struct {
	repo     *repository.ServerRepository
	logRepo  *repository.LogRepository
}

func NewServerHandler(db *sql.DB) *ServerHandler {
	return &ServerHandler{
		repo:    repository.NewServerRepository(db),
		logRepo: repository.NewLogRepository(db),
	}
}

func (h *ServerHandler) List(c *gin.Context) {
	servers, err := h.repo.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, servers)
}

func (h *ServerHandler) Get(c *gin.Context) {
	id := c.Param("id")
	server, err := h.repo.GetByID(id)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "server not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, server)
}

func (h *ServerHandler) Create(c *gin.Context) {
	var req model.CreateServerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	server, err := h.repo.Create(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 记录操作日志
	h.logRepo.Create(&model.OperationLog{
		Operator:   "admin",
		Action:     "create_server",
		TargetType: "server",
		TargetID:   server.ID,
		Detail:     map[string]any{"name": server.Name, "ip": server.IP},
		IP:         c.ClientIP(),
	})

	c.JSON(http.StatusCreated, server)
}

func (h *ServerHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var req model.UpdateServerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	server, err := h.repo.Update(id, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, server)
}

func (h *ServerHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if err := h.repo.Delete(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	h.logRepo.Create(&model.OperationLog{
		Operator:   "admin",
		Action:     "delete_server",
		TargetType: "server",
		TargetID:   id,
		IP:         c.ClientIP(),
	})

	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

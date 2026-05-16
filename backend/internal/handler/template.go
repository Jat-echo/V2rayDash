package handler

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
	"v2ray-dash/backend/internal/model"
	"v2ray-dash/backend/internal/repository"
)

type TemplateHandler struct {
	repo *repository.TemplateRepository
}

func NewTemplateHandler(db *sql.DB) *TemplateHandler {
	return &TemplateHandler{repo: repository.NewTemplateRepository(db)}
}

func (h *TemplateHandler) List(c *gin.Context) {
	templates, err := h.repo.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, templates)
}

func (h *TemplateHandler) Create(c *gin.Context) {
	var tmpl model.Template
	if err := c.ShouldBindJSON(&tmpl); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate required fields
	if tmpl.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
		return
	}
	if tmpl.Config.Port < 1 || tmpl.Config.Port > 65535 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "port must be between 1 and 65535"})
		return
	}
	if tmpl.Config.Port == 0 {
		tmpl.Config.Port = 443 // default
	}
	if tmpl.Config.ReportInterval == 0 {
		tmpl.Config.ReportInterval = 30 // default
	}

	if err := h.repo.Create(&tmpl); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, tmpl)
}

func (h *TemplateHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if err := h.repo.Delete(id); err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "template not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

func (h *TemplateHandler) RegisterRoutes(rg *gin.RouterGroup) {
	templates := rg.Group("/templates")
	templates.GET("", h.List)
	templates.POST("", h.Create)
	templates.DELETE("/:id", h.Delete)
}
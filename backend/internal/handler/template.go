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

	if err := h.repo.Create(&tmpl); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, tmpl)
}

func (h *TemplateHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if err := h.repo.Delete(id); err != nil {
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
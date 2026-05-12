package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"v2ray-dash/backend/internal/model"
	"v2ray-dash/backend/internal/repository"
)

type LogHandler struct {
	repo *repository.LogRepository
}

func NewLogHandler(repo *repository.LogRepository) *LogHandler {
	return &LogHandler{repo: repo}
}

func (h *LogHandler) ListOperationLogs(c *gin.Context) {
	filter := &model.OperationLogFilter{
		// 从 query 参数构建 filter
	}

	logs, err := h.repo.List(filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, logs)
}

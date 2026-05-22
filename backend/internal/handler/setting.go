package handler

import (
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"v2ray-dash/backend/internal/repository"
)

type SettingHandler struct {
	repo *repository.SettingRepository
}

func NewSettingHandler(repo *repository.SettingRepository) *SettingHandler {
	return &SettingHandler{repo: repo}
}

// fetchPublicIP attempts to get public IP from multiple services
func fetchPublicIP() (string, error) {
	// Try multiple services in order of preference
	services := []string{
		"https://api.ipify.org?format=text",
		"https://icanhazip.com",
		"https://ifconfig.me/ip",
		"http://checkip.amazonaws.com",
		"https://api.my-ip.io/v1/ip.json",
	}

	for _, service := range services {
		resp, err := http.Get(service)
		if err != nil {
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			continue
		}

		// Read IP address
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			continue
		}

		ip := strings.TrimSpace(string(body))
		// Validate IP format (basic check)
		if len(ip) > 6 && len(ip) < 45 {
			return ip, nil
		}
	}

	return "", nil
}

func (h *SettingHandler) GetPublicIP(c *gin.Context) {
	ip, err := fetchPublicIP()
	if err != nil || ip == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get public IP"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ip": ip})
}

func (h *SettingHandler) GetPublicURL(c *gin.Context) {
	setting, err := h.repo.Get("public_url")
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"value": "http://localhost:8080"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"value": setting.Value})
}

func (h *SettingHandler) UpdatePublicURL(c *gin.Context) {
	var req struct {
		Value string `json:"value" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.repo.Update("public_url", req.Value); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update setting"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "updated", "value": req.Value})
}

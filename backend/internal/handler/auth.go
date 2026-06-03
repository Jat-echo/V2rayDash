package handler

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"v2ray-dash/backend/internal/repository"
)

type AuthHandler struct {
	authRepo    *repository.AuthRepository
	settingRepo *repository.SettingRepository
}

func NewAuthHandler(authRepo *repository.AuthRepository, settingRepo *repository.SettingRepository) *AuthHandler {
	return &AuthHandler{authRepo: authRepo, settingRepo: settingRepo}
}

func (h *AuthHandler) Status(c *gin.Context) {
	count, _ := h.authRepo.CountAdmins()
	c.JSON(http.StatusOK, gin.H{"setup_required": count == 0})
}

func (h *AuthHandler) Setup(c *gin.Context) {
	count, _ := h.authRepo.CountAdmins()
	if count > 0 {
		c.JSON(http.StatusForbidden, gin.H{"error": "管理员账号已存在"})
		return
	}
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if len(req.Username) < 3 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "用户名至少3个字符"})
		return
	}
	if len(req.Password) < 6 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "密码至少6个字符"})
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "密码加密失败"})
		return
	}
	if err := h.authRepo.Create(req.Username, string(hash)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建账号失败"})
		return
	}
	token, err := newToken()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "生成令牌失败"})
		return
	}
	expires := time.Now().Add(7 * 24 * time.Hour).Unix()
	h.settingRepo.Update("auth_token", fmt.Sprintf("%s:%d:%s", token, expires, req.Username))
	c.JSON(http.StatusOK, gin.H{"token": token, "username": req.Username})
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	admin, err := h.authRepo.GetByUsername(req.Username)
	if err != nil || admin == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户名或密码错误"})
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(admin.PasswordHash), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户名或密码错误"})
		return
	}
	token, err := newToken()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "生成令牌失败"})
		return
	}
	expires := time.Now().Add(7 * 24 * time.Hour).Unix()
	h.settingRepo.Update("auth_token", fmt.Sprintf("%s:%d:%s", token, expires, admin.Username))
	c.JSON(http.StatusOK, gin.H{"token": token, "username": admin.Username})
}

func (h *AuthHandler) ChangePassword(c *gin.Context) {
	username, _ := c.Get("username")
	var req struct {
		OldPassword string `json:"old_password" binding:"required"`
		NewPassword string `json:"new_password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if len(req.NewPassword) < 6 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "新密码至少6个字符"})
		return
	}
	admin, err := h.authRepo.GetByUsername(username.(string))
	if err != nil || admin == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "账号不存在"})
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(admin.PasswordHash), []byte(req.OldPassword)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "原密码错误"})
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "加密失败"})
		return
	}
	if err := h.authRepo.UpdatePassword(admin.ID, string(hash)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "修改失败"})
		return
	}
	// 使旧 token 失效，强制重新登录
	h.settingRepo.Update("auth_token", "")
	c.JSON(http.StatusOK, gin.H{"message": "密码已修改，请重新登录"})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	h.settingRepo.Update("auth_token", "")
	c.JSON(http.StatusOK, gin.H{"message": "已退出"})
}

// AuthMiddleware validates Bearer token on protected routes
func AuthMiddleware(settingRepo *repository.SettingRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
			c.Abort()
			return
		}
		token := strings.TrimPrefix(authHeader, "Bearer ")

		setting, err := settingRepo.Get("auth_token")
		if err != nil || setting == nil || setting.Value == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
			c.Abort()
			return
		}

		parts := strings.SplitN(setting.Value, ":", 3)
		if len(parts) != 3 || subtle.ConstantTimeCompare([]byte(parts[0]), []byte(token)) != 1 {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "令牌无效"})
			c.Abort()
			return
		}

		expires, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil || time.Now().Unix() > expires {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "登录已过期，请重新登录"})
			c.Abort()
			return
		}

		c.Set("username", parts[2])
		c.Next()
	}
}

func newToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

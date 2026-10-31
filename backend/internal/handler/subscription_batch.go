package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"strings"
	"v2ray-dash/backend/internal/model"
	"v2ray-dash/backend/internal/ssh"
)

func (h *SubscriptionHandler) syncTargetServers(ids []string) {
	seen := map[string]bool{}
	for _, id := range ids {
		if seen[id] {
			continue
		}
		seen[id] = true
		server, err := h.serverRepo.GetByIDForInstall(id)
		if err != nil || server.Node != nil {
			continue
		} // Managed targets were marked pending inside the transaction.
		var auth ssh.SSHAuth
		if server.SSHKeyType == "password" {
			auth = &ssh.PasswordAuth{Password: server.SSHPassword}
		} else {
			auth = &ssh.KeyAuth{PrivateKey: server.SSHKey}
		}
		if auth.AuthMethod() != nil {
			h.accountSvc.RequestSync(id, auth)
		}
	}
}

func (h *SubscriptionHandler) Batch(c *gin.Context) {
	var req model.BatchSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "批量参数无效"})
		return
	}
	if req.Action != "" && req.Action != "add" && req.Action != "remove" {
		c.JSON(400, gin.H{"error": "关联操作无效"})
		return
	}
	if req.Action == "remove" && len(req.Names) > 0 {
		c.JSON(400, gin.H{"error": "解除关联请选择已有订阅"})
		return
	}
	if (len(req.Names) == 0) == (len(req.SubscriptionIDs) == 0) || len(req.Names) > 200 || len(req.SubscriptionIDs) > 200 || req.TrafficLimit < 0 {
		c.JSON(400, gin.H{"error": "选择新建订阅名称或已有订阅，每批 1–200 个"})
		return
	}
	if (req.ServerID == "") == (req.CombinationID == "") {
		c.JSON(400, gin.H{"error": "选择一个直连节点或代理链路"})
		return
	}
	targetID := req.ServerID
	if targetID == "" {
		targetID = req.CombinationID
	}
	if _, err := uuid.Parse(targetID); err != nil {
		c.JSON(400, gin.H{"error": "节点标识无效"})
		return
	}
	seen := map[string]bool{}
	for i, name := range req.Names {
		name = strings.TrimSpace(name)
		if name == "" || len([]rune(name)) > 100 || seen[name] {
			c.JSON(400, gin.H{"error": "每行一个不同的订阅名称，最多 100 字"})
			return
		}
		req.Names[i] = name
		seen[name] = true
	}
	for _, id := range req.SubscriptionIDs {
		if _, err := uuid.Parse(id); err != nil || seen[id] {
			c.JSON(400, gin.H{"error": "订阅标识无效或重复"})
			return
		}
		seen[id] = true
	}
	mapping := model.AccountMapping{ServerID: req.ServerID, CombinationID: req.CombinationID, AutoCreate: true}
	results := []model.BatchSubscriptionResult{}
	affected := []string{}
	for _, name := range req.Names {
		id, servers, err := h.repo.CreateWithTargets(&model.CreateSubscriptionRequest{Name: name, Remark: req.Remark, TrafficLimit: req.TrafficLimit, AccountMappings: []model.AccountMapping{mapping}})
		item := model.BatchSubscriptionResult{ID: id, Name: name, Status: "created"}
		if err != nil {
			item.Status = "failed"
			item.Error = err.Error()
		} else {
			affected = append(affected, servers...)
		}
		results = append(results, item)
	}
	for _, id := range req.SubscriptionIDs {
		item := model.BatchSubscriptionResult{ID: id, Name: id, Status: "added"}
		if sub, err := h.repo.GetByID(id); err == nil {
			item.Name = sub.Name
		}
		var changed bool
		var servers []string
		var err error
		if req.Action == "remove" {
			changed, servers, err = h.repo.RemoveTarget(id, mapping)
			item.Status = "removed"
		} else {
			changed, servers, err = h.repo.AddTarget(id, mapping)
		}
		if err != nil {
			item.Status = "failed"
			item.Error = err.Error()
		} else if !changed {
			item.Status = "skipped"
		} else {
			affected = append(affected, servers...)
		}
		results = append(results, item)
	}
	h.syncTargetServers(affected)
	h.logRepo.Create(&model.OperationLog{Operator: "admin", Action: "batch_subscription_nodes", TargetType: "subscription", Detail: map[string]any{"action": req.Action, "server_id": req.ServerID, "combination_id": req.CombinationID, "results": results}, IP: c.ClientIP()})
	c.JSON(200, gin.H{"results": results})
}

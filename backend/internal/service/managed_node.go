package service

import (
	"bytes"
	"crypto/ecdh"
	"database/sql"
	_ "embed"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"v2ray-dash/backend/internal/model"
	"v2ray-dash/backend/internal/repository"
	"v2ray-dash/backend/internal/ssh"
)

//go:embed node_remote.py
var nodeRemoteScript string

//go:embed node_dependencies.sh
var nodeDependenciesScript string

// ponytail: serial node operations suit the private panel; use per-node locks if concurrent publishing becomes necessary.
var nodeOperationMu sync.Mutex

type NodeService struct {
	accounts *repository.AccountRepository
	servers  *repository.ServerRepository
}

func NewNodeService(a *repository.AccountRepository, s *repository.ServerRepository) *NodeService {
	return &NodeService{a, s}
}

type remoteNode struct {
	Layout     string           `json:"layout"`
	Hash       string           `json:"hash"`
	Backup     string           `json:"backup"`
	EgressIP   string           `json:"egress_ip"`
	Config     map[string]any   `json:"config"`
	Inbound    map[string]any   `json:"inbound"`
	PublicPort int              `json:"public_port"`
	Error      string           `json:"error"`
	Checks     []map[string]any `json:"checks"`
}

func object(v any) map[string]any {
	m, _ := v.(map[string]any)
	if m == nil {
		return map[string]any{}
	}
	return m
}
func array(v any) []any      { a, _ := v.([]any); return a }
func textValue(v any) string { s, _ := v.(string); return s }

func nodePythonCommand(client *ssh.SSHClient, script string, needsCurl bool) (string, error) {
	var out, stderr bytes.Buffer
	bootstrap := "sh -s --"
	if needsCurl {
		bootstrap += " curl"
	}
	err := client.ExecuteInput(bootstrap, strings.NewReader(nodeDependenciesScript), &out, &stderr)
	mode := strings.TrimSpace(out.String())
	if err != nil {
		reason := map[string]string{
			"error:privilege":           "需要 root 或免密 sudo 权限",
			"error:timeout_unavailable": "节点缺少 timeout 命令，无法限制安装时长",
			"error:temporary_file":      "无法创建临时文件，请检查磁盘空间和权限",
			"error:package_manager":     "缺少依赖，系统没有支持的 apt-get、dnf 或 yum 包管理器",
			"error:install_timeout":     "安装超时，请检查软件源网络后重试",
			"error:package_lock":        "包管理器被其他安装任务占用，请稍后重试",
			"error:disk_space":          "磁盘空间不足",
			"error:package_network":     "软件源连接或证书校验失败，请检查节点网络和软件源",
			"error:install_failed":      "软件包安装失败，请检查节点包管理器状态",
			"error:python_version":      "需要 Python 3.8 或更高版本并支持 SSL，系统软件源未提供可用版本",
			"error:certificates":        "系统 CA 证书不可用",
			"error:curl":                "安装后仍无法找到 curl",
		}[mode]
		if reason == "" {
			reason = "SSH 检查未完成或连接中断，请重试"
		}
		return "", fmt.Errorf("节点运行依赖准备失败：%s", reason)
	}
	command := "python3 -c '" + strings.ReplaceAll(script, "'", "'\"'\"'") + "'"
	if mode == "sudo" {
		command = "sudo -n " + command
	} else if mode != "root" {
		return "", fmt.Errorf("节点运行依赖检查返回异常结果")
	}
	return command, nil
}

func remoteNodeCall(client *ssh.SSHClient, payload map[string]any) (*remoteNode, error) {
	input, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	command, err := nodePythonCommand(client, nodeRemoteScript, payload["operation"] == "publish" || payload["operation"] == "probe")
	if err != nil {
		return nil, err
	}
	var out, stderr bytes.Buffer
	execErr := client.ExecuteInput(command, bytes.NewReader(input), &out, &stderr)
	var result remoteNode
	if err = json.Unmarshal(out.Bytes(), &result); err != nil {
		return nil, fmt.Errorf("节点操作未完成，请检查 SSH、Python 3 和 sudo 权限")
	}
	if result.Error != "" {
		return nil, errors.New(result.Error)
	}
	if execErr != nil {
		return nil, fmt.Errorf("节点操作失败，请检查服务状态")
	}
	return &result, nil
}

func realityFromInbound(inbound map[string]any, current *model.Server) (*RealityConfig, error) {
	settings := object(object(inbound["streamSettings"])["realitySettings"])
	raw, err := base64.RawURLEncoding.DecodeString(textValue(settings["privateKey"]))
	if err != nil {
		return nil, fmt.Errorf("无法读取节点的 Reality 密钥")
	}
	key, err := ecdh.X25519().NewPrivateKey(raw)
	if err != nil {
		return nil, fmt.Errorf("节点 Reality 密钥无效")
	}
	ids, names := array(settings["shortIds"]), array(settings["serverNames"])
	if len(ids) == 0 || len(names) == 0 {
		return nil, fmt.Errorf("节点缺少 Reality Short ID 或 SNI")
	}
	sid := textValue(ids[0])
	name := textValue(names[0])
	for _, id := range ids {
		if textValue(id) == RealityShortID(current) {
			sid = textValue(id)
			break
		}
	}
	for _, candidate := range names {
		if textValue(candidate) == current.RealityServerName {
			name = textValue(candidate)
			break
		}
	}
	if _, err = hex.DecodeString(sid); err != nil || len(sid) > 16 {
		return nil, fmt.Errorf("Reality Short ID 无效")
	}
	port, _ := inbound["port"].(float64)
	if port < 1 || port > 65535 {
		return nil, fmt.Errorf("节点端口无效")
	}
	return &RealityConfig{Enabled: true, ServerName: name, PublicKey: base64.RawURLEncoding.EncodeToString(key.PublicKey().Bytes()), Port: int(port), ShortID: sid, ShortIDConfigured: true}, nil
}

func (s *NodeService) Adopt(serverID, layout string) (*model.ManagedNode, error) {
	if !nodeOperationMu.TryLock() {
		return nil, fmt.Errorf("正在操作其他节点，请稍后重试")
	}
	defer nodeOperationMu.Unlock()
	if layout != "" && layout != "relay" && layout != "exit" && layout != "legacy" {
		return nil, fmt.Errorf("不支持的服务类型")
	}
	server, err := s.servers.GetByIDForInstall(serverID)
	if err != nil {
		return nil, err
	}
	if server.Node != nil && server.Node.State == "publishing" {
		return nil, fmt.Errorf("节点正在发布，请稍后再接管")
	}
	client, err := NewServerSSHClient(s.servers, server, nil)
	if err != nil {
		return nil, err
	}
	defer client.Close()
	info, err := remoteNodeCall(client, map[string]any{"operation": "inspect", "layout": layout, "egress": true})
	if err != nil {
		return nil, err
	}
	if info.Layout == "relay" {
		for _, raw := range array(info.Config["inbounds"]) {
			in := object(raw)
			if textValue(in["protocol"]) != "vless" && (textValue(in["tag"]) != "api-in" || textValue(in["listen"]) != "127.0.0.1") {
				return nil, fmt.Errorf("中转包含自定义入站，暂不支持直接接管")
			}
		}
		for _, raw := range array(object(info.Config["routing"])["rules"]) {
			rule := object(raw)
			if textValue(rule["outboundTag"]) != "api" && len(array(rule["user"])) == 0 {
				return nil, fmt.Errorf("中转包含自定义路由，请先核对后再接管")
			}
		}
	}
	reality, err := realityFromInbound(info.Inbound, server)
	if err != nil {
		return nil, err
	}
	if info.PublicPort > 0 && info.PublicPort <= 65535 {
		reality.Port = info.PublicPort
	}
	clients := array(object(info.Inbound["settings"])["clients"])
	observedBindings := map[string]string{}
	for _, raw := range clients {
		id := textValue(object(raw)["id"])
		if existing, e := s.accounts.GetByUUID(id); e == nil && existing.ServerID != serverID {
			return nil, fmt.Errorf("发现其他节点已使用的账号 UUID，请使用受控迁移而非重复导入")
		}
	}
	role := "exit"
	if info.Layout == "relay" {
		role = "relay"
	}
	if server.Role != role {
		return nil, fmt.Errorf("已有服务类型与节点角色不符，请先核对节点角色")
	}
	n := &model.ManagedNode{ServerID: serverID, Role: role, Layout: info.Layout, InboundTag: textValue(info.Inbound["tag"]), ShortID: reality.ShortID, EgressIP: info.EgressIP, ConfigHash: info.Hash, SSHHostKey: client.HostKey}
	if err = s.servers.SaveNode(n); err != nil {
		return nil, err
	}
	if _, err = s.servers.Update(serverID, &model.UpdateServerRequest{RealityEnabled: &reality.Enabled, RealityServerName: &reality.ServerName, RealityPublicKey: &reality.PublicKey, RealityPort: &reality.Port}); err != nil {
		return nil, err
	}
	for _, raw := range clients {
		item := object(raw)
		id := textValue(item["id"])
		email := textValue(item["email"])
		if email == "" {
			email = id
		}
		acc, e := s.accounts.GetByUUID(id)
		if errors.Is(e, sql.ErrNoRows) {
			acc, e = s.accounts.Upsert(&model.CreateAccountRequest{ServerID: serverID, UUID: id, Email: email, Protocols: []string{"vless_reality_vision"}})
		}
		if e != nil {
			return nil, e
		}
		if err = s.accounts.SetStatsKey(acc.ID, email); err != nil {
			return nil, err
		}
		if role == "relay" {
			for _, ruleRaw := range array(object(info.Config["routing"])["rules"]) {
				rule := object(ruleRaw)
				matched := false
				for _, value := range array(rule["user"]) {
					if textValue(value) == email {
						matched = true
					}
				}
				if !matched {
					continue
				}
				for _, outRaw := range array(info.Config["outbounds"]) {
					out := object(outRaw)
					if textValue(out["tag"]) != textValue(rule["outboundTag"]) {
						continue
					}
					peers := array(object(out["settings"])["vnext"])
					if len(peers) != 1 {
						continue
					}
					upUsers := array(object(peers[0])["users"])
					if len(upUsers) != 1 {
						continue
					}
					upstream, e := s.accounts.GetByUUID(textValue(object(upUsers[0])["id"]))
					if e != nil {
						continue
					}
					if s.servers.ValidateRoute(acc.ID, upstream.ID) == nil {
						if e = s.servers.BindRoute(acc.ID, upstream.ID); e != nil {
							return nil, e
						}
						observedBindings[acc.ID] = upstream.ID
					}
				}
			}
		}
	}
	imported, err := s.accounts.ListByServerID(serverID)
	if err != nil {
		return nil, err
	}
	live := map[string]bool{}
	for _, a := range imported {
		for _, raw := range clients {
			if textValue(object(raw)["id"]) == a.UUID {
				live[a.ID] = true
			}
		}
	}
	if err = s.servers.SavePublishedAccounts(imported, live); err != nil {
		return nil, err
	}
	allRoutes, err := s.servers.ListRoutes()
	if err != nil {
		return nil, err
	}
	importedRoutes := []*model.RelayRoute{}
	for _, r := range allRoutes {
		if live[r.AccountID] && observedBindings[r.AccountID] == r.ExitAccountID {
			importedRoutes = append(importedRoutes, r)
		}
	}
	if err = s.servers.MarkRoutesApplied(importedRoutes); err != nil {
		return nil, err
	}
	return s.servers.GetNode(serverID)
}

type managedUser struct{ ID, Email string }
type managedExit struct {
	UUID   string
	Server *model.Server
}

// buildManagedConfig controls the selected inbound and relay routing while preserving node keys and API listeners.
func buildManagedConfig(source map[string]any, node *model.ManagedNode, users []managedUser, exits map[string]managedExit) (map[string]any, error) {
	data, err := json.Marshal(source)
	if err != nil {
		return nil, err
	}
	var result map[string]any
	if err = json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	clients := make([]any, 0, len(users))
	rules := make([]any, 0)
	outbounds := []any{map[string]any{"tag": "blocked", "protocol": "blackhole"}}
	addedExits := map[string]bool{}
	for _, u := range users {
		clients = append(clients, map[string]any{"id": u.ID, "email": u.Email, "flow": "xtls-rprx-vision", "level": 0})
		if node.Role != "relay" {
			continue
		}
		e, ok := exits[u.ID]
		if !ok || e.Server == nil || e.Server.Node == nil {
			return nil, fmt.Errorf("存在未绑定落地的启用账号")
		}
		tag := "vd-" + e.UUID
		if !addedExits[tag] {
			outbounds = append(outbounds, map[string]any{"tag": tag, "protocol": "vless", "settings": map[string]any{"vnext": []any{map[string]any{"address": e.Server.IP, "port": e.Server.RealityPort, "users": []any{map[string]any{"id": e.UUID, "encryption": "none", "flow": "xtls-rprx-vision"}}}}}, "streamSettings": map[string]any{"network": "tcp", "security": "reality", "realitySettings": map[string]any{"serverName": e.Server.RealityServerName, "publicKey": e.Server.RealityPublicKey, "shortId": e.Server.Node.ShortID, "fingerprint": "chrome"}}})
			addedExits[tag] = true
		}
		rules = append(rules, map[string]any{"type": "field", "user": []any{u.Email}, "outboundTag": tag})
	}
	matched := 0
	for _, raw := range array(result["inbounds"]) {
		inbound := object(raw)
		if textValue(inbound["tag"]) == node.InboundTag && textValue(inbound["protocol"]) == "vless" && textValue(object(inbound["streamSettings"])["security"]) == "reality" {
			object(inbound["settings"])["clients"] = clients
			matched++
		}
	}
	if matched != 1 {
		return nil, fmt.Errorf("接管的入站已发生变化")
	}
	if node.Role == "relay" {
		result["outbounds"] = outbounds
		routing := object(result["routing"])
		kept := []any{}
		for _, raw := range array(routing["rules"]) {
			rule := object(raw)
			if textValue(rule["outboundTag"]) == "api" {
				kept = append(kept, rule)
			}
		}
		routing["rules"] = append(kept, rules...)
		result["routing"] = routing
	}
	if node.Layout == "exit" {
		result["stats"] = map[string]any{}
		result["api"] = map[string]any{"tag": "api", "services": []any{"StatsService"}}
		policy := object(result["policy"])
		levels := object(policy["levels"])
		zero := object(levels["0"])
		zero["statsUserUplink"] = true
		zero["statsUserDownlink"] = true
		levels["0"] = zero
		policy["levels"] = levels
		result["policy"] = policy
		hasAPI := false
		for _, raw := range array(result["inbounds"]) {
			if textValue(object(raw)["tag"]) == "api-in" {
				hasAPI = true
			}
		}
		if !hasAPI {
			result["inbounds"] = append(array(result["inbounds"]), map[string]any{"tag": "api-in", "listen": "127.0.0.1", "port": 10085, "protocol": "dokodemo-door", "settings": map[string]any{"address": "127.0.0.1"}})
		}
		routing := object(result["routing"])
		rules := array(routing["rules"])
		kept := []any{}
		for _, r := range rules {
			if textValue(object(r)["outboundTag"]) != "api" {
				kept = append(kept, r)
			}
		}
		routing["rules"] = append([]any{map[string]any{"type": "field", "inboundTag": []any{"api-in"}, "outboundTag": "api"}}, kept...)
		result["routing"] = routing
	}
	return result, nil
}

func (s *NodeService) Publish(serverID string) (checks []map[string]any, err error) {
	nodeOperationMu.Lock()
	defer nodeOperationMu.Unlock()
	return s.publishLocked(serverID)
}

func (s *NodeService) publishLocked(serverID string) (checks []map[string]any, err error) {
	defer func() {
		if err != nil {
			_ = s.servers.SetNodeState(serverID, "failed", err.Error(), "")
		}
	}()
	server, err := s.servers.GetByIDForInstall(serverID)
	if err != nil {
		return nil, err
	}
	if server.Node == nil {
		return nil, fmt.Errorf("请先接管已有服务")
	}
	if err = s.servers.SetNodeState(serverID, "publishing", "", ""); err != nil {
		return nil, err
	}
	client, err := NewServerSSHClient(s.servers, server, nil)
	if err != nil {
		return nil, err
	}
	defer client.Close()
	info, err := remoteNodeCall(client, map[string]any{"operation": "inspect", "layout": server.Node.Layout})
	if err != nil {
		return nil, err
	}
	if info.Hash != server.Node.ConfigHash {
		return nil, fmt.Errorf("远程配置已变化，请重新接管核对后发布")
	}
	// Prepare each combination's dedicated exit connection even before it has customers.
	if server.Node.Role == "relay" {
		ids, e := s.servers.CombinationExitServers(serverID)
		if e != nil {
			return nil, e
		}
		for _, id := range ids {
			exit, e := s.servers.GetNode(id)
			if e != nil {
				return nil, fmt.Errorf("请先接管链路的落地服务")
			}
			if exit.State == "pending" || exit.State == "failed" {
				if _, e = s.publishLocked(id); e != nil {
					return nil, fmt.Errorf("落地节点发布失败: %w", e)
				}
			} else if exit.State == "publishing" {
				return nil, fmt.Errorf("请先确认落地节点配置发布成功")
			}
		}
	}
	accounts, err := s.accounts.ListByServerID(serverID)
	if err != nil {
		return nil, err
	}
	routes, err := s.servers.ListRoutes()
	if err != nil {
		return nil, err
	}
	byAccount := map[string]*model.RelayRoute{}
	for _, r := range routes {
		byAccount[r.AccountID] = r
	}
	users := []managedUser{}
	exits := map[string]managedExit{}
	applied := []*model.RelayRoute{}
	probes := []map[string]any{}
	enabledAccounts := map[string]bool{}
	probedExits := map[string]bool{}
	for _, a := range accounts {
		enabled, e := s.accounts.EffectiveEnabled(a.ID)
		if e != nil {
			return nil, e
		}
		if !enabled {
			continue
		}
		key, e := s.accounts.StatsKey(a.ID)
		if e != nil {
			return nil, e
		}
		users = append(users, managedUser{ID: a.UUID, Email: key})
		enabledAccounts[a.ID] = true
		expected := server.Node.EgressIP
		probeKey := serverID
		if server.Node.Role == "relay" {
			r := byAccount[a.ID]
			if r == nil {
				return nil, fmt.Errorf("账号 %s 尚未绑定落地", a.Email)
			}
			if e = s.servers.ValidateRoute(a.ID, r.ExitAccountID); e != nil {
				return nil, e
			}
			exitAccount, e := s.accounts.GetByID(r.ExitAccountID)
			if e != nil {
				return nil, e
			}
			exitServer, e := s.servers.GetByID(exitAccount.ServerID)
			if e != nil {
				return nil, e
			}
			if exitServer.Node.State == "pending" {
				if _, e = s.publishLocked(exitServer.ID); e != nil {
					return nil, fmt.Errorf("落地节点发布失败: %w", e)
				}
				exitServer, e = s.servers.GetByID(exitServer.ID)
				if e != nil {
					return nil, e
				}
			}
			installed, e := s.servers.AccountInstalled(exitAccount)
			if e != nil {
				return nil, e
			}
			if !installed {
				return nil, fmt.Errorf("落地账号尚未发布，请先发布落地节点")
			}
			if exitServer.Node.State == "pending" || exitServer.Node.State == "publishing" || exitServer.Node.State == "failed" {
				return nil, fmt.Errorf("请先发布落地节点 %s 的配置", exitServer.Name)
			}
			exits[a.UUID] = managedExit{UUID: exitAccount.UUID, Server: exitServer}
			applied = append(applied, r)
			expected = exitServer.Node.EgressIP
			probeKey = exitServer.ID
		}
		if !probedExits[probeKey] {
			probes = append(probes, map[string]any{"account_id": a.ID, "uuid": a.UUID, "sni": server.RealityServerName, "public_key": server.RealityPublicKey, "short_id": server.Node.ShortID, "expected_ip": expected})
			probedExits[probeKey] = true
		}
	}
	candidate, err := buildManagedConfig(info.Config, server.Node, users, exits)
	if err != nil {
		return nil, err
	}
	before, _ := json.Marshal(info.Config)
	after, _ := json.Marshal(candidate)
	if bytes.Equal(before, after) {
		if err = s.servers.FinalizePublish(serverID, info.Hash, applied, accounts, enabledAccounts); err != nil {
			return nil, err
		}
		return []map[string]any{{"unchanged": true, "viewpoint": "node"}}, nil
	}
	result, err := remoteNodeCall(client, map[string]any{"operation": "publish", "layout": server.Node.Layout, "expected_hash": info.Hash, "config": candidate, "probes": probes})
	if err != nil {
		return nil, err
	}
	if err = s.servers.FinalizePublish(serverID, result.Hash, applied, accounts, enabledAccounts); err != nil {
		_, rollbackErr := remoteNodeCall(client, map[string]any{"operation": "rollback", "layout": server.Node.Layout, "expected_hash": result.Hash, "backup": result.Backup})
		if rollbackErr != nil {
			return nil, fmt.Errorf("发布结果保存失败，远程恢复未确认，请重新读取节点核对")
		}
		return nil, fmt.Errorf("发布结果保存失败，已恢复节点旧配置，请重新读取核对")
	}
	return result.Checks, nil
}

func (s *NodeService) Probe(serverID string) ([]map[string]any, error) {
	server, err := s.servers.GetByIDForInstall(serverID)
	if err != nil {
		return nil, err
	}
	if server.Node == nil {
		return nil, fmt.Errorf("请先接管已有服务")
	}
	client, err := NewServerSSHClient(s.servers, server, nil)
	if err != nil {
		return nil, err
	}
	defer client.Close()
	info, err := remoteNodeCall(client, map[string]any{"operation": "inspect", "layout": server.Node.Layout})
	if err != nil {
		return nil, err
	}
	accounts, err := s.accounts.ListByServerID(serverID)
	if err != nil {
		return nil, err
	}
	routes, err := s.servers.ListRoutes()
	if err != nil {
		return nil, err
	}
	byID := map[string]*model.RelayRoute{}
	for _, r := range routes {
		byID[r.AccountID] = r
	}
	live := map[string]bool{}
	for _, v := range array(object(info.Inbound["settings"])["clients"]) {
		live[textValue(object(v)["id"])] = true
	}
	probes := []map[string]any{}
	for _, a := range accounts {
		if !live[a.UUID] {
			continue
		}
		expected := server.Node.EgressIP
		if server.Node.Role == "relay" {
			r := byID[a.ID]
			if r == nil || r.AppliedExitAccountID == "" {
				continue
			}
			e, err := s.accounts.GetByID(r.AppliedExitAccountID)
			if err != nil {
				return nil, err
			}
			es, err := s.servers.GetByID(e.ServerID)
			if err != nil || es.Node == nil {
				return nil, fmt.Errorf("落地节点元数据缺失")
			}
			expected = es.Node.EgressIP
		}
		probes = append(probes, map[string]any{"account_id": a.ID, "uuid": a.UUID, "sni": server.RealityServerName, "public_key": server.RealityPublicKey, "short_id": server.Node.ShortID, "expected_ip": expected})
	}
	if len(probes) == 0 {
		return nil, fmt.Errorf("没有已发布的启用线路，请先绑定落地并发布")
	}
	result, err := remoteNodeCall(client, map[string]any{"operation": "probe", "layout": server.Node.Layout, "probes": probes})
	if err != nil {
		return nil, err
	}
	return result.Checks, nil
}

func (s *NodeService) Restart(serverID string) error {
	nodeOperationMu.Lock()
	defer nodeOperationMu.Unlock()
	server, err := s.servers.GetByIDForInstall(serverID)
	if err != nil {
		return err
	}
	if server.Node == nil {
		return fmt.Errorf("请先接管已有服务")
	}
	client, err := NewServerSSHClient(s.servers, server, nil)
	if err != nil {
		return err
	}
	defer client.Close()
	_, err = remoteNodeCall(client, map[string]any{"operation": "restart", "layout": server.Node.Layout})
	return err
}

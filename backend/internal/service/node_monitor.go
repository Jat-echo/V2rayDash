package service

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"net/url"
	"os"
	"strings"
)

//go:embed node_monitor.py
var nodeMonitorScript string

func (s *NodeService) InstallMonitor(serverID, publicURL, binaryPath string) error {
	nodeOperationMu.Lock()
	defer nodeOperationMu.Unlock()
	u, err := url.Parse(publicURL)
	if err != nil || u.Scheme != "https" || u.Hostname() == "" {
		return fmt.Errorf("请先在设置中填写节点可以访问的 HTTPS 控制中心地址")
	}
	server, err := s.servers.GetByIDForInstall(serverID)
	if err != nil {
		return err
	}
	if server.Node == nil {
		return fmt.Errorf("请先接管已有服务")
	}
	if server.Node.State == "publishing" {
		return fmt.Errorf("节点正在发布，请稍后接入监控")
	}
	binary, err := os.ReadFile(binaryPath)
	if err != nil {
		return fmt.Errorf("监控 Agent 构建产物不存在，请先配置 AGENT_BINARY_PATH")
	}
	if len(binary) < 20 || !bytes.Equal(binary[:4], []byte{0x7f, 'E', 'L', 'F'}) || binary[18] != 62 {
		return fmt.Errorf("需要 Linux amd64 Agent 构建产物")
	}
	client, err := NewServerSSHClient(s.servers, server, nil)
	if err != nil {
		return err
	}
	defer client.Close()
	remotePath := "/tmp/v2raydash-monitor-" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if err = client.UploadFileExec(binaryPath, remotePath); err != nil {
		return err
	}
	secret := make([]byte, 32)
	if _, err = rand.Read(secret); err != nil {
		return err
	}
	token := hex.EncodeToString(secret)
	tokenHash := sha256.Sum256([]byte(token))
	previous, _ := s.servers.AgentTokenHash(serverID)
	if err = s.servers.SetAgentTokenHash(serverID, hex.EncodeToString(tokenHash[:])); err != nil {
		return err
	}
	service := map[string]string{"legacy": "xray", "relay": "xray-relay", "exit": "xray-exit"}[server.Node.Layout]
	checksum := sha256.Sum256(binary)
	payload, _ := json.Marshal(map[string]any{"source": remotePath, "sha256": hex.EncodeToString(checksum[:]), "server_id": serverID, "url": strings.TrimRight(publicURL, "/"), "psk": token, "core_service": service})
	var out, stderr bytes.Buffer
	command := "sudo -n python3 -c '" + strings.ReplaceAll(nodeMonitorScript, "'", "'\"'\"'") + "'"
	if err = client.ExecuteInput(command, bytes.NewReader(payload), &out, &stderr); err != nil {
		_ = s.servers.SetAgentTokenHash(serverID, previous)
		return fmt.Errorf("监控安装失败，请检查节点架构与服务权限；未重启代理服务")
	}
	return nil
}

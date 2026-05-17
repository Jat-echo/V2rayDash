package service

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"v2ray-dash/backend/internal/model"
	"v2ray-dash/backend/internal/repository"
	"v2ray-dash/backend/internal/ssh"
)

type AccountService struct {
	accountRepo *repository.AccountRepository
	serverRepo  *repository.ServerRepository
}

func NewAccountService(accountRepo *repository.AccountRepository, serverRepo *repository.ServerRepository) *AccountService {
	return &AccountService{
		accountRepo: accountRepo,
		serverRepo:  serverRepo,
	}
}

// GetAccountLink 生成单个账号的订阅链接
func (s *AccountService) GetAccountLink(account *model.Account, serverIP string, subType string) string {
	var link string
	switch subType {
	case "vless":
		link = fmt.Sprintf("vless://%s@%s:443?encryption=none&flow=xtls-rprx-vision&security=tls&sni=%s#%s",
			account.UUID, serverIP, serverIP, account.Email)
	case "clash_meta":
		// Placeholder link for Clash Meta - not used for subscription generation
		link = fmt.Sprintf("clash://%s@%s:7890", account.UUID, serverIP)
	default:
		link = fmt.Sprintf("vless://%s@%s:443", account.UUID, serverIP)
	}
	return link
}

// GenerateVLESSSubscription 生成 VLESS 订阅内容
func (s *AccountService) GenerateVLESSSubscription(accounts []*model.Account, serverIP string) string {
	var lines []string
	for _, acc := range accounts {
		if !acc.Enabled {
			continue
		}
		// Generate standard VLESS URIs without reality transformation
		link := s.GetAccountLink(acc, serverIP, "vless")
		lines = append(lines, link)
	}
	return strings.Join(lines, "\n")
}

// GenerateClashMetaSubscription 生成 Clash.Meta 订阅内容
func (s *AccountService) GenerateClashMetaSubscription(accounts []*model.Account, serverIP string) (string, error) {
	proxies := make([]map[string]interface{}, 0)
	for _, acc := range accounts {
		if !acc.Enabled {
			continue
		}
		proxy := map[string]interface{}{
			"name": acc.Email,
			"type": "vless",
			"server": serverIP,
			"port": 443,
			"uuid": acc.UUID,
			"flow": "xtls-rprx-vision",
			"tls": map[string]interface{}{
				"enabled": true,
				"serverName": serverIP,
			},
		}
		proxies = append(proxies, proxy)
	}

	config := map[string]interface{}{
		"proxies": proxies,
	}
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// SyncToRemote 同步账号到远程服务器
func (s *AccountService) SyncToRemote(accountID string, auth ssh.SSHAuth) error {
	account, err := s.accountRepo.GetByID(accountID)
	if err != nil {
		return err
	}
	if account == nil {
		return fmt.Errorf("account not found: %s", accountID)
	}

	server, err := s.serverRepo.GetByID(account.ServerID)
	if err != nil {
		return err
	}
	if server == nil {
		return fmt.Errorf("server not found: %s", account.ServerID)
	}

	client, err := ssh.NewSSHClient(server.IP, server.SSHPort, server.SSHUser, auth)
	if err != nil {
		return err
	}
	defer client.Close()

	// Create SFTP client for file upload
	sftpClient, err := ssh.NewSFTPClient(client)
	if err != nil {
		return err
	}
	defer sftpClient.Close()

	// 生成配置文件内容
	config := map[string]interface{}{
		"log": map[string]interface{}{
			"loglevel": "warning",
		},
		"inbounds": []map[string]interface{}{
			{
				"port": 443,
				"protocol": "vless",
				"settings": map[string]interface{}{
					"clients": []map[string]interface{}{
						{
							"id": account.UUID,
							"email": account.Email,
						},
					},
				},
			},
		},
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write config to temp file and upload via SFTP
	tmpFile := "/tmp/v2ray_config_" + accountID + ".json"
	err = os.WriteFile(tmpFile, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write temp config: %w", err)
	}
	defer os.Remove(tmpFile)

	return sftpClient.UploadFile(tmpFile, "/etc/v2ray-agent/xray/conf/02_VLESS_TCP_inbounds.json")
}

// ImportFromRemote 从远程服务器导入账号
func (s *AccountService) ImportFromRemote(serverID string, auth ssh.SSHAuth) ([]*model.Account, error) {
	server, err := s.serverRepo.GetByID(serverID)
	if err != nil {
		return nil, err
	}

	client, err := ssh.NewSSHClient(server.IP, server.SSHPort, server.SSHUser, auth)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	// 读取 Xray 配置文件
	content, err := client.ReadRemoteFile("/etc/v2ray-agent/xray/conf/02_VLESS_TCP_inbounds.json")
	if err != nil {
		return nil, fmt.Errorf("failed to read remote config: %w", err)
	}

	// 解析 JSON 提取 users
	var config struct {
		Inbounds []struct {
			Settings struct {
				Clients []struct {
					ID    string `json:"id"`
					Email string `json:"email"`
				} `json:"clients"`
			} `json:"settings"`
		} `json:"inbounds"`
	}

	if err := json.Unmarshal([]byte(content), &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	var accounts []*model.Account
	var failed int
	for _, inbound := range config.Inbounds {
		for _, client := range inbound.Settings.Clients {
			account, err := s.accountRepo.Create(&model.CreateAccountRequest{
				ServerID:  serverID,
				UUID:      client.ID,
				Email:     client.Email,
				Protocols: []string{"vless_tcp"},
			})
			if err != nil {
				failed++
				continue
			}
			accounts = append(accounts, account)
		}
	}

	if failed > 0 {
		return accounts, fmt.Errorf("failed to import %d account(s)", failed)
	}

	return accounts, nil
}
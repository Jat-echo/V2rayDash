package service

import (
	"encoding/json"
	"fmt"
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
		link = fmt.Sprintf("clash://%s@%s:443", account.UUID, serverIP)
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
		for _, proto := range acc.Protocols {
			link := s.GetAccountLink(acc, serverIP, "vless")
			if strings.Contains(proto, "reality") {
				link = strings.Replace(link, "tls", "reality", 1)
			}
			lines = append(lines, link)
		}
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
		for _, proto := range acc.Protocols {
			proxy := map[string]interface{}{
				"name": acc.Email,
				"type": "vless",
				"server": serverIP,
				"port": 443,
				"uuid": acc.UUID,
				"flow": "xtls-rprx-vision",
				"tls": true,
			}
			if strings.Contains(proto, "reality") {
				proxy["tls"] = map[string]interface{}{
					"enabled": true,
					"serverName": serverIP,
					"reality": map[string]interface{}{
						"enabled": true,
					},
				}
			}
			proxies = append(proxies, proxy)
		}
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

	server, err := s.serverRepo.GetByID(account.ServerID)
	if err != nil {
		return err
	}

	client, err := ssh.NewSSHClient(server.IP, server.SSHPort, server.SSHUser, auth)
	if err != nil {
		return err
	}
	defer client.Close()

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

	data, _ := json.MarshalIndent(config, "", "  ")
	return client.UploadConfig("/etc/v2ray-agent/xray/conf/02_VLESS_TCP_inbounds.json", string(data))
}
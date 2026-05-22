package service

import (
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"

	"v2ray-dash/backend/internal/ssh"
)

type InstallResult struct {
	Success       bool
	Error         string
	RealityConfig *RealityConfig // Reality配置（如果启用了Reality协议）
	RealityPort   int            // Reality端口
}

type InstallConfig struct {
	Core       string // "xray-core" or "sing-box"
	UUID       string
	ServerName string // Reality SNI
	Protocols  []string
}

type Installer struct {
	serverID   string
	host       string
	port       int
	user       string
	auth       ssh.SSHAuth
	scriptPath string
	timeout    time.Duration
}

func NewInstaller(serverID, host string, port int, user string, auth ssh.SSHAuth, scriptPath string) *Installer {
	return &Installer{
		serverID:   serverID,
		host:       host,
		port:       port,
		user:       user,
		auth:       auth,
		scriptPath: scriptPath,
		timeout:    10 * time.Minute,
	}
}

func (i *Installer) Install(output io.Writer, config *InstallConfig) *InstallResult {
	// 1. 连接 SSH
	fmt.Fprintf(output, "[%s] 正在连接到 %s:%d...\n", time.Now().Format("15:04:05"), i.host, i.port)
	sshClient, err := ssh.NewSSHClient(i.host, i.port, i.user, i.auth)
	if err != nil {
		return &InstallResult{Success: false, Error: fmt.Sprintf("SSH连接失败: %v", err)}
	}
	defer sshClient.Close()
	fmt.Fprintf(output, "[%s] SSH连接成功\n", time.Now().Format("15:04:05"))

	// 2. 上传脚本
	fmt.Fprintf(output, "[%s] 正在上传安装脚本...\n", time.Now().Format("15:04:05"))
	sftpClient, err := ssh.NewSFTPClient(sshClient)
	if err != nil {
		return &InstallResult{Success: false, Error: fmt.Sprintf("SFTP连接失败: %v", err)}
	}
	defer func() {
		if sftpClient != nil {
			sftpClient.Close()
		}
	}()

	remotePath := "/tmp/v2ray_install.sh"
	if err := sftpClient.UploadFile(i.scriptPath, remotePath); err != nil {
		return &InstallResult{Success: false, Error: fmt.Sprintf("上传脚本失败: %v", err)}
	}
	fmt.Fprintf(output, "[%s] 脚本上传成功\n", time.Now().Format("15:04:05"))

	// 3. 构建安装命令参数
	var args []string

	// 核心
	if config.Core == "sing-box" {
		args = append(args, "--core sing-box")
	} else {
		args = append(args, "--core xray")
	}

	// 协议 - 只取第一个协议类型
	if len(config.Protocols) > 0 {
		protocol := config.Protocols[0]
		args = append(args, fmt.Sprintf("--protocol %s", protocol))
	}

	// UUID
	if config.UUID != "" {
		args = append(args, fmt.Sprintf("--uuid %s", config.UUID))
	}

	// Reality 服务器名称
	if config.ServerName != "" {
		args = append(args, fmt.Sprintf("--server-name %s", config.ServerName))
	}

	// 4. 执行安装 - 发送参数和菜单选择
	fmt.Fprintf(output, "[%s] 正在执行安装脚本...\n", time.Now().Format("15:04:05"))
	fmt.Fprintf(output, "[%s] 参数: %s\n", time.Now().Format("15:04:05"), strings.Join(args, " "))

	// 构建命令：传递参数 + 菜单选择（主菜单选择3=一键无域名Reality + 核心选择1=Xray）
	cmd := fmt.Sprintf("chmod +x %s && printf '3\\n1\\n' | bash %s %s", remotePath, remotePath, strings.Join(args, " "))

	// 捕获安装输出以便后续解析 publicKey 和端口
	var installOutput strings.Builder
	if err := sshClient.Execute(cmd, &installOutput, &installOutput); err != nil {
		return &InstallResult{Success: false, Error: fmt.Sprintf("安装执行失败: %v", err)}
	}

	// 解析安装输出中的端口（格式：---> 端口: 21313）
	realityPort := 443 // 默认端口
	portRe := regexp.MustCompile(`--->\s*端口[:\s]*(\d+)`)
	if matches := portRe.FindStringSubmatch(installOutput.String()); len(matches) >= 2 {
		if port, err := strconv.Atoi(matches[1]); err == nil {
			realityPort = port
		}
	}

	fmt.Fprintf(output, "\n[%s] ✓ 安装完成，正在获取Reality配置...\n", time.Now().Format("15:04:05"))

	// 5. 获取 Reality 配置（传入安装输出用于解析 publicKey）
	realityConfig, err := i.fetchRealityConfig(sshClient, config.Core, config.ServerName, installOutput.String())
	if err != nil {
		fmt.Fprintf(output, "[%s] 警告: 获取Reality配置失败: %v\n", time.Now().Format("15:04:05"), err)
	}

	if realityConfig != nil {
		fmt.Fprintf(output, "[%s] ✓ Reality配置获取成功\n", time.Now().Format("15:04:05"))
		fmt.Fprintf(output, "[%s]   ServerName: %s\n", time.Now().Format("15:04:05"), realityConfig.ServerName)
		fmt.Fprintf(output, "[%s]   PublicKey: %s\n", time.Now().Format("15:04:05"), realityConfig.PublicKey)
		fmt.Fprintf(output, "[%s]   Port: %d\n", time.Now().Format("15:04:05"), realityPort)
	}

	return &InstallResult{
		Success:       true,
		RealityConfig: realityConfig,
		RealityPort:   realityPort,
	}
}

// cleanANSICodes 清理 ANSI 转义序列
func cleanANSICodes(s string) string {
	// 移除 ANSI 颜色转义序列
	s = strings.ReplaceAll(s, "\x1B[0m", "")
	s = strings.ReplaceAll(s, "\x1B[1;36m", "")
	s = strings.ReplaceAll(s, "\x1B[32m", "")
	s = strings.ReplaceAll(s, "\x1B[33m", "")
	s = strings.ReplaceAll(s, "\x1B[31m", "")
	s = strings.ReplaceAll(s, "\x1B[1;31m", "")
	s = strings.ReplaceAll(s, "\x1B[1;32m", "")
	s = strings.ReplaceAll(s, "\x1B[1;33m", "")
	s = strings.ReplaceAll(s, "\x1B[1;36m", "")
	s = strings.ReplaceAll(s, "\x1B[36m", "")
	s = strings.ReplaceAll(s, "\x1B[34m", "")
	s = strings.ReplaceAll(s, "\x1B[35m", "")
	s = strings.ReplaceAll(s, "\x1B[1m", "")
	s = strings.ReplaceAll(s, "\x1B[0m", "")
	// 清理常见的残留字符
	s = strings.TrimSpace(s)
	return s
}

// fetchRealityConfig 从远程服务器获取 Reality 配置
func (i *Installer) fetchRealityConfig(client *ssh.SSHClient, core, serverName string, installOutput string) (*RealityConfig, error) {
	var publicKey, realityServerName string

	// 优先从安装输出中解析 publicKey 和 serverName
	// 格式: publicKey:0Jbzi50zkYokBiMQkUgC9eT40SpYfVDXAWt-kWUZVHE
	// 格式: ---> 客户端可用域名: m.media-amazon.com:443
	if installOutput != "" {
		// 解析 publicKey
		re := regexp.MustCompile(`publicKey[:\s]*(.+)`)
		matches := re.FindStringSubmatch(installOutput)
		if len(matches) >= 2 {
			publicKey = cleanANSICodes(strings.TrimSpace(matches[1]))
		}

		// 解析 serverName（从安装输出中的客户端可用域名）
		sniRe := regexp.MustCompile(`--->\s*客户端可用域名[:\s]*([^\s:]+)`)
		sniMatches := sniRe.FindStringSubmatch(installOutput)
		if len(sniMatches) >= 2 {
			realityServerName = cleanANSICodes(strings.TrimSpace(sniMatches[1]))
		} else {
			// 如果解析不到，使用传入的默认值
			realityServerName = cleanANSICodes(serverName)
		}
	} else {
		realityServerName = cleanANSICodes(serverName)
	}

	// 如果从输出解析失败，尝试读取配置文件
	if publicKey == "" {
		// 优先读取 sing-box 的 reality_key 文件
		singBoxKeyPath := "/etc/v2ray-agent/sing-box/conf/config/reality_key"
		content, err := client.ReadRemoteFile(singBoxKeyPath)
		if err == nil && len(content) > 0 {
			re := regexp.MustCompile(`publicKey[:\s]*(.+)`)
			matches := re.FindStringSubmatch(content)
			if len(matches) >= 2 {
				publicKey = cleanANSICodes(strings.TrimSpace(matches[1]))
				realityServerName = cleanANSICodes(serverName)
			}
		}
	}

	// 如果 sing-box 没有，尝试读取 xray-core 的配置
	if publicKey == "" {
		// 尝试多个可能的配置文件路径
		xrayPaths := []string{
			"/etc/v2ray-agent/xray/conf/07_VLESS_vision_reality_inbounds.json",
			"/etc/v2ray-agent/xray/conf/07_VLESS_reality_vision_inbounds.json",
			"/etc/v2ray-agent/xray/conf/07_VLESS_vision_reality_inbounds.toml",
			"/etc/v2ray-agent/xray/conf/07_VLESS_vision_reality_inbounds",
			"/etc/v2ray-agent/xray/conf/07_VLESS_TCP_inbounds.json",
			"/etc/v2ray-agent/xray/conf/vLESS_vision_reality_inbounds.json",
		}
		for _, xrayPath := range xrayPaths {
			content, err := client.ReadRemoteFile(xrayPath)
			if err != nil {
				continue
			}
			// 尝试解析JSON格式
			var config struct {
				Inbounds []struct {
					StreamSettings struct {
						RealitySettings struct {
							PublicKey   string   `json:"publicKey"`
							ServerNames []string `json:"serverNames"`
						} `json:"realitySettings"`
					} `json:"streamSettings"`
				} `json:"inbounds"`
			}
			if err := json.Unmarshal([]byte(content), &config); err != nil {
				continue
			}
			if len(config.Inbounds) > 0 {
				publicKey = config.Inbounds[0].StreamSettings.RealitySettings.PublicKey
				if len(config.Inbounds[0].StreamSettings.RealitySettings.ServerNames) > 0 {
					realityServerName = config.Inbounds[0].StreamSettings.RealitySettings.ServerNames[0]
				}
				if publicKey != "" {
					break
				}
			}
		}
	}

	if publicKey == "" {
		return nil, fmt.Errorf("未能提取到publicKey")
	}

	return &RealityConfig{
		Enabled:    true,
		ServerName: realityServerName,
		PublicKey:  publicKey,
	}, nil
}
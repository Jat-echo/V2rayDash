package service

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"v2ray-dash/backend/internal/ssh"
)

type InstallResult struct {
	Success       bool
	Error         string
	GeneratedUUID string            // 安装脚本生成的UUID
	RealityConfig *RealityConfig    // Reality配置（如果启用了Reality协议）
	RealityPort   int               // Reality端口
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

	// 4. 执行安装 - 使用 script 命令模拟终端以支持交互式菜单
	fmt.Fprintf(output, "[%s] 正在执行安装脚本...\n", time.Now().Format("15:04:05"))
	fmt.Fprintf(output, "[%s] 参数: %s\n", time.Now().Format("15:04:05"), strings.Join(args, " "))

	// 使用 script 命令创建一个伪终端，这样脚本中的 read 命令就能正常工作
	cmd := fmt.Sprintf("chmod +x %s && script -q -c 'printf \"3\\n1\\n\" | bash %s %s' /dev/null", remotePath, remotePath, strings.Join(args, " "))

	// 捕获安装输出以便后续解析 publicKey 和端口
	var installOutput strings.Builder
	if err := sshClient.Execute(cmd, &installOutput, &installOutput); err != nil {
		return &InstallResult{Success: false, Error: fmt.Sprintf("安装执行失败: %v", err)}
	}

	// 解析安装输出
	cleanedOutput := cleanANSICodes(installOutput.String())

	// 解析安装输出中的 UUID（格式：---> UUID: xxxxxxxx-xxxx-xxxx 或 ---> UUID:xxxxxxxx-xxxx-xxxx）
	generatedUUID := ""
	uuidRe := regexp.MustCompile(`--->\s*UUID[:\s]*([a-fA-F0-9-]+)`)
	if matches := uuidRe.FindStringSubmatch(cleanedOutput); len(matches) >= 2 {
		generatedUUID = strings.TrimSpace(matches[1])
	}

	// 解析安装输出中的端口（支持多种格式：---> 端口: 21313 或 ---> xxx端口：21313）
	realityPort := 443 // 默认端口
	portRe := regexp.MustCompile(`--->.*端口[：:]\s*(\d+)`)
	if matches := portRe.FindStringSubmatch(cleanedOutput); len(matches) >= 2 {
		if port, err := strconv.Atoi(matches[1]); err == nil {
			realityPort = port
		}
	}

	// 备用：直接从数字中提取端口（用于 ---> VLESS_Reality_Vision端口：21313 这种格式）
	if realityPort == 443 {
		portRe2 := regexp.MustCompile(`端口[：:]\s*(\d+)`)
		if matches := portRe2.FindStringSubmatch(cleanedOutput); len(matches) >= 2 {
			if port, err := strconv.Atoi(matches[1]); err == nil && port > 0 && port < 65536 {
				realityPort = port
			}
		}
	}

	fmt.Fprintf(output, "\n[%s] ✓ 安装完成，正在获取Reality配置...\n", time.Now().Format("15:04:05"))

	// 5. 获取 Reality 配置
	realityConfig, err := i.fetchRealityConfig(sshClient, config.Core, config.ServerName, cleanedOutput)
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
		GeneratedUUID:  generatedUUID,
		RealityConfig:   realityConfig,
		RealityPort:     realityPort,
	}
}

// InstallStreaming 执行安装并实时流式输出到 HTTP 响应
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

// InstallStreaming 执行安装并实时流式输出到 HTTP 响应
func (i *Installer) InstallStreaming(flusher http.Flusher, config *InstallConfig) *InstallResult {
	output := &streamingWriter{flusher: flusher}

	// 1. 连接 SSH
	fmt.Fprintf(output, "[%s] 正在连接到 %s:%d...\n", time.Now().Format("15:04:05"), i.host, i.port)
	sshClient, err := ssh.NewSSHClient(i.host, i.port, i.user, i.auth)
	if err != nil {
		fmt.Fprintf(output, "[ERROR] SSH连接失败: %v\n", err)
		return &InstallResult{Success: false, Error: fmt.Sprintf("SSH连接失败: %v", err)}
	}
	defer sshClient.Close()
	fmt.Fprintf(output, "[%s] SSH连接成功\n", time.Now().Format("15:04:05"))

	// 2. 上传脚本
	fmt.Fprintf(output, "[%s] 正在上传安装脚本...\n", time.Now().Format("15:04:05"))
	sftpClient, err := ssh.NewSFTPClient(sshClient)
	if err != nil {
		fmt.Fprintf(output, "[ERROR] SFTP连接失败: %v\n", err)
		return &InstallResult{Success: false, Error: fmt.Sprintf("SFTP连接失败: %v", err)}
	}
	defer func() {
		if sftpClient != nil {
			sftpClient.Close()
		}
	}()

	remotePath := "/tmp/v2ray_install.sh"
	if err := sftpClient.UploadFile(i.scriptPath, remotePath); err != nil {
		fmt.Fprintf(output, "[ERROR] 上传脚本失败: %v\n", err)
		return &InstallResult{Success: false, Error: fmt.Sprintf("上传脚本失败: %v", err)}
	}
	fmt.Fprintf(output, "[%s] 脚本上传成功\n", time.Now().Format("15:04:05"))

	// 3. 构建安装命令参数
	var args []string
	if config.Core == "sing-box" {
		args = append(args, "--core sing-box")
	} else {
		args = append(args, "--core xray")
	}
	if len(config.Protocols) > 0 {
		args = append(args, fmt.Sprintf("--protocol %s", config.Protocols[0]))
	}
	if config.UUID != "" {
		args = append(args, fmt.Sprintf("--uuid %s", config.UUID))
	}
	if config.ServerName != "" {
		args = append(args, fmt.Sprintf("--server-name %s", config.ServerName))
	}

	// 4. 执行安装
	fmt.Fprintf(output, "[%s] 正在执行安装脚本...\n", time.Now().Format("15:04:05"))
	fmt.Fprintf(output, "[%s] 参数: %s\n", time.Now().Format("15:04:05"), strings.Join(args, " "))

	cmd := fmt.Sprintf("chmod +x %s && script -q -c 'printf \"3\\n1\\n\" | bash %s %s' /dev/null", remotePath, remotePath, strings.Join(args, " "))

	done, err := sshClient.ExecuteStreamingWithFlush(cmd, output, flusher)
	if err != nil {
		fmt.Fprintf(output, "[ERROR] 启动安装命令失败: %v\n", err)
		return &InstallResult{Success: false, Error: fmt.Sprintf("安装执行失败: %v", err)}
	}
	<-done

	cleanedOutput := cleanANSICodes(output.String())

	// 解析 UUID
	generatedUUID := ""
	uuidRe := regexp.MustCompile(`--->\s*UUID[:\s]*([a-fA-F0-9-]+)`)
	if matches := uuidRe.FindStringSubmatch(cleanedOutput); len(matches) >= 2 {
		generatedUUID = strings.TrimSpace(matches[1])
	}

	// 解析端口
	realityPort := 443
	portRe := regexp.MustCompile(`--->\s*端口[:\s]*(\d+)`)
	if matches := portRe.FindStringSubmatch(cleanedOutput); len(matches) >= 2 {
		if port, err := strconv.Atoi(matches[1]); err == nil {
			realityPort = port
		}
	}

	fmt.Fprintf(output, "\n[%s] ✓ 安装完成，正在获取Reality配置...\n", time.Now().Format("15:04:05"))

	realityConfig, err := i.fetchRealityConfig(sshClient, config.Core, config.ServerName, cleanedOutput)
	if err != nil {
		fmt.Fprintf(output, "[WARN] 获取Reality配置失败: %v\n", err)
	}

	if realityConfig != nil {
		fmt.Fprintf(output, "[%s] ✓ Reality配置获取成功\n", time.Now().Format("15:04:05"))
		fmt.Fprintf(output, "[%s]   ServerName: %s\n", time.Now().Format("15:04:05"), realityConfig.ServerName)
		fmt.Fprintf(output, "[%s]   PublicKey: %s\n", time.Now().Format("15:04:05"), realityConfig.PublicKey)
		fmt.Fprintf(output, "[%s]   Port: %d\n", time.Now().Format("15:04:05"), realityPort)
	}

	return &InstallResult{
		Success:       true,
		GeneratedUUID: generatedUUID,
		RealityConfig: realityConfig,
		RealityPort:   realityPort,
	}
}

type streamingWriter struct {
	buffer  strings.Builder
	flusher http.Flusher
}

func (w *streamingWriter) Write(p []byte) (int, error) {
	n, err := w.buffer.Write(p)
	if w.flusher != nil {
		w.flusher.Flush()
	}
	return n, err
}

func (w *streamingWriter) String() string {
	return w.buffer.String()
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
		singBoxKeyPaths := []string{
			"/etc/v2ray-agent/sing-box/conf/config/reality_key",
			"/etc/v2ray-agent/sing-box/config/reality_key",
		}
		for _, singBoxKeyPath := range singBoxKeyPaths {
			content, err := client.ReadRemoteFile(singBoxKeyPath)
			if err == nil && len(content) > 0 {
				// 直接提取 publicKey:xxx 格式
				re := regexp.MustCompile(`publicKey[:\s]*(.+)`)
				matches := re.FindStringSubmatch(content)
				if len(matches) >= 2 {
					publicKey = cleanANSICodes(strings.TrimSpace(matches[1]))
					if publicKey != "" {
						realityServerName = cleanANSICodes(serverName)
						break
					}
				}
				// 如果不是 publicKey: 格式，尝试直接使用整行作为 publicKey
				trimmed := strings.TrimSpace(content)
				if len(trimmed) > 20 { // publicKey 一般比较长
					publicKey = trimmed
					realityServerName = cleanANSICodes(serverName)
					break
				}
			}
		}
	}

	// 如果 sing-box 没有，尝试读取 xray-core 的配置
	if publicKey == "" {
		// 尝试多个可能的配置文件路径
		xrayPaths := []string{
			"/etc/v2ray-agent/xray/conf/07_VLESS_vision_reality_inbounds.json",
			"/etc/v2ray-agent/xray/conf/07_VLESS_reality_vision_inbounds.json",
			"/etc/v2ray-agent/xray/conf/07_VLESS_vision_reality_inbounds",
			"/etc/v2ray-agent/xray/conf/07_VLESS_TCP_inbounds.json",
			"/etc/v2ray-agent/xray/conf/vLESS_vision_reality_inbounds.json",
			"/etc/v2ray-agent/xray/config.json",
		}
		for _, xrayPath := range xrayPaths {
			content, err := client.ReadRemoteFile(xrayPath)
			if err != nil {
				continue
			}
			// 尝试解析JSON格式 - 从 realitySettings.publicKey 提取
			re := regexp.MustCompile(`"publicKey"\s*:\s*"([^"]+)"`)
			matches := re.FindStringSubmatch(content)
			if len(matches) >= 2 {
				publicKey = matches[1]
				// 同时提取 serverNames
				sniRe := regexp.MustCompile(`"serverNames"\s*:\s*\[\s*"([^"]+)"`)
				sniMatches := sniRe.FindStringSubmatch(content)
				if len(sniMatches) >= 2 {
					realityServerName = sniMatches[1]
				}
				break
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
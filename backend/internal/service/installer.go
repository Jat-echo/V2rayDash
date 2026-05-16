package service

import (
	"fmt"
	"io"
	"time"

	"v2ray-dash/backend/internal/ssh"
)

type InstallResult struct {
	Success bool
	Error   string
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

func (i *Installer) Install(output io.Writer) *InstallResult {
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

	// 3. 执行安装
	fmt.Fprintf(output, "[%s] 正在执行安装脚本...\n", time.Now().Format("15:04:05"))
	cmd := fmt.Sprintf("chmod +x %s && bash %s", remotePath, remotePath)
	if err := sshClient.Execute(cmd, output, output); err != nil {
		return &InstallResult{Success: false, Error: fmt.Sprintf("安装执行失败: %v", err)}
	}

	fmt.Fprintf(output, "\n[%s] ✓ 安装完成\n", time.Now().Format("15:04:05"))
	return &InstallResult{Success: true}
}
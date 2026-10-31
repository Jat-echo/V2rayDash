package service

import (
	"fmt"
	"v2ray-dash/backend/internal/model"
	"v2ray-dash/backend/internal/repository"
	"v2ray-dash/backend/internal/ssh"
)

func serverSSHAuth(server *model.Server) ssh.SSHAuth {
	if server.SSHKeyType == "password" {
		return &ssh.PasswordAuth{Password: server.SSHPassword}
	}
	return &ssh.KeyAuth{PrivateKey: server.SSHKey}
}

// NewServerSSHClient is shared by configuration, monitoring and legacy operations.
func NewServerSSHClient(servers *repository.ServerRepository, server *model.Server, auth ssh.SSHAuth) (*ssh.SSHClient, error) {
	if auth == nil {
		auth = serverSSHAuth(server)
	}
	key := ""
	if server.Node != nil {
		key = server.Node.SSHHostKey
	}
	var jump *ssh.JumpHost
	if server.SSHJumpServerID != "" {
		bridge, err := servers.GetByIDForInstall(server.SSHJumpServerID)
		if err != nil {
			return nil, fmt.Errorf("SSH 跳板节点不存在")
		}
		if bridge.ID == server.ID || bridge.Role != "relay" || bridge.SSHJumpServerID != "" || bridge.Node == nil || bridge.Node.SSHHostKey == "" {
			return nil, fmt.Errorf("请选择已读取配置且可直接连接的中转节点作为 SSH 跳板")
		}
		jump = &ssh.JumpHost{Host: bridge.IP, Port: bridge.SSHPort, User: bridge.SSHUser, Auth: serverSSHAuth(bridge), HostKey: bridge.Node.SSHHostKey}
	}
	return ssh.NewTrustedSSHClientViaJump(server.IP, server.SSHPort, server.SSHUser, auth, key, jump)
}

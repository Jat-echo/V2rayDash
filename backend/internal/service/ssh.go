package service

import (
	"bytes"
	"fmt"
	"net"
	"os"

	"golang.org/x/crypto/ssh"
)

type SSHService struct{}

func NewSSHService() *SSHService {
	return &SSHService{}
}

type SSHResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

func (s *SSHService) Connect(host string, port int, user, privateKey string) (*ssh.Client, error) {
	key, err := ssh.ParsePrivateKey([]byte(privateKey))
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.PublicAuth(key),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	addr := fmt.Sprintf("%s:%d", host, port)
	conn, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}

	return conn, nil
}

func (s *SSHService) Execute(client *ssh.Client, command string) (*SSHResult, error) {
	session, err := client.NewSession()
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Close()

	var stdout, stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr

	err = session.Run(command)
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*ssh.ExitError); ok {
			exitCode = exitErr.ExitStatus()
		}
	}

	return &SSHResult{
		Stdout:  stdout.String(),
		Stderr:  stderr.String(),
		ExitCode: exitCode,
	}, nil
}

func (s *SSHService) ExecuteWithPassword(host string, port int, user, password, command string) (*SSHResult, error) {
	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	addr := fmt.Sprintf("%s:%d", host, port)
	conn, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	session, err := conn.NewSession()
	if err != nil {
		return nil, err
	}
	defer session.Close()

	var stdout, stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr

	err = session.Run(command)
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*ssh.ExitError); ok {
			exitCode = exitErr.ExitStatus()
		}
	}

	return &SSHResult{
		Stdout:  stdout.String(),
		Stderr:  stderr.String(),
		ExitCode: exitCode,
	}, nil
}

func (s *SSHService) InstallV2ray(client *ssh.Client, controlCenterURL string) (*SSHResult, error) {
	installCmd := fmt.Sprintf(
		"curl -sL https://raw.githubusercontent.com/your-repo/install.sh | bash -s -- --agent %s",
		controlCenterURL,
	)
	return s.Execute(client, installCmd)
}

// ReadPrivateKeyFromFile 从文件读取私钥
func ReadPrivateKeyFromFile(path string) (string, error) {
	keyBytes, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(keyBytes), nil
}

// GetLocalIP 获取本机 IP
func GetLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return ""
}
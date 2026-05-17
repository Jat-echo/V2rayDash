package ssh

import (
	"fmt"
	"io"
	"time"

	"golang.org/x/crypto/ssh"
)

type SSHAuth interface {
	AuthMethod() ssh.AuthMethod
}

// KeyAuth implements SSHAuth for private key authentication
type KeyAuth struct {
	PrivateKey string
	Passphrase string
}

func (a *KeyAuth) AuthMethod() ssh.AuthMethod {
	signer, err := ssh.ParsePrivateKeyWithPassphrase([]byte(a.PrivateKey), []byte(a.Passphrase))
	if err != nil {
		// Try without passphrase
		signer, err = ssh.ParsePrivateKey([]byte(a.PrivateKey))
		if err != nil {
			// WARNING: Returns nil if both passphrase and non-passphrase parsing fail.
			// The caller should validate the auth method before use.
			return nil
		}
	}
	return ssh.PublicKeys(signer)
}

// PasswordAuth implements SSHAuth for password authentication
type PasswordAuth struct {
	Password string
}

func (a *PasswordAuth) AuthMethod() ssh.AuthMethod {
	if a.Password == "" {
		return nil
	}
	return ssh.Password(a.Password)
}

// SSHClient wraps an SSH client connection
type SSHClient struct {
	client *ssh.Client
}

// NewSSHClient creates a new SSH client connection
func NewSSHClient(host string, port int, user string, auth SSHAuth) (*SSHClient, error) {
	authMethod := auth.AuthMethod()
	if authMethod == nil {
		return nil, fmt.Errorf("ssh authentication method is invalid (check credentials)")
	}

	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{authMethod},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout: 10 * time.Second,
	}

	addr := fmt.Sprintf("%s:%d", host, port)
	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return nil, fmt.Errorf("ssh dial failed: %w", err)
	}

	return &SSHClient{client: client}, nil
}

// Execute runs a command and writes output to the provided writers
func (c *SSHClient) Execute(cmd string, stdout io.Writer, stderr io.Writer) error {
	session, err := c.client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	session.Stdout = stdout
	session.Stderr = stderr

	err = session.Run(cmd)
	if err != nil {
		return fmt.Errorf("command '%s' failed: %w", cmd, err)
	}
	return nil
}

// ReadRemoteFile reads a file from the remote server via SFTP
func (c *SSHClient) ReadRemoteFile(path string) (string, error) {
	sftpClient, err := NewSFTPClient(c)
	if err != nil {
		return "", fmt.Errorf("failed to create sftp client: %w", err)
	}
	defer sftpClient.Close()

	file, err := sftpClient.client.Open(path)
	if err != nil {
		return "", fmt.Errorf("failed to open remote file: %w", err)
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		return "", fmt.Errorf("failed to read remote file: %w", err)
	}

	return string(content), nil
}

// Close closes the SSH client connection
func (c *SSHClient) Close() error {
	return c.client.Close()
}
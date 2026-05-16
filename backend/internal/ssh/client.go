package ssh

import (
	"fmt"
	"io"
	"net"

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
	return ssh.Password(a.Password)
}

// SSHClient wraps an SSH client connection
type SSHClient struct {
	client *ssh.Client
}

// NewSSHClient creates a new SSH client connection
func NewSSHClient(host string, port int, user string, auth SSHAuth) (*SSHClient, error) {
	config := &ssh.ClientConfig{
		User:            user,
		Auth:            []ssh.AuthMethod{auth.AuthMethod()},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
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

	return session.Run(cmd)
}

// Close closes the SSH client connection
func (c *SSHClient) Close() error {
	return c.client.Close()
}
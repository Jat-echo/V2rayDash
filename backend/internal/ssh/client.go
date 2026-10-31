package ssh

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"sync"
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
	client  *ssh.Client
	HostKey string
	jump    *SSHClient
}

// JumpHost is one directly reachable SSH hop. Target authentication stays local.
type JumpHost struct {
	Host    string
	Port    int
	User    string
	Auth    SSHAuth
	HostKey string
}

// NewSSHClient creates a new SSH client connection
func NewSSHClient(host string, port int, user string, auth SSHAuth) (*SSHClient, error) {
	return NewTrustedSSHClient(host, port, user, auth, "")
}

func NewTrustedSSHClient(host string, port int, user string, auth SSHAuth, expectedKey string) (*SSHClient, error) {
	return NewTrustedSSHClientViaJump(host, port, user, auth, expectedKey, nil)
}

func NewTrustedSSHClientViaJump(host string, port int, user string, auth SSHAuth, expectedKey string, jump *JumpHost) (*SSHClient, error) {
	if auth == nil {
		return nil, fmt.Errorf("SSH credentials are missing")
	}
	authMethod := auth.AuthMethod()
	if authMethod == nil {
		return nil, fmt.Errorf("ssh authentication method is invalid (check credentials)")
	}

	var hostKey string
	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{authMethod},
		HostKeyCallback: func(_ string, _ net.Addr, key ssh.PublicKey) error {
			hostKey = base64.StdEncoding.EncodeToString(key.Marshal())
			if expectedKey != "" && expectedKey != hostKey {
				return fmt.Errorf("SSH host key changed; verify the server before adopting again")
			}
			return nil
		},
		Timeout: 60 * time.Second,
	}

	addr := net.JoinHostPort(host, fmt.Sprint(port))
	if jump != nil {
		bridge, err := NewTrustedSSHClient(jump.Host, jump.Port, jump.User, jump.Auth, jump.HostKey)
		if err != nil {
			return nil, fmt.Errorf("SSH 跳板连接失败: %w", err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		conn, err := bridge.client.DialContext(ctx, "tcp", addr)
		if err != nil {
			bridge.Close()
			return nil, fmt.Errorf("SSH 跳板无法连接目标服务器: %w", err)
		}
		deadline := time.AfterFunc(60*time.Second, func() { conn.Close() })
		transport, channels, requests, err := ssh.NewClientConn(conn, addr, config)
		deadline.Stop()
		if err != nil {
			conn.Close()
			bridge.Close()
			return nil, fmt.Errorf("目标服务器 SSH 认证失败: %w", err)
		}
		return &SSHClient{client: ssh.NewClient(transport, channels, requests), HostKey: hostKey, jump: bridge}, nil
	}
	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return nil, fmt.Errorf("ssh dial failed: %w", err)
	}

	return &SSHClient{client: client, HostKey: hostKey}, nil
}

// Execute runs a command and writes output to the provided writers.
// For streaming output, use ExecuteStreaming instead.
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

// ExecuteInput passes sensitive payloads over stdin instead of command arguments.
func (c *SSHClient) ExecuteInput(cmd string, input io.Reader, stdout, stderr io.Writer) error {
	session, err := c.client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()
	session.Stdin, session.Stdout, session.Stderr = input, stdout, stderr
	if err := session.Start(cmd); err != nil {
		return err
	}
	done := make(chan error, 1)
	go func() { done <- session.Wait() }()
	select {
	case err := <-done:
		return err
	case <-time.After(180 * time.Second):
		_ = session.Close()
		return fmt.Errorf("remote operation timed out")
	}
}

// ExecuteStreaming starts a command and streams output to writers in real-time.
// It returns a Done channel that signals when execution completes.
func (c *SSHClient) ExecuteStreaming(cmd string, stdout io.Writer, stderr io.Writer) (<-chan struct{}, error) {
	session, err := c.client.NewSession()
	if err != nil {
		return nil, err
	}

	// Create pipes for stdout and stderr
	stdoutPipe, err := session.StdoutPipe()
	if err != nil {
		session.Close()
		return nil, err
	}
	stderrPipe, err := session.StderrPipe()
	if err != nil {
		session.Close()
		return nil, err
	}

	done := make(chan struct{})

	// Start the command
	if err := session.Start(cmd); err != nil {
		session.Close()
		return nil, err
	}

	// Stream stdout to the provided writer
	go func() {
		io.Copy(stdout, stdoutPipe)
	}()

	// Stream stderr to the provided writer
	go func() {
		io.Copy(stderr, stderrPipe)
	}()

	// Wait for command completion
	go func() {
		session.Wait()
		session.Close()
		close(done)
	}()

	return done, nil
}

// ExecuteStreamingWithFlush is like ExecuteStreaming but calls Flush after each write.
// This is useful for SSE streaming responses. The returned channel sends the command's
// exit error (nil on success) when execution completes.
func (c *SSHClient) ExecuteStreamingWithFlush(cmd string, w io.Writer, flusher http.Flusher) (<-chan error, error) {
	session, err := c.client.NewSession()
	if err != nil {
		return nil, err
	}

	stdoutPipe, err := session.StdoutPipe()
	if err != nil {
		session.Close()
		return nil, err
	}
	stderrPipe, err := session.StderrPipe()
	if err != nil {
		session.Close()
		return nil, err
	}

	done := make(chan error, 1)

	if err := session.Start(cmd); err != nil {
		session.Close()
		return nil, err
	}

	// Create a writer that also flushes on each write
	flushingWriter := &flushingWriter{w: w, flusher: flusher}

	// Stream stdout
	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := stdoutPipe.Read(buf)
			if n > 0 {
				flushingWriter.Write(buf[:n])
				flusher.Flush()
			}
			if err != nil {
				break
			}
		}
	}()

	// Stream stderr
	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := stderrPipe.Read(buf)
			if n > 0 {
				flushingWriter.Write(buf[:n])
				flusher.Flush()
			}
			if err != nil {
				break
			}
		}
	}()

	go func() {
		err := session.Wait()
		session.Close()
		done <- err
	}()

	return done, nil
}

type flushingWriter struct {
	mu      sync.Mutex
	w       io.Writer
	flusher http.Flusher
}

func (fw *flushingWriter) Write(p []byte) (int, error) {
	fw.mu.Lock()
	defer fw.mu.Unlock()
	return fw.w.Write(p)
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

// UploadFileExec uploads a local file to the remote path.
// Strategy:
//  1. SFTP subsystem — pure binary transfer, no shell commands (avoids SIGALRM
//     or other exec-channel restrictions on hardened servers).
//  2. Chunked base64 via exec — fallback for servers without SFTP subsystem.
func (c *SSHClient) UploadFileExec(localPath, remotePath string) error {
	// --- attempt 1: SFTP ---
	// Close explicitly before falling through so the subsystem channel is freed
	// before the chunked exec loop opens its own sessions.
	if sftpClient, err := NewSFTPClient(c); err == nil {
		uploadErr := sftpClient.UploadFile(localPath, remotePath)
		sftpClient.Close()
		if uploadErr == nil {
			return c.Execute(fmt.Sprintf("chmod +x %s", remotePath), io.Discard, io.Discard)
		}
	}

	// --- attempt 2: chunked base64 exec ---
	content, err := os.ReadFile(localPath)
	if err != nil {
		return fmt.Errorf("failed to open local file: %w", err)
	}
	encoded := base64.StdEncoding.EncodeToString(content)

	// 32 768 chars of base64 ≈ 24 KB binary; multiple of 4 keeps block boundaries valid.
	const chunkSize = 32768
	for i := 0; i < len(encoded); i += chunkSize {
		end := i + chunkSize
		if end > len(encoded) {
			end = len(encoded)
		}
		chunk := encoded[i:end]
		redirect := ">>"
		if i == 0 {
			redirect = ">"
		}
		// base64 alphabet has no single quotes, so single-quoting is safe.
		cmd := fmt.Sprintf("printf '%%s' '%s' | base64 -d %s %s", chunk, redirect, remotePath)
		if err := c.Execute(cmd, io.Discard, io.Discard); err != nil {
			return fmt.Errorf("failed to upload file chunk at offset %d: %w", i, err)
		}
	}

	return c.Execute(fmt.Sprintf("chmod +x %s", remotePath), io.Discard, io.Discard)
}

// Close closes the SSH client connection
func (c *SSHClient) Close() error {
	err := c.client.Close()
	if c.jump != nil {
		c.jump.Close()
	}
	return err
}

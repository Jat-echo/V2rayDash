package ssh

import (
	"fmt"
	"io"
	"os"

	"github.com/pkg/sftp"
)

// SFTPClient wraps sftp client operations
type SFTPClient struct {
	client *sftp.Client
}

// NewSFTPClient creates a new SFTP client from an SSH connection
func NewSFTPClient(sshClient *SSHClient) (*SFTPClient, error) {
	sftpClient, err := sftp.NewClient(sshClient.client)
	if err != nil {
		return nil, fmt.Errorf("sftp client failed: %w", err)
	}
	return &SFTPClient{client: sftpClient}, nil
}

// UploadFile uploads a local file to a remote path
func (c *SFTPClient) UploadFile(localPath, remotePath string) error {
	file, err := os.Open(localPath)
	if err != nil {
		return err
	}
	defer file.Close()

	remoteFile, err := c.client.Create(remotePath)
	if err != nil {
		return err
	}
	defer remoteFile.Close()

	_, err = io.Copy(remoteFile, file)
	return err
}

// Close closes the SFTP client
func (c *SFTPClient) Close() error {
	return c.client.Close()
}
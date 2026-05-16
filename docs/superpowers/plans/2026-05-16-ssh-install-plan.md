# 远程服务器 SSH 自动安装 - 实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 实现从 Web 界面一键 SSH 连接远程服务器，自动上传 install.sh 并执行，通过 SSE 实时显示安装日志。

**Architecture:** 后端使用 golang.org/x/crypto/ssh 构建 SSH 客户端，通过 SFTP 上传脚本，HTTP SSE 流式推送安装输出到前端。

**Tech Stack:** Go (golang.org/x/crypto/ssh, github.com/pkg/sftp), Gin, SSE, React

---

## 文件结构

```
backend/
├── internal/
│   ├── ssh/
│   │   ├── client.go        # SSH 客户端封装
│   │   └── sftp.go         # SFTP 文件传输
│   ├── service/
│   │   └── installer.go    # 安装服务（SSH + 执行）
│   ├── handler/
│   │   ├── server.go       # 添加 InstallAPI
│   │   └── install.go      # SSE 安装输出处理
│   └── model/
│       └── task.go         # 安装任务模型
frontend/
└── src/
    └── pages/servers/
        └── index.tsx       # 修改安装按钮逻辑
```

---

## Phase 1: SSH 客户端

### Task 1: SSH 客户端封装

**Files:**
- Create: `backend/internal/ssh/client.go`

```go
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

// 密钥认证
type KeyAuth struct {
    PrivateKey string
    Passphrase string
}

func (a *KeyAuth) AuthMethod() ssh.AuthMethod {
    signer, _ := ssh.ParsePrivateKeyWithPassphrase([]byte(a.PrivateKey), []byte(a.Passphrase))
    return ssh.PublicKeys(signer)
}

// 密码认证
type PasswordAuth struct {
    Password string
}

func (a *PasswordAuth) AuthMethod() ssh.AuthMethod {
    return ssh.Password(a.Password)
}

type SSHClient struct {
    client *ssh.Client
}

func NewSSHClient(host string, port int, user string, auth SSHAuth) (*SSHClient, error) {
    config := &ssh.ClientConfig{
        User: user,
        Auth: []ssh.AuthMethod{auth.AuthMethod()},
        HostKeyCallback: ssh.InsecureIgnoreHostKey(),
    }

    addr := fmt.Sprintf("%s:%d", host, port)
    client, err := ssh.Dial("tcp", addr, config)
    if err != nil {
        return nil, fmt.Errorf("ssh dial failed: %w", err)
    }

    return &SSHClient{client: client}, nil
}

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

func (c *SSHClient) ExecuteWithPty(cmd string, stdout io.Writer, stderr io.Writer) error {
    session, err := c.client.NewSession()
    if err != nil {
        return err
    }
    defer session.Close()

    session.Stdout = stdout
    session.Stderr = stderr
    session.RequestPty("xterm", 80, 30, ssh.TerminalModes{})

    return session.Shell()
}

func (c *SSHClient) Close() error {
    return c.client.Close()
}
```

- [ ] **Step 1: 创建 ssh 目录**

```bash
mkdir -p backend/internal/ssh
```

- [ ] **Step 2: 创建 client.go**

```go
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
```

- [ ] **Step 3: 提交**

```bash
git add backend/internal/ssh/client.go
git commit -m "feat: add SSH client with key and password auth support"
```

---

### Task 2: SFTP 文件传输

**Files:**
- Create: `backend/internal/ssh/sftp.go`

```go
package ssh

import (
    "fmt"
    "github.com/pkg/sftp"
    "io"
    "os"
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
```

- [ ] **Step 1: 添加 sftp 依赖**

```bash
go get github.com/pkg/sftp
```

- [ ] **Step 2: 创建 sftp.go**

```go
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
```

- [ ] **Step 3: 提交**

```bash
git add backend/internal/ssh/sftp.go
git commit -m "feat: add SFTP client for file upload"
```

---

## Phase 2: 安装服务

### Task 3: 安装服务

**Files:**
- Create: `backend/internal/service/installer.go`

```go
package service

import (
    "fmt"
    "io"
    "os"
    "sync"
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
}

func NewInstaller(serverID, host string, port int, user string, auth ssh.SSHAuth, scriptPath string) *Installer {
    return &Installer{
        serverID:   serverID,
        host:       host,
        port:       port,
        user:       user,
        auth:       auth,
        scriptPath: scriptPath,
    }
}

func (i *Installer) Install(output io.Writer) *InstallResult {
    // 1. 连接 SSH
    fmt.Fprintf(output, "正在连接到 %s:%d...\n", i.host, i.port)
    sshClient, err := ssh.NewSSHClient(i.host, i.port, i.user, i.auth)
    if err != nil {
        return &InstallResult{Success: false, Error: fmt.Sprintf("SSH连接失败: %v", err)}
    }
    defer sshClient.Close()
    fmt.Fprintf(output, "SSH连接成功\n")

    // 2. 上传脚本
    fmt.Fprintf(output, "正在上传安装脚本...\n")
    sftpClient, err := ssh.NewSFTPClient(sshClient)
    if err != nil {
        return &InstallResult{Success: false, Error: fmt.Sprintf("SFTP连接失败: %v", err)}
    }
    defer sftpClient.Close()

    remotePath := "/tmp/v2ray_install.sh"
    if err := sftpClient.UploadFile(i.scriptPath, remotePath); err != nil {
        return &InstallResult{Success: false, Error: fmt.Sprintf("上传脚本失败: %v", err)}
    }
    fmt.Fprintf(output, "脚本上传成功\n")

    // 3. 执行安装
    fmt.Fprintf(output, "正在执行安装脚本...\n")
    cmd := fmt.Sprintf("chmod +x %s && bash %s", remotePath, remotePath)
    if err := sshClient.Execute(cmd, output, output); err != nil {
        return &InstallResult{Success: false, Error: fmt.Sprintf("安装执行失败: %v", err)}
    }

    fmt.Fprintf(output, "\n✓ 安装完成\n")
    return &InstallResult{Success: true}
}
```

- [ ] **Step 1: 创建 service 目录**

```bash
mkdir -p backend/internal/service
```

- [ ] **Step 2: 创建 installer.go**

```go
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
}

func NewInstaller(serverID, host string, port int, user string, auth ssh.SSHAuth, scriptPath string) *Installer {
    return &Installer{
        serverID:   serverID,
        host:       host,
        port:       port,
        user:       user,
        auth:       auth,
        scriptPath: scriptPath,
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
    defer sftpClient.Close()

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
```

- [ ] **Step 3: 提交**

```bash
git add backend/internal/service/installer.go
git commit -m "feat: add installer service with SSH and SFTP support"
```

---

## Phase 3: API

### Task 4: 安装任务模型

**Files:**
- Create: `backend/internal/model/task.go`

```go
package model

import (
    "time"
)

type InstallTask struct {
    ID          string     `json:"id"`
    ServerID    string     `json:"server_id"`
    Status      string     `json:"status"` // pending, running, success, failed
    Output      string     `json:"output"`
    Error       string     `json:"error"`
    StartedAt   time.Time  `json:"started_at"`
    CompletedAt *time.Time `json:"completed_at"`
}
```

- [ ] **Step 1: 创建 task.go**

```go
package model

import (
    "time"
)

type InstallTask struct {
    ID          string     `json:"id"`
    ServerID    string     `json:"server_id"`
    Status      string     `json:"status"` // pending, running, success, failed
    Output      string     `json:"output"`
    Error       string     `json:"error"`
    StartedAt   time.Time  `json:"started_at"`
    CompletedAt *time.Time `json:"completed_at"`
}
```

- [ ] **Step 2: 提交**

```bash
git add backend/internal/model/task.go
git commit -m "feat: add InstallTask model"
```

---

### Task 5: 安装 API 和 SSE

**Files:**
- Create: `backend/internal/handler/install.go`
- Modify: `backend/internal/handler/routes.go`

```go
package handler

import (
    "encoding/json"
    "fmt"
    "net/http"
    "time"

    "github.com/gin-gonic/gin"
    "v2ray-dash/backend/internal/model"
    "v2ray-dash/backend/internal/service"
    "v2ray-dash/backend/internal/ssh"
)

type InstallHandler struct {
    scriptPath string
}

func NewInstallHandler(scriptPath string) *InstallHandler {
    return &InstallHandler{scriptPath: scriptPath}
}

func (h *InstallHandler) StartInstall(c *gin.Context) {
    serverID := c.Param("id")

    // 获取服务器信息
    var server struct {
        IP         string `json:"ip"`
        SSHPort    int    `json:"ssh_port"`
        SSHUser    string `json:"ssh_user"`
        SSHKeyType string `json:"ssh_key_type"`
        SSHKey     string `json:"ssh_key"`
        SSHPassword string `json:"ssh_password"`
    }

    // TODO: 从数据库获取服务器信息
    // 这里简化处理，实际应该调用 serverRepo.GetByID(serverID)

    server.IP = "127.0.0.1" // 测试用
    server.SSHPort = 22
    server.SSHUser = "root"
    server.SSHKeyType = "key"
    server.SSHKey = ""

    // 创建 SSH 认证
    var auth ssh.SSHAuth
    if server.SSHKeyType == "password" {
        auth = &ssh.PasswordAuth{Password: server.SSHPassword}
    } else {
        auth = &ssh.KeyAuth{PrivateKey: server.SSHKey}
    }

    // 设置 SSE headers
    c.Header("Content-Type", "text/event-stream")
    c.Header("Cache-Control", "no-cache")
    c.Header("Connection", "keep-alive")
    c.Header("Transfer-Encoding", "chunked")

    // 创建安装器
    installer := service.NewInstaller(serverID, server.IP, server.SSHPort, server.SSHUser, auth, h.scriptPath)

    // 创建 PipeWriter 用于流式输出
   flusher, ok := c.Writer.(http.Flusher)
    if !ok {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "streaming not supported"})
        return
    }

    // 执行安装，写入响应
    result := installer.Install(c.Writer)
    flusher.Flush()

    if !result.Success {
        c.Writer.Write([]byte("\n❌ " + result.Error + "\n"))
        flusher.Flush()
    }
}
```

- [ ] **Step 1: 创建 install.go**

```go
package handler

import (
    "net/http"

    "github.com/gin-gonic/gin"
    "v2ray-dash/backend/internal/service"
    "v2ray-dash/backend/internal/ssh"
)

type InstallHandler struct {
    scriptPath string
}

func NewInstallHandler(scriptPath string) *InstallHandler {
    return &InstallHandler{scriptPath: scriptPath}
}

func (h *InstallHandler) StartInstall(c *gin.Context) {
    serverID := c.Param("id")

    // TODO: 从数据库获取服务器信息
    // 这里简化处理，实际应该调用 serverRepo.GetByID(serverID)
    // 获取请求体中的模板信息
    var req struct {
        Template string `json:"template"`
    }
    c.ShouldBindJSON(&req)

    // 模拟服务器信息（实际从DB获取）
    server := struct {
        IP          string `json:"ip"`
        SSHPort     int    `json:"ssh_port"`
        SSHUser     string `json:"ssh_user"`
        SSHKeyType  string `json:"ssh_key_type"`
        SSHKey      string `json:"ssh_key"`
        SSHPassword string `json:"ssh_password"`
    }{
        IP:         "127.0.0.1",
        SSHPort:    22,
        SSHUser:    "root",
        SSHKeyType: "key",
    }

    // 创建 SSH 认证
    var auth ssh.SSHAuth
    if server.SSHKeyType == "password" {
        auth = &ssh.PasswordAuth{Password: server.SSHPassword}
    } else {
        auth = &ssh.KeyAuth{PrivateKey: server.SSHKey}
    }

    // 设置 SSE headers
    c.Header("Content-Type", "text/event-stream")
    c.Header("Cache-Control", "no-cache")
    c.Header("Connection", "keep-alive")
    c.Header("Transfer-Encoding", "chunked")

    flusher, ok := c.Writer.(http.Flusher)
    if !ok {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "streaming not supported"})
        return
    }

    // 创建安装器
    installer := service.NewInstaller(serverID, server.IP, server.SSHPort, server.SSHUser, auth, h.scriptPath)

    // 执行安装
    result := installer.Install(c.Writer)
    flusher.Flush()

    if !result.Success {
        c.Writer.Write([]byte("\n❌ " + result.Error + "\n"))
        flusher.Flush()
    }
}
```

- [ ] **Step 2: 添加路由到 routes.go**

在 routes.go 中添加：

```go
installHandler := NewInstallHandler("./install.sh")
api.POST("/servers/:id/install", installHandler.StartInstall)
```

- [ ] **Step 3: 提交**

```bash
git add backend/internal/handler/install.go backend/internal/handler/routes.go
git commit -m "feat: add install API with SSE streaming output"
```

---

## Phase 4: 前端

### Task 6: 前端安装弹窗

**Files:**
- Modify: `frontend/src/pages/servers/index.tsx`

- [ ] **Step 1: 修改 servers/index.tsx 添加安装功能**

主要改动：
1. 添加 `installModalVisible` state
2. 添加 `installOutput` state 存储实时输出
3. 添加 `handleInstall` 函数发起 SSE 请求
4. 修改安装按钮点击事件

```tsx
const handleInstall = (server: Server) => {
    setSelectedServer(server)
    setInstallOutput('')
    setInstallModalVisible(true)

    // 通过 SSE 获取实时输出
    const eventSource = new EventSource(`/api/servers/${server.id}/install`)

    eventSource.onmessage = (e) => {
        setInstallOutput(prev => prev + e.data)
    }

    eventSource.onerror = () => {
        eventSource.close()
    }

    setEventSource(eventSource)
}
```

- [ ] **Step 2: 提交**

```bash
git add frontend/src/pages/servers/index.tsx
git commit -m "feat: add real-time install output via SSE"
```

---

## 实施检查清单

| Phase | Task | Description | Status |
|-------|------|-------------|--------|
| 1 | 1 | SSH 客户端封装 | ☐ |
| 1 | 2 | SFTP 文件传输 | ☐ |
| 2 | 3 | 安装服务 | ☐ |
| 3 | 4 | 安装任务模型 | ☐ |
| 3 | 5 | 安装 API 和 SSE | ☐ |
| 4 | 6 | 前端安装弹窗 | ☐ |

---

**Plan saved to:** `docs/superpowers/plans/2026-05-16-ssh-install-plan.md`

**执行选项：**

**1. Subagent-Driven (推荐)** - 每个任务分配一个 subagent 执行，完成后 review，快速迭代

**2. Inline Execution** - 在当前 session 中执行任务，带 checkpoint 审核

**选择哪个方式？**
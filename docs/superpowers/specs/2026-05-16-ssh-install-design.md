# 远程服务器自动安装功能 - 设计文档

**日期：** 2026-05-16
**状态：** 待审批

---

## 1. 目标

实现从 Web 界面一键连接远程服务器，自动上传并执行 install.sh，完成 v2ray + Agent 安装。

---

## 2. 技术方案

### 2.1 SSH 连接

使用 Go 标准库 `golang.org/x/crypto/ssh`，这是 Go 生态中最成熟稳定的 SSH 客户端库。

**关键组件：**
- `ssh.Dial()` - 建立 SSH 连接
- `ssh.NewClientConn()` - 创建会话
- `ssh.Session` - 执行命令
- `ssh.Session.Stdout/Stderr` - 获取输出流

### 2.2 安装脚本传输

通过 SFTP (SSH File Transfer Protocol) 上传 install.sh：

```bash
sftp client := sftp.NewClient(sshClient)
client.Upload("/tmp/v2ray_install.sh", localFile)
```

### 2.3 执行与结果实时显示

```bash
session.Stdout = os.Stdout  // 实时输出到控制台
session.Stderr = os.Stderr  // 实时错误输出
session.Run("bash /tmp/v2ray_install.sh --agent --url ... --id ... --psk ...")
```

### 2.4 流程设计

```
用户点击"安装"
    ↓
后端建立 SSH 连接（使用服务器保存的 SSH 凭证）
    ↓
通过 SFTP 上传 install.sh 到远程服务器 /tmp/
    ↓
SSH 执行脚本，实时返回输出到前端
    ↓
安装完成后，Agent 自动启动并连接控制中心
    ↓
Agent 后续通过心跳上报状态
```

---

## 3. 数据结构

### 3.1 安装任务模型

```go
type InstallTask struct {
    ID           string    `json:"id"`
    ServerID     string    `json:"server_id"`
    Status       string    `json:"status"`        // pending, running, success, failed
    Output       string    `json:"output"`        // 安装日志
    StartedAt    time.Time `json:"started_at"`
    CompletedAt  time.Time `json:"completed_at"`
}
```

### 3.2 WebSocket 消息格式

```go
type InstallOutput struct {
    TaskID   string `json:"task_id"`
    Type     string `json:"type"`     // stdout, stderr, status
    Content  string `json:"content"`
    Final    bool   `json:"final"`    // 是否最终结果
}
```

---

## 4. API 设计

### 4.1 发起安装

```
POST /api/servers/:id/install
Request: { "template": "standard-reality" }
Response: { "task_id": "xxx", "status": "running" }
```

### 4.2 WebSocket 连接

```
GET /api/servers/:id/install/ws
- 连接后接收安装实时输出
- 类型: stdout, stderr, status
```

### 4.3 获取安装状态

```
GET /api/tasks/:id
Response: { "id", "status", "output", "started_at", "completed_at" }
```

---

## 5. 后端实现

### 5.1 新增文件

- `backend/internal/ssh/client.go` - SSH 客户端封装
- `backend/internal/ssh/sftp.go` - SFTP 文件传输
- `backend/internal/service/installer.go` - 安装服务

### 5.2 SSH 客户端核心逻辑

```go
type SSHClient struct {
    client *ssh.Client
}

func NewSSHClient(host string, port int, user string, auth SSHAuth) (*SSHClient, error)

func (c *SSHClient) UploadFile(localPath, remotePath string) error

func (c *SSHClient) ExecuteWithOutput(cmd string, writer io.Writer) error

func (c *SSHClient) Close() error

type SSHAuth interface {
    AuthMethod() ssh.AuthMethod
}

// 密钥认证
type KeyAuth struct {
    PrivateKey string
}

func (a *KeyAuth) AuthMethod() ssh.AuthMethod

// 密码认证
type PasswordAuth struct {
    Password string
}

func (a *PasswordAuth) AuthMethod() ssh.AuthMethod
```

---

## 6. 前端实现

### 6.1 安装弹窗

```
┌─────────────────────────────────────────────────────┐
│  正在安装 v2ray 到 服务器A (1.2.3.4)                  │
├─────────────────────────────────────────────────────┤
│                                                     │
│  [root@1.2.3.4]# bash /tmp/install.sh              │
│  下载 v2ray...                                     │
│  配置中...                                          │
│  启动服务...                                        │
│  安装 Agent...                                      │
│                                                     │
│  ✓ 安装完成                                         │
│                                                     │
├─────────────────────────────────────────────────────┤
│                              [取消安装] [关闭]      │
└─────────────────────────────────────────────────────┘
```

### 6.2 技术实现

- 使用 WebSocket 连接后端实时获取安装输出
- 输出流式显示在终端风格的文本框中
- 完成后显示成功/失败状态

---

## 7. 错误处理

| 错误场景 | 处理方式 |
|---------|---------|
| SSH 连接失败 | 提示"无法连接服务器，检查网络和SSH配置" |
| 认证失败 | 提示"SSH认证失败，检查用户名和密码/密钥" |
| SFTP 上传失败 | 提示"上传安装脚本失败" |
| 安装脚本执行失败 | 显示错误日志，提供重试按钮 |
| 安装超时 (10分钟) | 自动终止，显示超时错误 |

---

## 8. 安全考虑

1. **敏感信息不记录日志** - SSH 密码/私钥不写入日志
2. **超时自动终止** - 防止僵尸安装进程
3. **命令白名单** - 只执行预定义的安装命令，防止注入

---

## 9. 实现步骤

### Phase 1: SSH 客户端
- [ ] SSH 客户端封装 (支持密钥和密码认证)
- [ ] SFTP 文件上传
- [ ] 命令执行与输出捕获

### Phase 2: 安装 API
- [ ] 安装任务管理
- [ ] WebSocket 实时输出
- [ ] 错误处理与重试

### Phase 3: 前端集成
- [ ] 安装弹窗 UI
- [ ] WebSocket 消息处理
- [ ] 状态显示与错误提示

---

**审批状态：** 待审批
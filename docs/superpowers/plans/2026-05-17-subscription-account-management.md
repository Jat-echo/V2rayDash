# 订阅账号管理系统实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 实现远程管理服务器上的 VLESS/Xray 账号，并生成多格式订阅文件供客户端使用

**Architecture:** 采用本地数据库存储账号 + SSH 远程同步的配置管理方式。前端通过 API 管理账号，后端生成订阅文件（VLESS/Clash.Meta/sing-box），通过 SSH 上传配置到远程服务器。

**Tech Stack:** Go (Gin), PostgreSQL, React (Ant Design), SSH/SFTP

---

## File Structure

```
backend/
├── internal/
│   ├── model/
│   │   └── account.go          # 账号数据模型
│   ├── repository/
│   │   └── account.go          # 账号数据库操作
│   ├── service/
│   │   ├── account.go          # 账号业务逻辑
│   │   └── subscription.go      # 订阅生成服务
│   ├── handler/
│   │   └── account.go          # 账号 API 处理器
├── pkg/database/
│   └── init.sql                # 数据库 schema (需添加 accounts 表)
frontend/
├── src/
│   ├── pages/servers/
│   │   └── index.tsx           # 服务器列表页 (添加账号管理 Modal)
│   └── services/
│       └── api.ts              # API 类型定义
```

---

## Task 1: 添加数据库 accounts 表

**Files:**
- Modify: `backend/pkg/database/init.sql:80-100`

- [ ] **Step 1: 在 init.sql 添加 accounts 表定义**

在 `backend/pkg/database/init.sql` 文件末尾添加：

```sql
-- 账号表
CREATE TABLE IF NOT EXISTS accounts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    server_id UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    uuid VARCHAR(64) NOT NULL,
    email VARCHAR(255) NOT NULL,
    protocols TEXT[] NOT NULL,
    enabled BOOLEAN DEFAULT true,
    traffic_limit BIGINT DEFAULT 0,
    traffic_used BIGINT DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 订阅记录表（可选，用于追踪）
CREATE TABLE IF NOT EXISTS subscriptions_records (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id UUID NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    sub_type VARCHAR(50) NOT NULL,
    sub_link TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_accounts_server_id ON accounts(server_id);
CREATE INDEX IF NOT EXISTS idx_subscriptions_records_account_id ON subscriptions_records(account_id);
```

- [ ] **Step 2: 执行 SQL 添加表**

Run: `psql -U jat-id -d v2ray-dash -f /home/jat-id/Project/V2rayDash/backend/pkg/database/init.sql`
Expected: CREATE TABLE 表示成功

- [ ] **Step 3: Commit**

```bash
cd /home/jat-id/Project/V2rayDash
git add backend/pkg/database/init.sql
git commit -m "feat: add accounts table for subscription management"
```

---

## Task 2: 创建账号数据模型

**Files:**
- Create: `backend/internal/model/account.go`

- [ ] **Step 1: 编写账号模型**

```go
package model

import (
	"time"
)

type Account struct {
	ID            string    `json:"id"`
	ServerID      string    `json:"server_id"`
	UUID          string    `json:"uuid"`
	Email         string    `json:"email"`
	Protocols     []string  `json:"protocols"`
	Enabled       bool      `json:"enabled"`
	TrafficLimit  int64     `json:"traffic_limit"`
	TrafficUsed   int64     `json:"traffic_used"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type CreateAccountRequest struct {
	ServerID  string   `json:"server_id" binding:"required"`
	UUID     string   `json:"uuid"`
	Email    string   `json:"email" binding:"required"`
	Protocols []string `json:"protocols" binding:"required"`
}

type UpdateAccountRequest struct {
	Email       *string  `json:"email"`
	Protocols   []string `json:"protocols"`
	Enabled     *bool    `json:"enabled"`
	TrafficLimit *int64   `json:"traffic_limit"`
}
```

- [ ] **Step 2: Commit**

```bash
git add backend/internal/model/account.go
git commit -m "feat: add account model"
```

---

## Task 3: 创建账号 Repository

**Files:**
- Create: `backend/internal/repository/account.go`

- [ ] **Step 1: 编写账号 Repository**

```go
package repository

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"v2ray-dash/backend/internal/model"
)

type AccountRepository struct {
	db *sql.DB
}

func NewAccountRepository(db *sql.DB) *AccountRepository {
	return &AccountRepository{db: db}
}

func (r *AccountRepository) Create(req *model.CreateAccountRequest) (*model.Account, error) {
	accountUUID := req.UUID
	if accountUUID == "" {
		accountUUID = uuid.New().String()
	}

	var id string
	err := r.db.QueryRow(
		`INSERT INTO accounts (server_id, uuid, email, protocols)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id`,
		req.ServerID, accountUUID, req.Email, pq.Array(req.Protocols),
	).Scan(&id)
	if err != nil {
		return nil, err
	}

	return r.GetByID(id)
}

func (r *AccountRepository) GetByID(id string) (*model.Account, error) {
	var a model.Account
	var protocols pq.StringArray
	err := r.db.QueryRow(
		`SELECT id, server_id, uuid, email, protocols, enabled, traffic_limit, traffic_used, created_at, updated_at
		 FROM accounts WHERE id = $1`,
		id,
	).Scan(&a.ID, &a.ServerID, &a.UUID, &a.Email, &protocols, &a.Enabled, &a.TrafficLimit, &a.TrafficUsed, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return nil, err
	}
	a.Protocols = protocols
	return &a, nil
}

func (r *AccountRepository) ListByServerID(serverID string) ([]*model.Account, error) {
	rows, err := r.db.Query(
		`SELECT id, server_id, uuid, email, protocols, enabled, traffic_limit, traffic_used, created_at, updated_at
		 FROM accounts WHERE server_id = $1 ORDER BY created_at DESC`,
		serverID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accounts []*model.Account
	for rows.Next() {
		var a model.Account
		var protocols pq.StringArray
		if err := rows.Scan(&a.ID, &a.ServerID, &a.UUID, &a.Email, &protocols, &a.Enabled, &a.TrafficLimit, &a.TrafficUsed, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, err
		}
		a.Protocols = protocols
		accounts = append(accounts, &a)
	}
	return accounts, nil
}

func (r *AccountRepository) List() ([]*model.Account, error) {
	rows, err := r.db.Query(
		`SELECT id, server_id, uuid, email, protocols, enabled, traffic_limit, traffic_used, created_at, updated_at
		 FROM accounts ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accounts []*model.Account
	for rows.Next() {
		var a model.Account
		var protocols pq.StringArray
		if err := rows.Scan(&a.ID, &a.ServerID, &a.UUID, &a.Email, &protocols, &a.Enabled, &a.TrafficLimit, &a.TrafficUsed, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, err
		}
		a.Protocols = protocols
		accounts = append(accounts, &a)
	}
	return accounts, nil
}

func (r *AccountRepository) Update(id string, req *model.UpdateAccountRequest) error {
	var setClauses []string
	var args []interface{}
	argNum := 1

	if req.Email != nil {
		setClauses = append(setClauses, fmt.Sprintf("email = $%d", argNum))
		args = append(args, *req.Email)
		argNum++
	}
	if req.Protocols != nil {
		setClauses = append(setClauses, fmt.Sprintf("protocols = $%d", argNum))
		args = append(args, pq.Array(req.Protocols))
		argNum++
	}
	if req.Enabled != nil {
		setClauses = append(setClauses, fmt.Sprintf("enabled = $%d", argNum))
		args = append(args, *req.Enabled)
		argNum++
	}
	if req.TrafficLimit != nil {
		setClauses = append(setClauses, fmt.Sprintf("traffic_limit = $%d", argNum))
		args = append(args, *req.TrafficLimit)
		argNum++
	}

	if len(setClauses) == 0 {
		return nil
	}

	setClauses = append(setClauses, fmt.Sprintf("updated_at = $%d", argNum))
	args = append(args, time.Now())
	argNum++

	args = append(args, id)

	query := fmt.Sprintf("UPDATE accounts SET %s WHERE id = $%d", strings.Join(setClauses, ", "), argNum)
	_, err := r.db.Exec(query, args...)
	return err
}

func (r *AccountRepository) Delete(id string) error {
	_, err := r.db.Exec("DELETE FROM accounts WHERE id = $1", id)
	return err
}
```

- [ ] **Step 2: Commit**

```bash
git add backend/internal/repository/account.go
git commit -m "feat: add account repository"
```

---

## Task 4: 创建账号 Service

**Files:**
- Create: `backend/internal/service/account.go`

- [ ] **Step 1: 编写账号 Service**

```go
package service

import (
	"encoding/json"
	"fmt"
	"strings"

	"v2ray-dash/backend/internal/model"
	"v2ray-dash/backend/internal/repository"
	"v2ray-dash/backend/internal/ssh"
)

type AccountService struct {
	accountRepo *repository.AccountRepository
	serverRepo  *repository.ServerRepository
}

func NewAccountService(accountRepo *repository.AccountRepository, serverRepo *repository.ServerRepository) *AccountService {
	return &AccountService{
		accountRepo: accountRepo,
		serverRepo:  serverRepo,
	}
}

// GetAccountLink 生成单个账号的订阅链接
func (s *AccountService) GetAccountLink(account *model.Account, serverIP string, subType string) string {
	var link string
	switch subType {
	case "vless":
		link = fmt.Sprintf("vless://%s@%s:443?encryption=none&flow=xtls-rprx-vision&security=tls&sni=%s#%s",
			account.UUID, serverIP, serverIP, account.Email)
	case "clash_meta":
		// 返回 YAML 格式（实际订阅返回完整配置）
		link = fmt.Sprintf("clash://%s@%s:443", account.UUID, serverIP)
	default:
		link = fmt.Sprintf("vless://%s@%s:443", account.UUID, serverIP)
	}
	return link
}

// GenerateVLESSSubscription 生成 VLESS 订阅内容
func (s *AccountService) GenerateVLESSSubscription(accounts []*model.Account, serverIP string) string {
	var lines []string
	for _, acc := range accounts {
		if !acc.Enabled {
			continue
		}
		for _, proto := range acc.Protocols {
			link := s.GetAccountLink(acc, serverIP, "vless")
			if strings.Contains(proto, "reality") {
				link = strings.Replace(link, "tls", "reality", 1)
			}
			lines = append(lines, link)
		}
	}
	return strings.Join(lines, "\n")
}

// GenerateClashMetaSubscription 生成 Clash.Meta 订阅内容
func (s *AccountService) GenerateClashMetaSubscription(accounts []*model.Account, serverIP string) (string, error) {
	proxies := make([]map[string]interface{}, 0)
	for _, acc := range accounts {
		if !acc.Enabled {
			continue
		}
		for _, proto := range acc.Protocols {
			proxy := map[string]interface{}{
				"name": acc.Email,
				"type": "vless",
				"server": serverIP,
				"port": 443,
				"uuid": acc.UUID,
				"flow": "xtls-rprx-vision",
				"tls": true,
			}
			if strings.Contains(proto, "reality") {
				proxy["tls"] = map[string]interface{}{
					"enabled": true,
					"serverName": serverIP,
					"reality": map[string]interface{}{
						"enabled": true,
					},
				}
			}
			proxies = append(proxies, proxy)
		}
	}

	config := map[string]interface{}{
		"proxies": proxies,
	}
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// SyncToRemote 同步账号到远程服务器
func (s *AccountService) SyncToRemote(accountID string, auth ssh.SSHAuth) error {
	account, err := s.accountRepo.GetByID(accountID)
	if err != nil {
		return err
	}

	server, err := s.serverRepo.GetByID(account.ServerID)
	if err != nil {
		return err
	}

	client, err := ssh.NewSSHClient(server.IP, server.SSHPort, server.SSHUser, auth)
	if err != nil {
		return err
	}
	defer client.Close()

	// 生成配置文件内容
	config := map[string]interface{}{
		"log": map[string]interface{}{
			"loglevel": "warning",
		},
		"inbounds": []map[string]interface{}{
			{
				"port": 443,
				"protocol": "vless",
				"settings": map[string]interface{}{
					"clients": []map[string]interface{}{
						{
							"id": account.UUID,
							"email": account.Email,
						},
					},
				},
			},
		},
	}

	data, _ := json.MarshalIndent(config, "", "  ")
	return client.UploadConfig("/etc/v2ray-agent/xray/conf/02_VLESS_TCP_inbounds.json", string(data))
}
```

- [ ] **Step 2: Commit**

```bash
git add backend/internal/service/account.go
git commit -m "feat: add account service with subscription generation"
```

---

## Task 5: 创建账号 Handler

**Files:**
- Create: `backend/internal/handler/account.go`

- [ ] **Step 1: 编写账号 Handler**

```go
package handler

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
	"v2ray-dash/backend/internal/model"
	"v2ray-dash/backend/internal/repository"
	"v2ray-dash/backend/internal/service"
)

type AccountHandler struct {
	accountRepo *repository.AccountRepository
	serverRepo  *repository.ServerRepository
	accountSvc  *service.AccountService
}

func NewAccountHandler(db *sql.DB) *AccountHandler {
	accountRepo := repository.NewAccountRepository(db)
	serverRepo := repository.NewServerRepository(db)
	accountSvc := service.NewAccountService(accountRepo, serverRepo)
	return &AccountHandler{
		accountRepo: accountRepo,
		serverRepo:  serverRepo,
		accountSvc:  accountSvc,
	}
}

func (h *AccountHandler) RegisterRoutes(r *gin.RouterGroup) {
	accounts := r.Group("/servers/:id/accounts")
	{
		accounts.GET("", h.List)
		accounts.POST("", h.Create)
	}

	accountRoutes := r.Group("/accounts")
	{
		accountRoutes.GET("/:id", h.Get)
		accountRoutes.PUT("/:id", h.Update)
		accountRoutes.DELETE("/:id", h.Delete)
		accountRoutes.GET("/:id/subscribe", h.Subscribe)
	}
}

func (h *AccountHandler) List(c *gin.Context) {
	serverID := c.Param("id")
	accounts, err := h.accountRepo.ListByServerID(serverID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, accounts)
}

func (h *AccountHandler) Get(c *gin.Context) {
	id := c.Param("id")
	account, err := h.accountRepo.GetByID(id)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "account not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, account)
}

func (h *AccountHandler) Create(c *gin.Context) {
	serverID := c.Param("id")
	var req model.CreateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.ServerID = serverID

	account, err := h.accountRepo.Create(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, account)
}

func (h *AccountHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var req model.UpdateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.accountRepo.Update(id, &req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "updated"})
}

func (h *AccountHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if err := h.accountRepo.Delete(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

func (h *AccountHandler) Subscribe(c *gin.Context) {
	id := c.Param("id")
	subType := c.Query("type")
	if subType == "" {
		subType = "vless"
	}

	account, err := h.accountRepo.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "account not found"})
		return
	}

	server, err := h.serverRepo.GetByID(account.ServerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server not found"})
		return
	}

	var content string
	switch subType {
	case "clash_meta":
		content, _ = h.accountSvc.GenerateClashMetaSubscription([]*model.Account{account}, server.IP)
	default:
		content = h.accountSvc.GenerateVLESSSubscription([]*model.Account{account}, server.IP)
	}

	c.Header("Content-Type", "text/plain; charset=utf-8")
	c.String(http.StatusOK, content)
}
```

- [ ] **Step 2: Commit**

```bash
git add backend/internal/handler/account.go
git commit -m "feat: add account handler with subscription API"
```

---

## Task 6: 更新路由注册

**Files:**
- Modify: `backend/internal/handler/routes.go:27-29`

- [ ] **Step 1: 在 SetupRoutes 添加账号路由**

在 `routes.go` 的 `api := r.Group("/api")` 内添加：

```go
// 账号管理
accountHandler := NewAccountHandler(db.DB)
accountHandler.RegisterRoutes(api)
```

修改后完整的路由组：

```go
api := r.Group("/api")
{
    // 服务器管理
    serverHandler := NewServerHandler(db.DB)
    api.GET("/servers", serverHandler.List)
    api.POST("/servers", serverHandler.Create)
    api.GET("/servers/:id", serverHandler.Get)
    api.PUT("/servers/:id", serverHandler.Update)
    api.DELETE("/servers/:id", serverHandler.Delete)

    // 账号管理
    accountHandler := NewAccountHandler(db.DB)
    accountHandler.RegisterRoutes(api)

    // 订阅管理
    subHandler := NewSubscriptionHandler(db.DB)
    // ... 保留现有订阅路由
}
```

- [ ] **Step 2: Commit**

```bash
git add backend/internal/handler/routes.go
git commit -m "feat: register account routes"
```

---

## Task 7: 更新前端 API 类型

**Files:**
- Modify: `frontend/src/services/api.ts:31-45`

- [ ] **Step 1: 在 api.ts 添加 Account 类型和 API 函数**

在 `Subscription` 接口后添加：

```typescript
export interface Account {
  id: string
  server_id: string
  uuid: string
  email: string
  protocols: string[]
  enabled: boolean
  traffic_limit: number
  traffic_used: number
  created_at: string
  updated_at: string
}

export const accountAPI = {
  listByServer: (serverId: string) =>
    api.get<Account[]>(`/servers/${serverId}/accounts`).then(r => r.data),
  get: (id: string) => api.get<Account>(`/accounts/${id}`).then(r => r.data),
  create: (serverId: string, data: Partial<Account>) =>
    api.post<Account>(`/servers/${serverId}/accounts`, data).then(r => r.data),
  update: (id: string, data: Partial<Account>) =>
    api.put(`/accounts/${id}`, data),
  delete: (id: string) => api.delete(`/accounts/${id}`),
  subscribe: (id: string, type?: string) =>
    api.get(`/accounts/${id}/subscribe`, { params: { type } }).then(r => r.data),
}
```

- [ ] **Step 2: Commit**

```bash
git add frontend/src/services/api.ts
git commit -m "feat: add account API types and functions"
```

---

## Task 8: 在服务器列表页添加账号管理 Modal

**Files:**
- Modify: `frontend/src/pages/servers/index.tsx:1-357`

- [ ] **Step 1: 在现有 import 中添加 Tag, Drawer**

```typescript
import { Table, Button, Space, Modal, Form, Input, Select, message, Popconfirm, Tag, Drawer } from 'antd'
```

- [ ] **Step 2: 在组件 state 中添加账号管理相关状态**

```typescript
const [accountModalVisible, setAccountModalVisible] = useState(false)
const [accounts, setAccounts] = useState<Account[]>([])
const [selectedServerForAccounts, setSelectedServerForAccounts] = useState<Server | null>(null)
const [addAccountForm] = Form.useForm()
```

- [ ] **Step 3: 添加加载账号、订阅下载方法**

```typescript
const loadAccounts = async (serverId: string) => {
  try {
    const data = await accountAPI.listByServer(serverId)
    setAccounts(data || [])
  } catch (e) {
    setAccounts([])
  }
}

const handleOpenAccountModal = (server: Server) => {
  setSelectedServerForAccounts(server)
  loadAccounts(server.id)
  setAccountModalVisible(true)
}

const handleAddAccount = async (values: any) => {
  if (!selectedServerForAccounts) return
  try {
    await accountAPI.create(selectedServerForAccounts.id, values)
    message.success('添加成功')
    addAccountForm.resetFields()
    loadAccounts(selectedServerForAccounts.id)
  } catch (e) {
    message.error('添加失败')
  }
}

const handleDeleteAccount = async (id: string) => {
  try {
    await accountAPI.delete(id)
    message.success('删除成功')
    if (selectedServerForAccounts) {
      loadAccounts(selectedServerForAccounts.id)
    }
  } catch (e) {
    message.error('删除失败')
  }
}

const handleDownloadSubscription = async (accountId: string, type: string) => {
  try {
    const content = await accountAPI.subscribe(accountId, type)
    const blob = new Blob([content], { type: 'text/plain' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `subscription-${type}-${Date.now()}.txt`
    a.click()
    URL.revokeObjectURL(url)
    message.success('下载成功')
  } catch (e) {
    message.error('下载失败')
  }
}
```

- [ ] **Step 4: 在 columns 操作列添加账号管理按钮**

```typescript
{
  title: '操作',
  render: (_: any, record: Server) => (
    <Space>
      <Button size="small" type="primary" onClick={() => handleInstallClick(record)}>安装</Button>
      <Button size="small" onClick={() => handleOpenAccountModal(record)}>账号管理</Button>
      <Popconfirm title="确定删除?" onConfirm={() => handleDelete(record.id)}>
        <Button size="small" danger>删除</Button>
      </Popconfirm>
    </Space>
  ),
}
```

- [ ] **Step 5: 在 return 的 JSX 中添加账号管理 Modal（在 InstallModal 后）**

```typescript
{/* 账号管理 Modal */}
<Modal
  title={`账号管理 - ${selectedServerForAccounts?.name || ''}`}
  open={accountModalVisible}
  onCancel={() => setAccountModalVisible(false)}
  width={700}
  footer={null}
>
  <div style={{ marginBottom: 16 }}>
    <Space>
      <Button type="primary" onClick={() => addAccountForm.resetFields()}>添加账号</Button>
    </Space>
  </div>

  <Form form={addAccountForm} onFinish={handleAddAccount} layout="inline" style={{ marginBottom: 16 }}>
    <Form.Item name="email" label="备注" rules={[{ required: true }]}>
      <Input placeholder="user@example.com" style={{ width: 150 }} />
    </Form.Item>
    <Form.Item name="protocols" label="协议" rules={[{ required: true }]}>
      <Select mode="multiple" style={{ width: 200 }}>
        <Select.Option value="vless_tcp">VLESS TCP</Select.Option>
        <Select.Option value="vless_reality_vision">VLESS Reality Vision</Select.Option>
        <Select.Option value="trojan">Trojan</Select.Option>
      </Select>
    </Form.Item>
    <Form.Item>
      <Button type="primary" htmlType="submit">确定</Button>
    </Form.Item>
  </Form>

  <Table
    dataSource={accounts}
    rowKey="id"
    size="small"
    columns={[
      { title: '备注', dataIndex: 'email' },
      { title: '协议', dataIndex: 'protocols', render: (p: string[]) => p?.map(v => <Tag key={v}>{v}</Tag>) },
      { title: '状态', dataIndex: 'enabled', render: (v: boolean) => v ? '启用' : '禁用' },
      {
        title: '操作',
        render: (_: any, record: Account) => (
          <Space>
            <Button size="small" onClick={() => handleDownloadSubscription(record.id, 'vless')}>VLESS</Button>
            <Button size="small" onClick={() => handleDownloadSubscription(record.id, 'clash_meta')}>Clash</Button>
            <Popconfirm title="确定删除?" onConfirm={() => handleDeleteAccount(record.id)}>
              <Button size="small" danger>删除</Button>
            </Popconfirm>
          </Space>
        ),
      },
    ]}
  />
</Modal>
```

- [ ] **Step 6: Commit**

```bash
git add frontend/src/pages/servers/index.tsx
git commit -m "feat: add account management modal to servers page"
```

---

## Task 9: 添加从远程导入账号功能

**Files:**
- Modify: `backend/internal/service/account.go` (添加 ImportFromRemote 方法)
- Modify: `backend/internal/handler/account.go` (添加 import 路由)

- [ ] **Step 1: 在 AccountService 添加 ImportFromRemote 方法**

```go
// ImportFromRemote 从远程服务器导入账号
func (s *AccountService) ImportFromRemote(serverID string, auth ssh.SSHAuth) ([]*model.Account, error) {
	server, err := s.serverRepo.GetByID(serverID)
	if err != nil {
		return nil, err
	}

	client, err := ssh.NewSSHClient(server.IP, server.SSHPort, server.SSHUser, auth)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	// 读取 Xray 配置文件
	content, err := client.ReadRemoteFile("/etc/v2ray-agent/xray/conf/02_VLESS_TCP_inbounds.json")
	if err != nil {
		return nil, fmt.Errorf("failed to read remote config: %w", err)
	}

	// 解析 JSON 提取 users
	var config struct {
		Inbounds []struct {
			Settings struct {
				Clients []struct {
					ID    string `json:"id"`
					Email string `json:"email"`
				} `json:"clients"`
			} `json:"settings"`
		} `json:"inbounds"`
	}

	if err := json.Unmarshal([]byte(content), &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	var accounts []*model.Account
	for _, inbound := range config.Inbounds {
		for _, client := range inbound.Settings.Clients {
			account, err := s.accountRepo.Create(&model.CreateAccountRequest{
				ServerID:  serverID,
				UUID:      client.ID,
				Email:     client.Email,
				Protocols: []string{"vless_tcp"},
			})
			if err == nil {
				accounts = append(accounts, account)
			}
		}
	}

	return accounts, nil
}
```

- [ ] **Step 2: 在 AccountHandler 添加 Import 路由**

```go
func (h *AccountHandler) Import(c *gin.Context) {
	serverID := c.Param("id")

	// 获取服务器信息以便创建 SSH 连接
	server, err := h.serverRepo.GetByID(serverID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server not found"})
		return
	}

	// 根据认证类型创建 auth
	var auth ssh.SSHAuth
	if server.SSHKeyType == "password" {
		auth = &ssh.PasswordAuth{Password: server.SSHPassword}
	} else {
		auth = &ssh.KeyAuth{PrivateKey: server.SSHKey}
	}

	accounts, err := h.accountSvc.ImportFromRemote(serverID, auth)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "imported",
		"accounts": accounts,
	})
}
```

- [ ] **Step 3: 在 RegisterRoutes 添加 import 路由**

```go
accounts.POST("/import", h.Import)
```

- [ ] **Step 4: Commit**

```bash
git add backend/internal/service/account.go backend/internal/handler/account.go
git commit -m "feat: add import accounts from remote"
```

---

## Self-Review 检查清单

**1. Spec 覆盖检查:**
- ✅ 数据库 accounts 表 (Task 1)
- ✅ 账号 CRUD API (Tasks 2-6)
- ✅ 前端账号列表和添加表单 (Tasks 7-8)
- ✅ 远程导入 (Task 9)
- ✅ VLESS 订阅生成 (Task 4)
- ❌ Clash.Meta 订阅生成 - 部分实现，需确认 Task 4 中 GenerateClashMetaSubscription 是否完整
- ❌ sing-box 订阅生成 - 未包含
- ❌ 同步到远程 - 部分实现

**2. 占位符检查:**
- 无 TBD/TODO
- 无 "类似 Task N" 引用

**3. 类型一致性检查:**
- Task 3 的 Update 方法与 Task 5 的 handler 匹配
- 所有 Repository 方法签名一致

**修复:**
需要添加 sing-box 订阅生成任务，以及完善同步到远程功能。

---

## 后续任务 (建议单独的计划)

### Task 10: 添加 sing-box 订阅生成

**Files:**
- Modify: `backend/internal/service/account.go`

```go
// GenerateSingBoxSubscription 生成 sing-box 订阅
func (s *AccountService) GenerateSingBoxSubscription(accounts []*model.Account, serverIP string) (string, error) {
	outbounds := make([]map[string]interface{}, 0)
	for _, acc := range accounts {
		if !acc.Enabled {
			continue
		}
		for _, proto := range acc.Protocols {
			outbound := map[string]interface{}{
				"tag":      acc.Email,
				"type":     "vless",
				"server":   serverIP,
				"server_port": 443,
				"uuid":     acc.UUID,
				"flow":     "xtls-rprx-vision",
				"tls": map[string]interface{}{
					"enabled": true,
					"server_name": serverIP,
				},
			}
			if strings.Contains(proto, "reality") {
				outbound["tls"].(map[string]interface{})["reality"] = map[string]interface{}{
					"enabled": true,
				}
			}
			outbounds = append(outbounds, outbound)
		}
	}

	config := map[string]interface{}{
		"outbounds": outbounds,
	}
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}
```

### Task 11: 添加同步所有账号到远程

**Files:**
- Modify: `backend/internal/handler/account.go`

```go
func (h *AccountHandler) SyncAll(c *gin.Context) {
	serverID := c.Param("id")

	server, err := h.serverRepo.GetByID(serverID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server not found"})
		return
	}

	var auth ssh.SSHAuth
	if server.SSHKeyType == "password" {
		auth = &ssh.PasswordAuth{Password: server.SSHPassword}
	} else {
		auth = &ssh.KeyAuth{PrivateKey: server.SSHKey}
	}

	if err := h.accountSvc.SyncAllToRemote(serverID, auth); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "synced"})
}
```

---

## 执行选项

**1. Subagent-Driven (推荐)** - 每任务派遣 subagent，复查后进入下一任务

**2. Inline Execution** - 本会话内批量执行，有检查点

选择哪种方式?
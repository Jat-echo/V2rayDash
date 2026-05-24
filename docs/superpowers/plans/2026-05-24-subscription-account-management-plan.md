# 订阅账号动态管理实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 实现订阅账号的增删改查和排序能力，修复服务器删除时订阅报错的问题

**Architecture:** 数据库层扩展 subscription_accounts 表增加 sort_order 和 note 字段；后端 repository 层完善账号管理方法；前端订阅页面支持账号管理和空订阅友好提示

**Tech Stack:** Go/Gin backend, React/TypeScript frontend, PostgreSQL

---

## 任务 1: 数据库迁移 - 添加排序和备注字段

**Files:**
- Modify: `backend/pkg/database/schema.sql:49-58`
- Modify: `backend/pkg/database/init.go`

- [ ] **Step 1: 修改 schema.sql 添加字段**

```sql
ALTER TABLE subscription_accounts ADD COLUMN sort_order INT DEFAULT 0;
ALTER TABLE subscription_accounts ADD COLUMN note VARCHAR(255);
ALTER TABLE subscription_accounts ADD COLUMN updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP;
```

- [ ] **Step 2: 更新数据库初始化逻辑支持迁移**

在 `init.go` 的 `RunMigrations()` 函数中添加检测和 ALTER 语句

- [ ] **Step 3: 验证迁移**

运行后端，观察日志输出 "Migration already completed, skipping" 后确认无报错

---

## 任务 2: 后端 Repository 层 - 完善账号管理

**Files:**
- Modify: `backend/internal/repository/subscription_account.go:121-144`
- Modify: `backend/internal/repository/subscription_account.go:82-114`

- [ ] **Step 1: 更新 ReplaceAccounts 方法支持排序**

```go
func (r *SubscriptionAccountRepository) ReplaceAccounts(subscriptionID string, accountIDs []string, sortOrders map[string]int) error {
    tx, err := r.db.Begin()
    if err != nil {
        return err
    }
    defer tx.Rollback()

    _, err = tx.Exec(`DELETE FROM subscription_accounts WHERE subscription_id = $1`, subscriptionID)
    if err != nil {
        return err
    }

    for i, accountID := range accountIDs {
        sortOrder := i
        if so, ok := sortOrders[accountID]; ok {
            sortOrder = so
        }
        _, err = tx.Exec(`
            INSERT INTO subscription_accounts (subscription_id, account_id, sort_order)
            VALUES ($1, $2, $3)
        `, subscriptionID, accountID, sortOrder)
        if err != nil {
            return err
        }
    }

    return tx.Commit()
}
```

- [ ] **Step 2: 更新 GetAccountsWithServerInfo 支持排序**

修改查询语句：
```sql
SELECT ... ORDER BY sa.sort_order ASC, a.created_at DESC
```

- [ ] **Step 3: 添加 UpdateSortOrder 方法**

```go
func (r *SubscriptionAccountRepository) UpdateSortOrder(subscriptionID string, accountID string, sortOrder int) error {
    _, err := r.db.Exec(`
        UPDATE subscription_accounts
        SET sort_order = $1, updated_at = CURRENT_TIMESTAMP
        WHERE subscription_id = $2 AND account_id = $3
    `, sortOrder, subscriptionID, accountID)
    return err
}
```

- [ ] **Step 4: 添加 GetBySubscriptionOrdered 方法**

```go
func (r *SubscriptionAccountRepository) GetBySubscriptionOrdered(subscriptionID string) ([]*model.AccountWithServerInfo, error) {
    rows, err := r.db.Query(`
        SELECT a.id, a.server_id, a.uuid, a.email, a.protocols, a.enabled,
               a.traffic_limit, a.traffic_used, a.created_at, a.updated_at,
               s.name as server_name, s.ip as server_ip,
               sa.sort_order
        FROM accounts a
        JOIN subscription_accounts sa ON a.id = sa.account_id
        JOIN servers s ON a.server_id = s.id
        WHERE sa.subscription_id = $1
        ORDER BY sa.sort_order ASC, a.created_at DESC
    `, subscriptionID)
    // ... 完整实现见现有 GetAccountsWithServerInfo
}
```

- [ ] **Step 5: 测试 Repository 方法**

```bash
cd backend && go test ./internal/repository/... -v -run TestSubscriptionAccount
```

---

## 任务 3: 后端 Handler 层 - 更新订阅更新接口

**Files:**
- Modify: `backend/internal/handler/subscription.go:141-182`

- [ ] **Step 1: 修改 Update 方法支持完整账号替换**

更新 `UpdateSubscriptionRequest` 模型：
```go
type UpdateSubscriptionRequest struct {
    Name            *string           `json:"name"`
    Enable          *bool             `json:"enable"`
    TrafficLimit    *int64            `json:"traffic_limit"`
    AccountMappings *[]AccountMapping `json:"account_mappings"`
}
```

在 `Update` handler 中，当 `AccountMappings` 不为空时，先删除所有关联再插入新关联：
```go
if req.AccountMappings != nil {
    accountIDs := make([]string, 0)
    sortOrders := make(map[string]int)

    for i, mapping := range *req.AccountMappings {
        var accountID string
        if mapping.AutoCreate {
            newAccount, err := h.accountRepo.Create(&model.CreateAccountRequest{
                ServerID:  mapping.ServerID,
                Email:    fmt.Sprintf("auto-%s", id[:8]),
                Protocols: []string{"vless_tcp"},
            })
            if err != nil {
                continue
            }
            accountID = newAccount.ID
        } else {
            accountID = mapping.AccountID
        }
        accountIDs = append(accountIDs, accountID)
        sortOrders[accountID] = i
    }

    if len(accountIDs) > 0 {
        h.subAccRepo.ReplaceAccounts(id, accountIDs, sortOrders)
    }
}
```

- [ ] **Step 6: 编译测试**

```bash
cd backend && go build -o /tmp/v2ray-dash-test ./cmd/server
```

---

## 任务 4: 后端 API 路由确认

**Files:**
- Modify: `backend/internal/handler/routes.go:71-79`

确认以下路由存在且正确：

```go
api.GET("/subscriptions", subHandler.List)
api.GET("/subscriptions/full", subHandler.ListWithAccounts)
api.POST("/subscriptions", subHandler.Create)
api.PUT("/subscriptions/:id", subHandler.Update)
api.DELETE("/subscriptions/:id", subHandler.Delete)
api.GET("/subscriptions/:id/link", subHandler.GetLink)
api.POST("/subscriptions/:id/accounts", subHandler.AddAccount)
api.DELETE("/subscriptions/:id/accounts/:accountId", subHandler.RemoveAccount)
```

---

## 任务 5: 前端 - 订阅详情页账号管理

**Files:**
- Modify: `frontend/src/pages/subscriptions/index.tsx`

- [ ] **Step 1: 添加账号排序和管理的 UI**

在订阅列表的操作列添加"管理账号"按钮，打开 Modal：

```tsx
const handleManageAccounts = async (subscriptionId: string) => {
  const accounts = await subscriptionAPI.getAccountsWithSort(subscriptionId)
  setManagedAccounts(accounts)
  setManageModalVisible(true)
}
```

Modal 内支持：
- 拖拽排序（DndKit 或 react-beautiful-dnd）
- 移除账号（确认后调用 DELETE）
- 添加账号（从服务器列表选择）

- [ ] **Step 2: 实现拖拽排序保存**

```tsx
const onDragEnd = async (result: any) => {
  if (!result.destination) return

  const items = Array.from(managedAccounts)
  const [reorderedItem] = items.splice(result.source.index, 1)
  items.splice(result.destination.index, 0, reorderedItem)

  setManagedAccounts(items)

  // 保存新顺序
  const newOrder = items.map((acc, idx) => ({ id: acc.id, sort_order: idx }))
  await subscriptionAPI.updateAccountOrder(subscriptionId, newOrder)
}
```

- [ ] **Step 3: 测试账号管理功能**

```bash
cd frontend && npm run dev
# 打开 http://localhost:5173/subscriptions
# 点击订阅的"订阅链接" -> 观察账号列表
# 测试增删排序
```

---

## 任务 6: 前端 - 空订阅友好提示

**Files:**
- Modify: `frontend/src/pages/subscriptions/index.tsx:164-174`

- [ ] **Step 1: 更新订阅列表渲染逻辑**

```tsx
{
  title: '服务器/账号',
  render: (_: any, record: SubscriptionWithAccounts) => {
    if (!record.accounts || record.accounts.length === 0) {
      return <Tag color="default">暂无可用节点，请联系管理员</Tag>
    }
    return (
      <Space direction="vertical" size={2}>
        {record.accounts.map(acc => (
          <Tag key={acc.id} color="blue">{acc.server_name} / {acc.email}</Tag>
        ))}
      </Space>
    )
  }
}
```

---

## 任务 7: 验证服务器删除场景

**Files:**
- Modify: `backend/internal/handler/server.go:90-106`

- [ ] **Step 1: 确认删除服务器的级联行为**

删除服务器后：
1. 数据库中该服务器的账号应被删除（CASCADE）
2. subscription_accounts 中相关记录应被自动清除
3. 订阅本身保留，`server_id` 设为 NULL

测试：
```bash
# 1. 确认有订阅关联账号
psql 'postgres://v2ray_dash:YourNewPass123@localhost:5432/v2ray_dash?sslmode=disable' -c "SELECT * FROM subscription_accounts WHERE account_id IN (SELECT id FROM accounts WHERE server_id = 'xxx');"

# 2. 删除服务器
curl -X DELETE http://localhost:8080/api/servers/xxx

# 3. 确认订阅仍然存在但账号为空
curl http://localhost:8080/api/subscriptions/full
```

---

## 任务 8: 部署到阿里云

**Files:**
- N/A - 执行部署

- [ ] **Step 1: 本地构建**

```bash
cd backend && /usr/local/go/bin/go build -o v2ray-dash-backend ./cmd/server
```

- [ ] **Step 2: 上传到服务器**

```bash
sshpass -p 'Jat02300920#AL' scp backend/v2ray-dash-backend root@112.125.93.190:/tmp/
sshpass -p 'Jat02300920#AL' ssh root@112.125.93.190 "mv /tmp/v2ray-dash-backend /opt/v2ray-dash/v2ray-dash-server && chmod +x /opt/v2ray-dash/v2ray-dash-server"
```

- [ ] **Step 3: 执行数据库迁移**

```bash
sshpass -p 'Jat02300920#AL' ssh root@112.125.93.190 "psql 'postgres://v2ray_dash:YourNewPass123@localhost:5432/v2ray_dash?sslmode=disable' -c \"ALTER TABLE subscription_accounts ADD COLUMN IF NOT EXISTS sort_order INT DEFAULT 0;\""
sshpass -p 'Jat02300920#AL' ssh root@112.125.93.190 "psql 'postgres://v2ray_dash:YourNewPass123@localhost:5432/v2ray_dash?sslmode=disable' -c \"ALTER TABLE subscription_accounts ADD COLUMN IF NOT EXISTS note VARCHAR(255);\""
```

- [ ] **Step 4: 重启服务**

```bash
sshpass -p 'Jat02300920#AL' ssh root@112.125.93.190 "systemctl restart v2ray-dash"
```

- [ ] **Step 5: 验证**

```bash
curl http://112.125.93.190:8080/api/subscriptions/full
```

---

## 自检清单

- [ ] 所有任务覆盖了 spec 中的需求
- [ ] 没有 "TBD"、"TODO" 等占位符
- [ ] 类型签名、方法名在前後任务间一致
- [ ] 测试命令可执行并有明确预期输出

---

**Plan complete and saved to `docs/superpowers/plans/2026-05-24-subscription-account-management-plan.md`**

两个执行选项：

**1. Subagent-Driven (recommended)** - I dispatch a fresh subagent per task, review between tasks, fast iteration

**2. Inline Execution** - Execute tasks in this session using executing-plans, batch execution with checkpoints

Which approach?
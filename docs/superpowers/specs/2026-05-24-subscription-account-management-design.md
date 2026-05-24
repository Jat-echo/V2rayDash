# 订阅账号动态管理设计

## 背景

用户场景：管理员希望不更换用户订阅链接的情况下，能够动态调整该订阅下关联的服务器账号组合。

核心问题：订阅 UUID 固定，但关联的账号（节点）可以灵活增删改。

## 设计方案

### 1. 订阅与账号关联关系

`subscription_accounts` 表作为多对多关联表，支持灵活的账号管理：

- **追加账号**：向订阅增加一个账号（INSERT）
- **移除账号**：从订阅移除一个账号（DELETE），账号本身不受影响
- **替换账号**：移除旧账号 + 添加新账号
- **排序**：通过 `sort_order` 字段决定节点顺序

### 2. 服务器删除时的级联行为

数据库外键约束已配置：

```
servers → accounts (ON DELETE CASCADE)
accounts → subscription_accounts (ON DELETE CASCADE)
```

删除服务器时：
1. 该服务器下的所有账号被删除
2. `subscription_accounts` 中对应的关联记录自动清除
3. 订阅本身保留，`server_id` 字段设为 NULL（已有 ON DELETE SET NULL）

### 3. 前端空订阅处理

订阅关联账号为空时，前端不报错，仅显示：
- 订阅状态正常
- 提示"暂无可用节点，请联系管理员"

### 4. API 设计

**账号管理接口**（已有基础，需完善排序功能）：

- `POST /api/subscriptions/:id/accounts` - 添加账号到订阅
- `DELETE /api/subscriptions/:id/accounts/:accountId` - 从订阅移除账号
- 替换和排序：通过 `PUT /api/subscriptions/:id` 的 `account_mappings` 参数实现

**订阅生成逻辑**（不变）：

- `GET /api/subscribe/:uuid` 读取当前关联账号，生成订阅配置

### 5. 数据库变更

无需新增表，扩展 `subscription_accounts` 表：

```sql
ALTER TABLE subscription_accounts ADD COLUMN sort_order INT DEFAULT 0;
ALTER TABLE subscription_accounts ADD COLUMN note VARCHAR(255);
```

排序字段用于控制用户端节点的显示顺序。

## 实现计划

1. 数据库迁移：添加 `sort_order` 和 `note` 字段
2. 后端：更新 `ReplaceAccounts` 方法支持排序参数
3. 前端：订阅详情页支持拖拽排序、增删账号
4. 前端：订阅列表页空账号时显示友好提示

## 优先级

P0：账号增删改能力、服务器删除时订阅不报错
P1：账号排序功能
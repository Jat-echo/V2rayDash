# 批量分配订阅 & 按钮图标化设计

**日期**: 2026-06-04  
**范围**: 服务器页操作按钮图标化 + 账号管理弹窗增加「关联订阅」Tab + 订阅页操作按钮图标化

## 背景

- 服务器列表操作栏现有 5 个文字按钮（安装、编辑、账号管理、重启服务、删除），排列拥挤
- 没有批量将某服务器分配给多个订阅的入口，只能逐个在订阅管理里手动操作
- 订阅页操作栏 4 个按钮中有 3 个是文字按钮，风格不统一（复制按钮已是图标）

## 目标

1. 服务器页和订阅页的操作按钮统一改为图标按钮，悬停显示 tooltip
2. 在「账号管理」弹窗中新增「关联订阅」Tab，支持批量将该服务器分配给多个订阅

## 不涉及后端改动

所有改动均为前端，后端接口已满足需求：
- `GET /api/subscriptions/full` → 所有订阅（含已关联账号列表，每个账号有 `server_id`）
- `GET /api/accounts?server_id=xxx`（`accountAPI.listByServer(id)`）→ 该服务器上所有账号
- `POST /api/subscriptions/:id/accounts`（`{ server_id, auto_create: true }`）→ 关联并自动创建账号

## 一、服务器页按钮图标化

### 图标映射

| 原按钮 | 图标 | tooltip | 样式 |
|---|---|---|---|
| 安装 | `CloudUploadOutlined` | 安装 Agent | 默认 primary |
| 编辑 | `EditOutlined` | 编辑 | 默认 |
| 账号管理 | `TeamOutlined` | 账号管理 | 默认 |
| 重启服务 | `ReloadOutlined` | 重启 xray | 默认，loading 态保留 |
| 删除 | `DeleteOutlined` | 删除 | danger |

所有按钮使用 `<Button size="small" icon={<Icon />} title="tooltip文字" />` 形式，无文字内容。

## 二、账号管理弹窗：增加「关联订阅」Tab

### 弹窗结构

```
Modal: 账号管理 · {server.name}
├── Tab 1: 账号          （现有功能，不改动）
└── Tab 2: 关联订阅       （新增）
```

### Tab 2「关联订阅」数据加载

打开 Tab 2（或切换至 Tab 2）时，并行拉取：
1. `subscriptionAPI.listFull()` → 所有订阅（含 `accounts: AccountWithServer[]`，每个账号有 `server_id`）
2. `accountAPI.listByServer(server.id)` → 该服务器上所有账号（含 `email`）

### 每行订阅的状态判断（前端计算）

| 状态 | 判断逻辑 | 显示 |
|---|---|---|
| **已分配** | `sub.accounts.some(a => a.server_id === server.id)` | Checkbox 禁用，Tag「已分配」(gray) |
| **将关联已有账号** | 服务器账号中有 `email === sub.name`，且未分配 | Checkbox 可选，Tag「已有账号 · {email}」(blue) |
| **将新建账号** | 以上两种都不满足 | Checkbox 可选，Tag「将新建 · {sub.name}」(default) |

### 交互细节

- 加载中：显示 Spin，不展示列表
- 列表上方：「全选」Checkbox（仅作用于可选项）
- 每行：`[Checkbox] 订阅名 · 备注（若有）` + 右侧状态 Tag
- Footer：
  - `取消`
  - `确认分配（已选 N 个）`（N=0 时禁用；执行中显示 loading）

### 分配执行

```
Promise.all(selectedSubIds.map(id =>
  subscriptionAPI.addAccount(id, { server_id: server.id, auto_create: true })
))
```

- 全部成功：`message.success('分配成功')`，刷新订阅状态，重置勾选
- 部分失败：`message.warning('N 个成功，M 个失败')`，不关闭弹窗，允许重试

## 三、订阅页按钮图标化

| 原按钮 | 图标 | tooltip | 样式 |
|---|---|---|---|
| 订阅链接 | `LinkOutlined` | 订阅链接 | primary |
| 复制（已图标）| `CopyOutlined` | 复制订阅链接 | 不变 |
| 编辑 | `EditOutlined` | 编辑 | 默认 |
| 删除 | `DeleteOutlined` | 删除 | danger |

## 涉及改动文件

| 文件 | 改动内容 |
|---|---|
| `frontend/src/pages/servers/index.tsx` | 操作按钮图标化；账号管理弹窗改为 Tabs；新增 Tab 2 关联订阅组件及状态 |
| `frontend/src/pages/subscriptions/index.tsx` | 操作按钮图标化 |

## 不需要改动

- 后端所有代码
- 订阅页其他功能（添加、编辑弹窗、流量图等）
- 服务器页 Tab 1 账号管理现有功能

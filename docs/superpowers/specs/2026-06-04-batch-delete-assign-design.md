# 批量删除关联订阅设计

**日期**: 2026-06-04  
**范围**: 服务器页「关联订阅」Tab 增加批量删除关联功能

## 背景

现有「关联订阅」Tab 只支持批量分配（将服务器关联到多个订阅）。当需要解除某服务器与订阅的关联时，没有入口，只能在订阅管理页逐个操作。

## 目标

在「关联订阅」Tab 中，将原来的单列布局改为**双列布局**：
- **左列「已关联」**：展示该服务器已分配给的订阅，支持批量选择并删除关联（同时删除账号）
- **右列「可分配」**：展示尚未分配的订阅，保留现有批量分配逻辑

## 不涉及后端改动

使用现有接口：
- `subscriptionAPI.removeAccount(subId, accountId)` — 解除订阅与账号的关联
- `accountAPI.delete(accountId)` — 删除账号本身

## UI 布局

```
┌──── 已关联 ─────────────────┬──── 可分配 ───────────────────┐
│ [全选] (N 个)               │ [全选] (M 个)                  │
│ [✓] 订阅A  · 备注  [已分配] │ [ ] 订阅B  [将新建 · B]        │
│ [✓] 订阅C          [已分配] │ [ ] 订阅D  [已有账号 · D]      │
│ ─────────────────────────── │ ──────────────────────────────  │
│ [删除关联（已选 N 个）]      │ [确认分配（已选 M 个）]         │
└─────────────────────────────┴────────────────────────────────┘
```

两列等宽，通过 flex 布局实现，中间加分隔线。每列内部独立滚动（maxHeight: 320px）。

## 删除流程

1. 用户在左列勾选一个或多个「已分配」订阅
2. 点击「删除关联（已选 N 个）」按钮
3. 弹出 `Popconfirm` 确认：「确定解除关联并删除这 N 个账号吗？此操作不可恢复。」
4. 确认后：
   - 对每个选中订阅，找到该服务器上的账号：`sub.accounts?.find(a => a.server_id === serverId)`
   - 并行调用：`subscriptionAPI.removeAccount(sub.id, acc.id)` + `accountAPI.delete(acc.id)`
   - 使用 `Promise.allSettled` 处理部分失败
5. 全部成功：`message.success('已删除 N 个关联')`，重置 `deleteSelected`，调用 `loadAssignData()`
6. 部分失败：`message.warning('N 个成功，M 个失败')`，调用 `loadAssignData()`

## 新增状态

```ts
const [deleteSelected, setDeleteSelected] = useState<string[]>([])
const [deleteSubmitting, setDeleteSubmitting] = useState(false)
```

onCancel 时同步清空：在现有清空逻辑中加入 `setDeleteSelected([])`

## 新增函数

```ts
const handleDeleteAssign = async () => {
  if (!selectedServerForAccounts || deleteSelected.length === 0) return
  setDeleteSubmitting(true)
  const serverId = selectedServerForAccounts.id
  try {
    const results = await Promise.allSettled(
      deleteSelected.map(subId => {
        const sub = assignSubs.find(s => s.id === subId)
        const acc = sub?.accounts?.find(a => a.server_id === serverId)
        if (!acc) return Promise.reject(new Error('account not found'))
        return Promise.all([
          subscriptionAPI.removeAccount(subId, acc.id),
          accountAPI.delete(acc.id),
        ])
      })
    )
    const succeeded = results.filter(r => r.status === 'fulfilled').length
    const failed = results.filter(r => r.status === 'rejected').length
    if (failed === 0) {
      message.success(`已删除 ${succeeded} 个关联`)
    } else {
      message.warning(`${succeeded} 个成功，${failed} 个失败`)
    }
    setDeleteSelected([])
    await loadAssignData()
  } finally {
    setDeleteSubmitting(false)
  }
}
```

## 涉及改动文件

| 文件 | 改动 |
|---|---|
| `frontend/src/pages/servers/index.tsx` | 新增状态变量、`handleDeleteAssign` 函数；「关联订阅」Tab JSX 改为双列布局 |

## 不需要改动

- 后端代码
- 其他前端页面
- Tab 1「账号」功能

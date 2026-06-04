# xray 同步按钮 & 批量分配串行化设计

**日期**: 2026-06-04  
**范围**: 解决批量分配订阅时 xray 账号未完整同步到远程服务器的问题

## 背景与根因

批量分配 22 个订阅时，前端用 `Promise.allSettled` 并发发送 22 个 `AddAccount` 请求。后端每次 `AddAccount` 都异步启动一个 `go h.accountSvc.SyncAllToRemote(...)` goroutine。22 个 goroutine 同时竞争 SSH 连接和 xray 配置文件写入，导致最终 xray 配置只包含部分账号（19/22）。

## 目标

1. 新增手动「同步 xray」按钮，用于修复已有不一致
2. 批量分配改为串行 + 最后统一同步，从根本上消除竞争

## 改动一：手动同步 xray 按钮

### 后端

新增 `POST /api/servers/:id/sync-xray` 端点，在 `server.go` 中实现：

```go
func (h *ServerHandler) SyncXray(c *gin.Context) {
    id := c.Param("id")
    server, err := h.repo.GetByIDForInstall(id)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "server not found"})
        return
    }
    var auth ssh.SSHAuth
    if server.SSHKeyType == "password" {
        auth = &ssh.PasswordAuth{Password: server.SSHPassword}
    } else {
        auth = &ssh.KeyAuth{PrivateKey: server.SSHKey}
    }
    if err := h.accountSvc.SyncAllToRemote(id, auth); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("同步失败: %v", err)})
        return
    }
    c.JSON(http.StatusOK, gin.H{"message": "同步成功"})
}
```

关键点：`SyncAllToRemote` **同步调用**（非 goroutine），明确返回成功/失败。

`SyncXray` 需要访问 `accountSvc`，因此 `ServerHandler` 需要持有 `AccountService` 的引用（当前不持有，需要在构造函数中注入）。

在 `routes.go` 注册路由：
```go
api.POST("/servers/:id/sync-xray", serverHandler.SyncXray)
```

### 前端

`services/api.ts` 新增：
```ts
syncXray: (id: string) => api.post(`/servers/${id}/sync-xray`).then(r => r.data),
```

`servers/index.tsx` 操作栏新增第 6 个图标按钮（`SyncOutlined`）：
- title: `"同步 xray 配置"`
- 有独立 `syncingXray: string | null` loading 状态（与现有 `restartingXray` 对称）
- 成功提示 `message.success('xray 配置同步成功')`，失败提示错误信息

## 改动二：批量分配串行化 + 自动同步

### 前端 handleAssign 修改

将 `Promise.allSettled` 并发改为 `for...of` 串行循环：

```ts
const handleAssign = async () => {
  if (!selectedServerForAccounts || assignSelected.length === 0) return
  setAssignSubmitting(true)
  let succeeded = 0, failed = 0
  try {
    for (const id of assignSelected) {
      try {
        await subscriptionAPI.addAccount(id, {
          server_id: selectedServerForAccounts.id,
          auto_create: true,
        })
        succeeded++
      } catch {
        failed++
      }
    }
    if (failed === 0) {
      message.success(`成功分配 ${succeeded} 个订阅`)
    } else {
      message.warning(`${succeeded} 个成功，${failed} 个失败`)
    }
    // 所有 DB 操作完成后统一同步一次 xray
    try {
      await serverAPI.syncXray(selectedServerForAccounts.id)
    } catch {
      message.warning('xray 同步失败，请手动点击同步按钮')
    }
    setAssignSelected([])
    await loadAssignData()
  } finally {
    setAssignSubmitting(false)
  }
}
```

串行的好处：
- 消除并发 SSH goroutine 竞争（每个 AddAccount 触发的 goroutine 在下一个请求发出前已大概率完成）
- 最后的 `syncXray` 调用确保完整同步（同步执行，明确成功/失败）

## 涉及改动文件

| 文件 | 改动 |
|---|---|
| `backend/internal/handler/server.go` | 新增 `SyncXray` handler；`ServerHandler` 持有 `AccountService` |
| `backend/internal/handler/routes.go` | 注册 `POST /servers/:id/sync-xray` |
| `frontend/src/services/api.ts` | 新增 `serverAPI.syncXray` |
| `frontend/src/pages/servers/index.tsx` | 新增 `SyncOutlined` 按钮；修改 `handleAssign` 为串行 + 自动同步 |

## 不需要改动

- `AddAccount` 后端逻辑（仍保留 goroutine sync，用于单次手动添加账号的场景）
- 其他前端页面
- 数据库 schema

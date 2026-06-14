# 服务器延迟显示功能设计

**日期：** 2026-06-14  
**状态：** 已批准，待实现

## 功能概述

在服务器列表页面新增延迟（ping）显示列，通过 TCP 握手方式测量控制中心到各服务器 SSH 端口的往返时间，支持页面加载自动测量、60 秒定时轮询以及手动批量重测。

## 后端设计

### 新增接口

`POST /api/servers/ping`

**处理器：** `handler/server.go` → `PingAll` 方法

**逻辑：**
1. 从数据库查出所有服务器的 `id`、`ip`、`ssh_port`
2. 用 `sync.WaitGroup` + goroutine 并发对每台服务器发起 TCP Dial（`ip:ssh_port`，超时 5 秒）
3. Dial 成功则记录耗时（毫秒）；连接失败或超时返回 `-1`
4. 所有 goroutine 完成后返回结果

**响应格式：**
```json
{
  "results": [
    { "server_id": "abc123", "latency_ms": 23 },
    { "server_id": "def456", "latency_ms": -1 }
  ]
}
```

**约束：**
- 不修改数据库，不持久化延迟数据
- 不修改 Server 模型
- 使用 `net.DialTimeout`，无需特殊系统权限

### 路由注册

在 `handler/routes.go` 中已有鉴权中间件的路由组下添加：
```
POST /servers/ping → serverHandler.PingAll
```

## 前端设计

### 文件：`frontend/src/services/api.ts`

在 `serverAPI` 对象中新增：
```ts
pingAll: () => api.post<{ results: { server_id: string; latency_ms: number }[] }>('/servers/ping').then(r => r.data),
```

### 文件：`frontend/src/pages/servers/index.tsx`

**新增 State：**
- `latencies: Map<string, number>` — server_id → 延迟 ms，-1 表示超时
- `pinging: boolean` — 控制测速按钮 loading 状态

**新增函数 `pingAll()`：**
```
async function pingAll() {
  setpinging(true)
  const data = await serverAPI.pingAll()
  const map = new Map(data.results.map(r => [r.server_id, r.latency_ms]))
  setLatencies(map)
  setPinging(false)
}
```

**生命周期：**
- `useEffect` 页面加载调用一次 `pingAll()`
- `setInterval(pingAll, 60_000)` 每 60 秒轮询
- 组件卸载时 `clearInterval`

**UI 变更：**
- 顶部操作区在"+ 添加服务器"旁新增"测速"按钮，`pinging` 为 true 时 loading
- 表格"状态"列后新增"延迟"列，渲染规则：

| 值 | 显示 | 颜色 |
|---|---|---|
| 未测量（Map 中无此 ID） | `—` | 默认灰 |
| `-1` | `超时` | 红色 Tag |
| `0–100 ms` | `Xms` | 绿色 Tag |
| `101–200 ms` | `Xms` | 橙色 Tag |
| `> 200 ms` | `Xms` | 红色 Tag |

## 不在本次范围内

- 延迟历史趋势图
- 单台服务器独立测速
- 延迟数据持久化
- Agent 侧反向上报延迟

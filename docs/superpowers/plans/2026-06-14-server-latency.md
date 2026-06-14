# Server Latency Display Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在服务器列表新增"延迟"列，通过后端并发 TCP 握手测量各服务器 SSH 端口延迟，支持页面加载自动测量、60 秒轮询和手动批量重测。

**Architecture:** 新增 `POST /api/servers/ping` 后端接口，用 goroutine 并发 `net.DialTimeout` 测量所有服务器的 TCP 握手时间，返回 `{server_id, latency_ms}` 列表（-1 表示超时）。前端在 `serverAPI` 添加调用方法，组件内维护 `latencies` state 并用 `useInterval` 轮询，表格新增彩色 Tag 列展示延迟。

**Tech Stack:** Go 1.21 + Gin, React 18 + TypeScript + Ant Design 5, `net.DialTimeout`（标准库，无额外依赖）

---

## 文件清单

| 文件 | 操作 | 说明 |
|---|---|---|
| `backend/internal/handler/server.go` | 修改 | 新增 `PingAll` 方法 |
| `backend/internal/handler/routes.go` | 修改 | 注册 `POST /servers/ping` 路由 |
| `frontend/src/services/api.ts` | 修改 | `serverAPI` 添加 `pingAll` 方法 |
| `frontend/src/pages/servers/index.tsx` | 修改 | 新增 state、轮询逻辑、测速按钮、延迟列 |

---

### Task 1: 后端 — 添加 PingAll 处理器

**Files:**
- Modify: `backend/internal/handler/server.go`
- Modify: `backend/internal/handler/routes.go`

- [ ] **Step 1: 在 `server.go` 末尾添加 PingAll 方法**

在 `backend/internal/handler/server.go` 的 `import` 块中加入 `"net"`, `"sync"`, `"time"` 三个包（`"fmt"`, `"io"`, `"net/http"` 已存在），然后在文件末尾追加：

```go
type PingResult struct {
	ServerID  string `json:"server_id"`
	LatencyMs int64  `json:"latency_ms"`
}

func (h *ServerHandler) PingAll(c *gin.Context) {
	servers, err := h.repo.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	results := make([]PingResult, len(servers))
	var wg sync.WaitGroup
	for i, s := range servers {
		wg.Add(1)
		go func(idx int, id, ip string, port int) {
			defer wg.Done()
			addr := fmt.Sprintf("%s:%d", ip, port)
			start := time.Now()
			conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
			if err != nil {
				results[idx] = PingResult{ServerID: id, LatencyMs: -1}
				return
			}
			conn.Close()
			results[idx] = PingResult{ServerID: id, LatencyMs: time.Since(start).Milliseconds()}
		}(i, s.ID, s.IP, s.SSHPort)
	}
	wg.Wait()

	c.JSON(http.StatusOK, gin.H{"results": results})
}
```

- [ ] **Step 2: 注册路由**

在 `backend/internal/handler/routes.go` 的服务器管理路由块（`api.POST("/servers/:id/sync-xray", ...)` 之后）添加一行：

```go
api.POST("/servers/ping", serverHandler.PingAll)
```

注意：此行必须在 `api.GET("/servers/:id", ...)` 之前或同一 `serverHandler` 实例下，且 `/servers/ping` 要放在 `/servers/:id` 系列路由**之前**，否则 Gin 会将 `ping` 识别为 `:id`。

检查当前路由顺序：

```
api.GET("/servers", serverHandler.List)
api.POST("/servers", serverHandler.Create)
api.POST("/servers/ping", serverHandler.PingAll)   ← 插入这里
api.GET("/servers/:id", serverHandler.Get)
api.PUT("/servers/:id", serverHandler.Update)
api.DELETE("/servers/:id", serverHandler.Delete)
api.POST("/servers/:id/restart-xray", serverHandler.RestartXray)
api.POST("/servers/:id/sync-xray", serverHandler.SyncXray)
```

- [ ] **Step 3: 验证编译通过**

```bash
cd /home/jat-id/Project/V2rayDash/backend && go build ./...
```

预期输出：无报错，无输出。

- [ ] **Step 4: 提交**

```bash
git add backend/internal/handler/server.go backend/internal/handler/routes.go
git commit -m "feat(backend): add POST /servers/ping for TCP latency measurement"
```

---

### Task 2: 前端 API — 添加 pingAll 方法

**Files:**
- Modify: `frontend/src/services/api.ts`

- [ ] **Step 1: 在 `serverAPI` 对象中添加 `pingAll`**

在 `frontend/src/services/api.ts` 中找到 `export const serverAPI = {` 块，在 `syncXray` 一行后追加：

```ts
  pingAll: () =>
    api.post<{ results: { server_id: string; latency_ms: number }[] }>('/servers/ping').then(r => r.data),
```

最终 `serverAPI` 结构：

```ts
export const serverAPI = {
  list: () => api.get<Server[]>('/servers').then(r => r.data),
  get: (id: string) => api.get<Server>(`/servers/${id}`).then(r => r.data),
  create: (data: Partial<Server>) => api.post<Server>('/servers', data).then(r => r.data),
  update: (id: string, data: Partial<Server>) => api.put(`/servers/${id}`, data),
  delete: (id: string) => api.delete(`/servers/${id}`),
  restartXray: (id: string) => api.post(`/servers/${id}/restart-xray`).then(r => r.data),
  syncXray: (id: string) => api.post(`/servers/${id}/sync-xray`).then(r => r.data),
  pingAll: () =>
    api.post<{ results: { server_id: string; latency_ms: number }[] }>('/servers/ping').then(r => r.data),
}
```

- [ ] **Step 2: 验证 TypeScript 编译**

```bash
cd /home/jat-id/Project/V2rayDash/frontend && npx tsc --noEmit
```

预期输出：无报错。

- [ ] **Step 3: 提交**

```bash
git add frontend/src/services/api.ts
git commit -m "feat(api): add pingAll to serverAPI"
```

---

### Task 3: 前端 UI — 延迟 state、轮询、按钮、表格列

**Files:**
- Modify: `frontend/src/pages/servers/index.tsx`

- [ ] **Step 1: 添加 latencies 和 pinging state**

在 `servers/index.tsx` 中已有的 state 声明区域（`const [deleteSubmitting, setDeleteSubmitting] = useState(false)` 之后）添加：

```tsx
const [latencies, setLatencies] = useState<Map<string, number>>(new Map())
const [pinging, setPinging] = useState(false)
```

- [ ] **Step 2: 添加 pingAll 函数**

在 `loadServers` 函数之后，`loadAssignData` 之前添加：

```tsx
const pingAll = async () => {
  if (pinging) return
  setPinging(true)
  try {
    const data = await serverAPI.pingAll()
    setLatencies(new Map(data.results.map(r => [r.server_id, r.latency_ms])))
  } catch {
    // 静默失败，不打断主流程
  } finally {
    setPinging(false)
  }
}
```

- [ ] **Step 3: 添加 useEffect 实现页面加载自动测量 + 60 秒轮询**

在已有的 `useEffect(() => { loadServers() }, [timeRange])` 之后添加：

```tsx
useEffect(() => {
  pingAll()
  const timer = setInterval(pingAll, 60_000)
  return () => clearInterval(timer)
}, [])
```

注意：空依赖数组确保只在挂载时启动一次，卸载时清除定时器。

- [ ] **Step 4: 在顶部添加"测速"按钮**

找到页面头部的"+ 添加服务器"按钮：

```tsx
<Button type="primary" className="page-action" onClick={() => setModalVisible(true)}>+ 添加服务器</Button>
```

在该按钮前加入测速按钮（用 Space 包裹两个按钮）：

```tsx
<Space>
  <Button
    loading={pinging}
    onClick={pingAll}
    icon={<ThunderboltOutlined />}
  >
    测速
  </Button>
  <Button type="primary" className="page-action" onClick={() => setModalVisible(true)}>+ 添加服务器</Button>
</Space>
```

同时在文件顶部的 Ant Design import 中添加 `ThunderboltOutlined`：

```tsx
import { ..., ThunderboltOutlined } from '@ant-design/icons'
```

- [ ] **Step 5: 在 columns 数组中添加"延迟"列**

在 `columns` 数组中，找到 `{ title: '状态', dataIndex: 'status' }` 这一列，在其**后面**插入延迟列：

```tsx
{
  title: '延迟',
  render: (_: any, record: Server) => {
    if (!latencies.has(record.id)) return <span style={{ color: 'var(--text-muted)' }}>—</span>
    const ms = latencies.get(record.id)!
    if (ms === -1) return <Tag color="red">超时</Tag>
    if (ms <= 100) return <Tag color="green">{ms}ms</Tag>
    if (ms <= 200) return <Tag color="orange">{ms}ms</Tag>
    return <Tag color="red">{ms}ms</Tag>
  },
},
```

- [ ] **Step 6: 验证 TypeScript 编译**

```bash
cd /home/jat-id/Project/V2rayDash/frontend && npx tsc --noEmit
```

预期输出：无报错。

- [ ] **Step 7: 构建前端产物**

```bash
cd /home/jat-id/Project/V2rayDash/frontend && npm run build
```

预期输出：`dist/index.html` 等文件更新，无编译错误。

- [ ] **Step 8: 提交**

```bash
git add frontend/src/pages/servers/index.tsx
git commit -m "feat(servers): add latency column with TCP ping, auto-poll every 60s"
```

---

## 手动验证清单

功能完成后，在浏览器中验证：

1. 打开服务器页面，表格"延迟"列初始显示 `—`，约 5 秒内自动更新为实际延迟值
2. 点击"测速"按钮，按钮显示 loading，所有服务器延迟更新
3. 等待 60 秒，延迟值自动刷新（可通过 DevTools Network 面板确认 POST /api/servers/ping 请求）
4. 延迟 ≤100ms 显示绿色 Tag，101-200ms 橙色，>200ms 红色，超时显示红色"超时"
5. 测速期间点击"测速"按钮无重复请求（`pinging` 守卫生效）

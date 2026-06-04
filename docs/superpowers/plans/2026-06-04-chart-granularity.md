# Chart Granularity & Data Cleanup Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 不同时间窗口使用不同 SQL 分桶粒度返回固定数量级数据点，并定时清理超过 30 天的历史数据。

**Architecture:** 在 PostgreSQL 查询层用 `to_timestamp(floor(epoch/$bucket)*$bucket)` + `GROUP BY` 实现任意粒度分桶，不改表结构。后台 goroutine 每 6 小时删除三张时序表中超过 30 天的行。前端服务器页面时间窗口选项与订阅流量页面对齐。

**Tech Stack:** Go 1.21+, PostgreSQL (pgx/pq driver), React + TypeScript + Ant Design

---

## 文件变动一览

| 操作 | 文件 | 说明 |
|---|---|---|
| Modify | `backend/internal/repository/helpers.go` | 新增 `timeRangeToBucketSeconds()` |
| Create | `backend/internal/repository/helpers_test.go` | 对应单元测试 |
| Modify | `backend/internal/repository/subscription.go` | `GetAccountTrafficLogs` 改用分桶 SQL |
| Modify | `backend/internal/repository/log.go` | `GetNodeStatusesByTimeRange` 改用分桶 SQL |
| Modify | `backend/cmd/server/main.go` | 启动清理 goroutine |
| Modify | `frontend/src/pages/servers/index.tsx` | TIME_RANGES 改为 1h/3h/6h/1d/7d/30d |

---

## Task 1: 新增 `timeRangeToBucketSeconds()`

**Files:**
- Modify: `backend/internal/repository/helpers.go`
- Create: `backend/internal/repository/helpers_test.go`

- [ ] **Step 1: 在 helpers.go 末尾追加新函数**

```go
// timeRangeToBucketSeconds maps a time range key to bucket size in seconds
// for use in SQL time-bucketing queries.
func timeRangeToBucketSeconds(timeRange string) int {
	m := map[string]int{
		"1h":  60,
		"3h":  120,
		"6h":  300,
		"1d":  600,
		"7d":  7200,
		"30d": 21600,
	}
	if v, ok := m[timeRange]; ok {
		return v
	}
	return 60
}
```

- [ ] **Step 2: 创建测试文件**

新建 `backend/internal/repository/helpers_test.go`，内容：

```go
package repository

import "testing"

func TestTimeRangeToBucketSeconds(t *testing.T) {
	cases := []struct {
		input string
		want  int
	}{
		{"1h", 60},
		{"3h", 120},
		{"6h", 300},
		{"1d", 600},
		{"7d", 7200},
		{"30d", 21600},
		{"unknown", 60},
		{"", 60},
	}
	for _, c := range cases {
		got := timeRangeToBucketSeconds(c.input)
		if got != c.want {
			t.Errorf("timeRangeToBucketSeconds(%q) = %d, want %d", c.input, got, c.want)
		}
	}
}
```

- [ ] **Step 3: 运行测试确认通过**

```bash
cd /home/jat-id/Project/V2rayDash/backend
go test ./internal/repository/ -run TestTimeRangeToBucketSeconds -v
```

预期输出：
```
--- PASS: TestTimeRangeToBucketSeconds (0.00s)
PASS
```

- [ ] **Step 4: Commit**

```bash
git add backend/internal/repository/helpers.go backend/internal/repository/helpers_test.go
git commit -m "feat(repository): add timeRangeToBucketSeconds for query-time bucketing"
```

---

## Task 2: 订阅流量查询改用分桶 SQL

**Files:**
- Modify: `backend/internal/repository/subscription.go:394-436`

目标：`GetAccountTrafficLogs` 按时间窗口分桶，每个 bucket 内 account 取 `MAX(traffic_bytes)`（累计值），返回行数大幅减少。

- [ ] **Step 1: 替换 `GetAccountTrafficLogs` 函数体**

找到当前实现（约第 394 行），将整个函数替换为：

```go
// GetAccountTrafficLogs returns per-account cumulative traffic snapshots for a subscription,
// bucketed by time range to limit returned point count.
func (r *SubscriptionRepository) GetAccountTrafficLogs(subscriptionID, timeRange string) ([]model.AccountTrafficSeries, error) {
	interval := timeRangeToInterval(timeRange, "1 day")
	bucket := timeRangeToBucketSeconds(timeRange)

	rows, err := r.db.Query(`
		SELECT
		  a.id,
		  a.email,
		  srv.name,
		  MAX(atl.traffic_bytes) AS traffic_bytes,
		  to_timestamp(floor(extract(epoch from atl.recorded_at) / $3) * $3) AS bucket_time
		FROM subscription_accounts sa
		JOIN accounts a ON sa.account_id = a.id
		JOIN servers srv ON a.server_id = srv.id
		JOIN account_traffic_logs atl ON atl.account_id = a.id
		WHERE sa.subscription_id = $1
		  AND atl.recorded_at > NOW() - $2::interval
		GROUP BY a.id, a.email, srv.name,
		         to_timestamp(floor(extract(epoch from atl.recorded_at) / $3) * $3)
		ORDER BY a.id, bucket_time ASC
	`, subscriptionID, interval, bucket)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	seriesMap := make(map[string]*model.AccountTrafficSeries)
	var order []string
	for rows.Next() {
		var accID, email, serverName string
		var bytes int64
		var t time.Time
		if err := rows.Scan(&accID, &email, &serverName, &bytes, &t); err != nil {
			return nil, err
		}
		if _, ok := seriesMap[accID]; !ok {
			seriesMap[accID] = &model.AccountTrafficSeries{
				AccountID:  accID,
				Email:      email,
				ServerName: serverName,
				Points:     []model.BandwidthPoint{},
			}
			order = append(order, accID)
		}
		seriesMap[accID].Points = append(seriesMap[accID].Points, model.BandwidthPoint{Time: t, Value: bytes})
	}

	result := make([]model.AccountTrafficSeries, 0, len(order))
	for _, id := range order {
		result = append(result, *seriesMap[id])
	}
	return result, nil
}
```

- [ ] **Step 2: 确认编译通过**

```bash
cd /home/jat-id/Project/V2rayDash/backend
go build ./...
```

预期：无报错输出。

- [ ] **Step 3: Commit**

```bash
git add backend/internal/repository/subscription.go
git commit -m "feat(repository): bucket GetAccountTrafficLogs by time range"
```

---

## Task 3: 服务器监控查询改用分桶 SQL

**Files:**
- Modify: `backend/internal/repository/log.go:108-137`

目标：`GetNodeStatusesByTimeRange` 按分桶聚合，CPU/内存/磁盘/带宽均取 `MAX`。因为是 GROUP BY 聚合查询，返回行中不再有 `id` 字段，scan 顺序需同步调整。

- [ ] **Step 1: 替换 `GetNodeStatusesByTimeRange` 函数体**

找到当前实现（约第 108 行），将整个函数替换为：

```go
func (r *LogRepository) GetNodeStatusesByTimeRange(serverID string, timeRange string) ([]*model.NodeStatus, error) {
	interval := timeRangeToInterval(timeRange, "1 hour")
	bucket := timeRangeToBucketSeconds(timeRange)

	query := `
		SELECT
		  server_id,
		  to_timestamp(floor(extract(epoch from reported_at) / $3) * $3) AS bucket_time,
		  MAX(cpu_percent)    AS cpu_percent,
		  MAX(memory_percent) AS memory_percent,
		  MAX(disk_percent)   AS disk_percent,
		  MAX(bandwidth_in)   AS bandwidth_in,
		  MAX(bandwidth_out)  AS bandwidth_out,
		  MAX(v2ray_status)   AS v2ray_status
		FROM node_status
		WHERE reported_at > NOW() - $2::interval
		  AND ($1 = '' OR server_id = $1::uuid)
		GROUP BY server_id,
		         to_timestamp(floor(extract(epoch from reported_at) / $3) * $3)
		ORDER BY server_id, bucket_time ASC
	`

	rows, err := r.db.Query(query, serverID, interval, bucket)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var statuses []*model.NodeStatus
	for rows.Next() {
		var s model.NodeStatus
		err := rows.Scan(
			&s.ServerID, &s.ReportedAt,
			&s.CPUPercent, &s.MemoryPercent, &s.DiskPercent,
			&s.BandwidthIn, &s.BandwidthOut, &s.V2rayStatus,
		)
		if err != nil {
			return nil, err
		}
		statuses = append(statuses, &s)
	}

	return statuses, nil
}
```

注意：scan 中去掉了 `&s.ID`（聚合查询无 id 列）；`buildNodeStatusResponse` 不使用 `s.ID`，无需改动。

- [ ] **Step 2: 确认编译通过**

```bash
cd /home/jat-id/Project/V2rayDash/backend
go build ./...
```

预期：无报错输出。

- [ ] **Step 3: Commit**

```bash
git add backend/internal/repository/log.go
git commit -m "feat(repository): bucket GetNodeStatusesByTimeRange by time range"
```

---

## Task 4: 启动后台清理 goroutine

**Files:**
- Modify: `backend/cmd/server/main.go`

- [ ] **Step 1: 在 main.go 中添加 `startCleanupWorker` 和 `cleanOldData`**

在 `main()` 函数定义之前添加：

```go
func startCleanupWorker(db *database.DB) {
	go func() {
		cleanOldData(db)
		ticker := time.NewTicker(6 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			cleanOldData(db)
		}
	}()
}

func cleanOldData(db *database.DB) {
	db.Exec(`DELETE FROM account_traffic_logs      WHERE recorded_at < NOW() - INTERVAL '30 days'`)
	db.Exec(`DELETE FROM node_status               WHERE reported_at < NOW() - INTERVAL '30 days'`)
	db.Exec(`DELETE FROM subscription_traffic_logs WHERE recorded_at < NOW() - INTERVAL '30 days'`)
}
```

- [ ] **Step 2: 在 `main()` 中调用 `startCleanupWorker`**

在 `database.InitSchema(db)` 调用之后、`handler.SetupRoutes` 之前加一行：

```go
startCleanupWorker(db)
```

完整插入位置示例（`main.go` 约第 32 行附近）：

```go
if err := database.InitSchema(db); err != nil {
    log.Fatalf("Failed to init schema: %v", err)
}

startCleanupWorker(db)   // ← 新增

r := gin.Default()
handler.SetupRoutes(r, db, cfg)
```

- [ ] **Step 3: 确认 `time` 包已在 import 中**

`main.go` 已 import `"time"`（用于 `time.WithTimeout`），无需新增。

- [ ] **Step 4: 确认编译通过**

```bash
cd /home/jat-id/Project/V2rayDash/backend
go build ./cmd/server/
```

预期：无报错输出。

- [ ] **Step 5: Commit**

```bash
git add backend/cmd/server/main.go
git commit -m "feat(server): start background cleanup worker to purge data older than 30 days"
```

---

## Task 5: 前端服务器页面时间窗口对齐

**Files:**
- Modify: `frontend/src/pages/servers/index.tsx:84-91`

- [ ] **Step 1: 替换 `TIME_RANGES` 常量**

找到约第 84 行的 `TIME_RANGES`：

```ts
const TIME_RANGES = [
  { label: '1H', value: '1h' },
  { label: '3H', value: '3h' },
  { label: '6H', value: '6h' },
  { label: '12H', value: '12h' },
  { label: '1天', value: '1d' },
  { label: '3天', value: '3d' },
]
```

替换为：

```ts
const TIME_RANGES = [
  { label: '1小时', value: '1h' },
  { label: '3小时', value: '3h' },
  { label: '6小时', value: '6h' },
  { label: '1天', value: '1d' },
  { label: '7天', value: '7d' },
  { label: '30天', value: '30d' },
]
```

- [ ] **Step 2: 确认 `timeRange` 初始值仍合法**

第 80 行：`const [timeRange, setTimeRange] = useState('1h')`，`'1h'` 仍在新列表中，无需改动。

- [ ] **Step 3: 前端编译确认**

```bash
cd /home/jat-id/Project/V2rayDash/frontend
npm run build 2>&1 | tail -5
```

预期末尾输出 `built in` 字样，无 TypeScript 错误。

- [ ] **Step 4: Commit**

```bash
git add frontend/src/pages/servers/index.tsx
git commit -m "feat(frontend): align server monitor time ranges with subscription chart (1h/3h/6h/1d/7d/30d)"
```

---

## 验收检查

完成所有任务后，启动后端并用浏览器验证：

1. **订阅流量图**：展开任一订阅行，切换 7d / 30d，确认返回点数明显少于切换 1h 时（可在 DevTools Network 面板查看响应体大小）。
2. **服务器监控图**：服务器页面时间窗口出现 7天 / 30天 选项，切换后图表正常渲染。
3. **数据清理**：可临时将 `INTERVAL '30 days'` 改为 `INTERVAL '1 minute'` 启动后观察日志，确认 DELETE 执行后数据被清除（验证后改回）。

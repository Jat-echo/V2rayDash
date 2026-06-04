# 图表数据颗粒度与历史数据清理设计

**日期**: 2026-06-04  
**范围**: 订阅流量曲线 + 服务器监控图表

## 背景

当前查询不做任何聚合，直接返回全部原始行（心跳间隔 30 秒）：
- 1 天查询 = 2,880 行/账号
- 7 天查询 = 20,160 行/账号
- 30 天查询 = 86,400 行/账号

长时间窗口下数据量过大，导致查询、传输、渲染三处无谓消耗。同时没有任何清理逻辑，数据无限累积。

## 目标

1. 不同时间窗口返回固定数量级的数据点（60–150 点），减少传输和渲染开销。
2. 每 6 小时自动清理超过 30 天的历史数据。

## 统一分桶规则

订阅流量图表与服务器监控图表使用相同的时间窗口和分桶大小：

| 时间窗口 | 分桶大小 | 预期返回点数 |
|---|---|---|
| 1h  | 60s（1 分钟）   | ~60  |
| 3h  | 120s（2 分钟）  | ~90  |
| 6h  | 300s（5 分钟）  | ~72  |
| 1d  | 600s（10 分钟） | ~144 |
| 7d  | 7200s（2 小时） | ~84  |
| 30d | 21600s（6 小时）| ~120 |

SQL 分桶表达式（适用于任意分桶秒数 `$bucket`）：
```sql
to_timestamp(floor(extract(epoch from <time_col>) / $bucket) * $bucket) AS bucket_time
```

## 聚合策略

### 订阅流量（account_traffic_logs）

`traffic_bytes` 是累计值（递增快照），分桶内取 `MAX`，返回每个 bucket 的峰值累计量。  
前端 delta 计算逻辑（`i === 0 ? 0 : Math.max(0, p.value - prev.value)`）**不需要改动**，自动适配分桶数据。

```sql
SELECT
  to_timestamp(floor(extract(epoch from atl.recorded_at) / $bucket) * $bucket) AS bucket_time,
  MAX(atl.traffic_bytes) AS traffic_bytes
FROM subscription_accounts sa
JOIN accounts a ON sa.account_id = a.id
JOIN servers srv ON a.server_id = srv.id
JOIN account_traffic_logs atl ON atl.account_id = a.id
WHERE sa.subscription_id = $1
  AND atl.recorded_at > NOW() - $interval::interval
GROUP BY a.id, a.email, srv.name, bucket_time
ORDER BY a.id, bucket_time ASC
```

### 服务器监控（node_status）

- CPU / 内存 / 磁盘：`MAX`（保留 bucket 内峰值，监控场景下比 AVG 更有价值）
- 带宽累计值：`MAX`（与原始数据语义一致，后端 `buildNodeStatusResponse` 的 delta 计算逻辑**不需要改动**）

```sql
SELECT
  to_timestamp(floor(extract(epoch from reported_at) / $bucket) * $bucket) AS bucket_time,
  server_id,
  MAX(cpu_percent)    AS cpu_percent,
  MAX(memory_percent) AS memory_percent,
  MAX(disk_percent)   AS disk_percent,
  MAX(bandwidth_in)   AS bandwidth_in,
  MAX(bandwidth_out)  AS bandwidth_out,
  MAX(v2ray_status)   AS v2ray_status
FROM node_status
WHERE reported_at > NOW() - $interval::interval
  AND ($server_id = '' OR server_id = $server_id::uuid)
GROUP BY server_id, bucket_time
ORDER BY server_id, bucket_time ASC
```

> `v2ray_status` 取 MAX 仅用于向后兼容，`V2rayRestarts` 计数在分桶后仍通过状态转换检测（stopped→running）得出，逻辑不变。

## 数据清理

在 `cmd/server/main.go` 启动时开启一个后台 goroutine：

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
    db.Exec(`DELETE FROM account_traffic_logs       WHERE recorded_at < NOW() - INTERVAL '30 days'`)
    db.Exec(`DELETE FROM node_status                WHERE reported_at < NOW() - INTERVAL '30 days'`)
    db.Exec(`DELETE FROM subscription_traffic_logs  WHERE recorded_at < NOW() - INTERVAL '30 days'`)
}
```

- 启动时立即执行一次（覆盖已有积压数据）
- 之后每 6 小时执行一次
- 错误静默忽略（非关键路径）

## 涉及改动文件

| 文件 | 改动内容 |
|---|---|
| `backend/internal/repository/helpers.go` | 新增 `timeRangeToBucketSeconds()` 映射表 |
| `backend/internal/repository/subscription.go` | `GetAccountTrafficLogs` 改用分桶 SQL |
| `backend/internal/repository/log.go` | `GetNodeStatusesByTimeRange` 改用分桶 SQL |
| `backend/internal/handler/log.go` | 无需改动（delta 计算自动适配） |
| `backend/cmd/server/main.go` | 启动 `startCleanupWorker` |
| `frontend/src/pages/monitor/index.tsx` | 时间窗口选项改为 1h/3h/6h/1d/7d/30d |

## 不需要改动

- `frontend/src/pages/subscriptions/index.tsx`：时间窗口选项和 delta 计算已与设计一致
- 数据库表结构：无 schema 变更，只改查询
- 前端图表渲染逻辑：时间轴标签格式（`fmtTime`）已按实际 span 自适应

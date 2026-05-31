# 监控中心图表化实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将监控中心从实时数值显示改为时间范围曲线图展示，并修复入站/出站带宽统计问题

**Architecture:** Agent 采集带宽数据上报控制中心，控制中心存储历史记录，前端通过时间范围参数请求数据并渲染曲线图。带宽差值在后端计算后返回给前端。

**Tech Stack:** Go (Agent + Backend), React + Ant Design Charts (Frontend), PostgreSQL

---

## 文件结构

### Agent 改动
- Modify: `agent/internal/collector/collector.go` - 添加 getBandwidth() 函数
- Modify: `agent/internal/model/model.go` - 确保 BandwidthIn/BandwidthOut 字段存在

### Backend 改动
- Modify: `backend/internal/handler/log.go` - 扩展 ListNodeStatuses 支持时间范围过滤和带宽差值计算
- Modify: `backend/internal/repository/log.go` - 添加时间范围查询方法

### Frontend 改动
- Modify: `frontend/src/pages/monitor/index.tsx` - 重构为图表展示
- Modify: `frontend/src/services/api.ts` - 添加时间范围参数和新的响应类型
- Install: `@ant-design/charts` - 图表组件库

---

## Task 1: Agent 添加带宽采集功能

**Files:**
- Modify: `agent/internal/collector/collector.go`
- Modify: `agent/internal/model/model.go`

- [ ] **Step 1: 在 collector.go 中添加 getBandwidth 函数**

在 `collector.go` 文件的 `getDiskUsage` 函数后添加：

```go
func (c *Collector) getBandwidth() (int64, int64, error) {
	if runtime.GOOS != "linux" {
		return 0, 0, nil
	}

	data, err := os.ReadFile("/proc/net/dev")
	if err != nil {
		return 0, 0, err
	}

	lines := strings.Split(string(data), "\n")
	var totalRx, totalTx int64

	for _, line := range lines[2:] { // 跳过前两行表头
		fields := strings.Fields(line)
		if len(fields) < 10 {
			continue
		}
		// 格式: eth0: rx_bytes rx_packets ... tx_bytes tx_packets
		interfaceName := strings.TrimSuffix(fields[0], ":")
		if interfaceName == "lo" || interfaceName == "lo@" {
			continue
		}

		rx, _ := strconv.ParseInt(fields[1], 10, 64)
		tx, _ := strconv.ParseInt(fields[9], 10, 64)

		totalRx += rx
		totalTx += tx
	}

	return totalRx, totalTx, nil
}
```

- [ ] **Step 2: 在 Collect() 函数中调用 getBandwidth**

在 `collector.go` 的 `Collect()` 函数中添加：

```go
func (c *Collector) Collect() (*model.NodeStatus, error) {
	cpu, err := c.getCPUUsage()
	if err != nil {
		cpu = 0
	}

	mem, err := c.getMemoryUsage()
	if err != nil {
		mem = 0
	}

	disk, err := c.getDiskUsage()
	if err != nil {
		disk = 0
	}

	// 新增：获取带宽
	bandwidthIn, bandwidthOut, _ := c.getBandwidth()

	v2rayStatus := c.checkV2ray()

	return &model.NodeStatus{
		CPUPercent:    cpu,
		MemoryPercent: mem,
		DiskPercent:   disk,
		BandwidthIn:   bandwidthIn,
		BandwidthOut:  bandwidthOut,
		V2rayStatus:   v2rayStatus,
	}, nil
}
```

- [ ] **Step 3: 验证编译**

Run: `cd /home/jat-id/Project/V2rayDash/agent && /usr/local/go/bin/go build -o /tmp/v2ray-agent-test ./cmd/agent`
Expected: 编译成功，无错误

- [ ] **Step 4: 提交代码**

```bash
git add agent/internal/collector/collector.go
git commit -m "feat(agent): add bandwidth collection from /proc/net/dev"
```

---

## Task 2: Backend API 扩展 - 时间范围过滤和带宽差值计算

**Files:**
- Modify: `backend/internal/handler/log.go`
- Modify: `backend/internal/repository/log.go`
- Modify: `backend/internal/model/node_status.go`

- [ ] **Step 1: 创建新的响应结构体**

在 `backend/internal/model/node_status.go` 中添加：

```go
type MetricPoint struct {
	Time  time.Time `json:"time"`
	Value float64   `json:"value"`
}

type BandwidthPoint struct {
	Time  time.Time `json:"time"`
	Value int64     `json:"value"`
}

type NodeStatusMetrics struct {
	CPU         []MetricPoint    `json:"cpu"`
	Memory      []MetricPoint    `json:"memory"`
	Disk        []MetricPoint    `json:"disk"`
	BandwidthIn []BandwidthPoint `json:"bandwidth_in"`
	BandwidthOut []BandwidthPoint `json:"bandwidth_out"`
}

type NodeStatusResponse struct {
	ServerID string             `json:"server_id"`
	Metrics  NodeStatusMetrics  `json:"metrics"`
	Current  *NodeStatusCurrent `json:"current"`
}

type NodeStatusCurrent struct {
	CPUPercent     float64 `json:"cpu_percent"`
	MemoryPercent  float64 `json:"memory_percent"`
	DiskPercent    float64 `json:"disk_percent"`
	BandwidthIn    int64   `json:"bandwidth_in"`
	BandwidthOut   int64   `json:"bandwidth_out"`
	V2rayStatus    string  `json:"v2ray_status"`
	ReportedAt     time.Time `json:"reported_at"`
}
```

- [ ] **Step 2: 修改 repository 层添加时间范围查询**

在 `backend/internal/repository/log.go` 中添加方法：

```go
func (r *LogRepository) GetNodeStatusesByTimeRange(serverID string, timeRange string) ([]*model.NodeStatus, error) {
	var interval string
	switch timeRange {
	case "1h":
		interval = "1 hour"
	case "4h":
		interval = "4 hours"
	case "12h":
		interval = "12 hours"
	case "24h":
		interval = "24 hours"
	default:
		interval = "1 hour"
	}

	query := fmt.Sprintf(`
		SELECT id, server_id, cpu_percent, memory_percent, disk_percent,
		       bandwidth_in, bandwidth_out, v2ray_status, reported_at
		FROM node_status
		WHERE reported_at > NOW() - INTERVAL '%s'
		AND ($1 = '' OR server_id = $1)
		ORDER BY reported_at ASC
	`, interval)

	rows, err := r.db.Query(query, serverID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var statuses []*model.NodeStatus
	for rows.Next() {
		var s model.NodeStatus
		err := rows.Scan(&s.ID, &s.ServerID, &s.CPUPercent, &s.MemoryPercent,
			&s.DiskPercent, &s.BandwidthIn, &s.BandwidthOut, &s.V2rayStatus, &s.ReportedAt)
		if err != nil {
			return nil, err
		}
		statuses = append(statuses, &s)
	}

	return statuses, nil
}
```

- [ ] **Step 3: 修改 handler 层**

在 `backend/internal/handler/log.go` 中修改 `ListNodeStatuses` 方法：

```go
func (h *LogHandler) ListNodeStatuses(c *gin.Context) {
	timeRange := c.DefaultQuery("time_range", "1h")
	serverID := c.Query("server_id")

	statuses, err := h.logRepo.GetNodeStatusesByTimeRange(serverID, timeRange)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if len(statuses) == 0 {
		c.JSON(http.StatusOK, []interface{}{})
		return
	}

	// 按 server_id 分组
	grouped := make(map[string][]*model.NodeStatus)
	for _, s := range statuses {
		grouped[s.ServerID] = append(grouped[s.ServerID], s)
	}

	var responses []model.NodeStatusResponse
	for sid, ss := range grouped {
		resp := buildNodeStatusResponse(sid, ss)
		responses = append(responses, resp)
	}

	c.JSON(http.StatusOK, responses)
}

func buildNodeStatusResponse(serverID string, statuses []*model.NodeStatus) model.NodeStatusResponse {
	// 计算带宽差值
	var baseIn, baseOut int64
	if len(statuses) > 0 {
		baseIn = statuses[0].BandwidthIn
		baseOut = statuses[0].BandwidthOut
	}

	cpu := make([]model.MetricPoint, 0, len(statuses))
	memory := make([]model.MetricPoint, 0, len(statuses))
	disk := make([]model.MetricPoint, 0, len(statuses))
	bandwidthIn := make([]model.BandwidthPoint, 0, len(statuses))
	bandwidthOut := make([]model.BandwidthPoint, 0, len(statuses))

	for _, s := range statuses {
		cpu = append(cpu, model.MetricPoint{Time: s.ReportedAt, Value: s.CPUPercent})
		memory = append(memory, model.MetricPoint{Time: s.ReportedAt, Value: s.MemoryPercent})
		disk = append(disk, model.MetricPoint{Time: s.ReportedAt, Value: s.DiskPercent})
		bandwidthIn = append(bandwidthIn, model.BandwidthPoint{Time: s.ReportedAt, Value: s.BandwidthIn - baseIn})
		bandwidthOut = append(bandwidthOut, model.BandwidthPoint{Time: s.ReportedAt, Value: s.BandwidthOut - baseOut})
	}

	latest := statuses[len(statuses)-1]

	return model.NodeStatusResponse{
		ServerID: serverID,
		Metrics: model.NodeStatusMetrics{
			CPU:          cpu,
			Memory:       memory,
			Disk:         disk,
			BandwidthIn:  bandwidthIn,
			BandwidthOut: bandwidthOut,
		},
		Current: &model.NodeStatusCurrent{
			CPUPercent:    latest.CPUPercent,
			MemoryPercent: latest.MemoryPercent,
			DiskPercent:   latest.DiskPercent,
			BandwidthIn:   latest.BandwidthIn,
			BandwidthOut:  latest.BandwidthOut,
			V2rayStatus:   latest.V2rayStatus,
			ReportedAt:    latest.ReportedAt,
		},
	}
}
```

- [ ] **Step 4: 验证编译**

Run: `cd /home/jat-id/Project/V2rayDash/backend && /usr/local/go/bin/go build -o /tmp/backend-test ./cmd/server`
Expected: 编译成功，无错误

- [ ] **Step 5: 提交代码**

```bash
git add backend/internal/handler/log.go backend/internal/repository/log.go backend/internal/model/node_status.go
git commit -m "feat(backend): add time range filter and bandwidth delta calculation for node status API"
```

---

## Task 3: Frontend 图表展示重构

**Files:**
- Modify: `frontend/src/services/api.ts`
- Modify: `frontend/src/pages/monitor/index.tsx`
- Install: `@ant-design/charts`

- [ ] **Step 1: 安装图表库**

Run: `cd /home/jat-id/Project/V2rayDash/frontend && npm install @ant-design/charts @ant-design/plots`
Expected: 安装成功

- [ ] **Step 2: 更新 API 类型定义**

在 `frontend/src/services/api.ts` 中添加：

```typescript
export interface MetricPoint {
  time: string
  value: number
}

export interface BandwidthPoint {
  time: string
  value: number
}

export interface NodeStatusMetrics {
  cpu: MetricPoint[]
  memory: MetricPoint[]
  disk: MetricPoint[]
  bandwidth_in: BandwidthPoint[]
  bandwidth_out: BandwidthPoint[]
}

export interface NodeStatusCurrent {
  cpu_percent: number
  memory_percent: number
  disk_percent: number
  bandwidth_in: number
  bandwidth_out: number
  v2ray_status: string
  reported_at: string
}

export interface NodeStatusResponse {
  server_id: string
  metrics: NodeStatusMetrics
  current: NodeStatusCurrent
}

// 更新 logAPI
const logAPI = {
  // ...
  getNodeStatuses: (timeRange: string = '1h', serverId?: string): Promise<NodeStatusResponse[]> => {
    const params = new URLSearchParams({ time_range: timeRange })
    if (serverId) params.append('server_id', serverId)
    return api.get(`/logs/node-status?${params.toString()}`)
  },
}
```

- [ ] **Step 3: 重构监控页面**

重写 `frontend/src/pages/monitor/index.tsx`：

```tsx
import { useState, useEffect } from 'react'
import { Card, Row, Col, Statistic, Tag, message, Segmented } from 'antd'
import { Line } from '@ant-design/plots'
import { serverAPI, logAPI, Server, NodeStatusResponse } from '../../services/api'

const TIME_RANGES = [
  { label: '1小时', value: '1h' },
  { label: '4小时', value: '4h' },
  { label: '12小时', value: '12h' },
  { label: '24小时', value: '24h' },
]

export default function Monitor() {
  const [servers, setServers] = useState<Server[]>([])
  const [statuses, setStatuses] = useState<Map<string, NodeStatusResponse>>(new Map())
  const [loading, setLoading] = useState(false)
  const [timeRange, setTimeRange] = useState('1h')

  const loadData = async () => {
    setLoading(true)
    try {
      const [serverData, statusData] = await Promise.all([
        serverAPI.list(),
        logAPI.getNodeStatuses(timeRange)
      ])

      setServers(serverData || [])

      const statusMap = new Map<string, NodeStatusResponse>()
      if (statusData && statusData.length > 0) {
        statusData.forEach(s => statusMap.set(s.server_id, s))
      }
      setStatuses(statusMap)
    } catch (e) {
      message.error('加载失败')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    loadData()
  }, [timeRange])

  const getStatusColor = (status: string) => {
    if (status === 'running') return 'green'
    if (status === 'stopped') return 'red'
    return 'default'
  }

  const getV2rayTag = (status: string) => (
    <Tag color={getStatusColor(status)}>
      {status === 'running' ? '运行中' : status === 'stopped' ? '已停止' : '未知'}
    </Tag>
  )

  const formatBytes = (bytes: number) => {
    if (bytes === 0) return '0 B'
    const k = 1024
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
  }

  const renderLineChart = (data: any[], color: string) => {
    if (!data || data.length === 0) {
      return <div style={{ height: 200, textAlign: 'center', color: '#999' }}>暂无数据</div>
    }
    const config = {
      data,
      xField: 'time',
      yField: 'value',
      smooth: true,
      color,
      lineStyle: { lineWidth: 2 },
      xAxis: { type: 'time' },
      yAxis: { min: 0 },
    }
    return <Line {...config} style={{ height: 200 }} />
  }

  return (
    <div className="animate-in">
      <div className="page-header" style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <div>
          <h1>监控中心</h1>
          <p>查看服务器状态和性能指标</p>
        </div>
        <Segmented
          options={TIME_RANGES}
          value={timeRange}
          onChange={setTimeRange}
        />
      </div>

      {/* Server Cards */}
      <Row gutter={16} style={{ marginBottom: 24 }}>
        {servers.map(server => {
          const status = statuses.get(server.id)
          return (
            <Col span={24} key={server.id} style={{ marginBottom: 16 }}>
              <Card className="morandi-card" loading={loading}>
                <Row gutter={16}>
                  <Col span={4}>
                    <Statistic title={server.name} value={server.ip} />
                    <Tag color={status ? getStatusColor(status.current?.v2ray_status) : 'default'}>
                      {status ? (status.current?.v2ray_status === 'running' ? '运行中' : '已停止') : '离线'}
                    </Tag>
                  </Col>
                  <Col span={20}>
                    {status ? (
                      <>
                        {/* CPU Chart */}
                        <div style={{ marginBottom: 16 }}>
                          <h4>CPU 使用率 (%)</h4>
                          {renderLineChart(
                            status.metrics.cpu.map(p => ({ time: p.time, value: p.value })),
                            '#ee6666'
                          )}
                        </div>
                        {/* Memory Chart */}
                        <div style={{ marginBottom: 16 }}>
                          <h4>内存使用率 (%)</h4>
                          {renderLineChart(
                            status.metrics.memory.map(p => ({ time: p.time, value: p.value })),
                            '#3cb371'
                          )}
                        </div>
                        {/* Bandwidth Chart */}
                        <div style={{ marginBottom: 16 }}>
                          <h4>带宽流量 (入站/出站)</h4>
                          {renderLineChart([
                            ...status.metrics.bandwidth_in.map(p => ({ time: p.time, value: p.value, type: '入站' })),
                            ...status.metrics.bandwidth_out.map(p => ({ time: p.time, value: p.value, type: '出站' })),
                          ], '#7265e6')}
                        </div>
                        {/* Disk */}
                        <div>
                          <h4>磁盘使用率</h4>
                          <Statistic
                            value={status.current?.disk_percent || 0}
                            suffix="%"
                            valueStyle={{ color: '#1890ff' }}
                          />
                        </div>
                        <div style={{ marginTop: 8, fontSize: 12, color: '#999' }}>
                          上报时间: {new Date(status.current?.reported_at).toLocaleString()}
                        </div>
                      </>
                    ) : (
                      <div style={{ textAlign: 'center', padding: 40, color: '#999' }}>
                        等待 Agent 上报状态...
                      </div>
                    )}
                  </Col>
                </Row>
              </Card>
            </Col>
          )
        })}
      </Row>

      {servers.length === 0 && (
        <Card className="morandi-card">
          <div style={{ textAlign: 'center', padding: 40, color: '#999' }}>
            <h3>暂无服务器</h3>
            <p>请先在服务器管理中添加服务器</p>
          </div>
        </Card>
      )}
    </div>
  )
}
```

- [ ] **Step 4: 测试前端编译**

Run: `cd /home/jat-id/Project/V2rayDash/frontend && npm run build 2>&1 | head -50`
Expected: 编译成功，无错误

- [ ] **Step 5: 提交代码**

```bash
git add frontend/src/pages/monitor/index.tsx frontend/src/services/api.ts
git commit -m "feat(frontend): add chart display for monitor with time range selection"
```

---

## Task 4: 构建并部署

- [ ] **Step 1: 构建新 Agent 二进制**

Run: `cd /home/jat-id/Project/V2rayDash/agent && CGO_ENABLED=0 /usr/local/go/bin/go build -ldflags="-s -w" -o /tmp/v2ray-agent-new ./cmd/agent`
Expected: 编译成功

- [ ] **Step 2: 上传到 VOLL-HK**

Upload via dd method (参考之前的方法)

- [ ] **Step 3: 重启 Agent 服务**

Run: `sshpass -p '7e0bcRWr6g7P15QTjJ' ssh -o StrictHostKeyChecking=no root@103.20.223.104 "systemctl restart v2ray-agent && sleep 2 && systemctl status v2ray-agent | head -15"`

- [ ] **Step 4: 构建并部署后端**

Run: `cd /home/jat-id/Project/V2rayDash/backend && /usr/local/go/bin/go build -ldflags="-s -w" -o /tmp/backend-new ./cmd/server`

Upload to Aliyun and restart service

- [ ] **Step 5: 前端部署**

Run: `cd /home/jat-id/Project/V2rayDash/frontend && npm run build`

Deploy dist folder to web server

---

## 自检清单

- [ ] Spec 覆盖检查：
  - [x] Agent 带宽采集 - Task 1
  - [x] 后端时间范围过滤 - Task 2
  - [x] 后端带宽差值计算 - Task 2
  - [x] 前端 Tab 切换 - Task 3
  - [x] CPU 曲线图 - Task 3
  - [x] 内存曲线图 - Task 3
  - [x] 入站/出站合并曲线图 - Task 3
  - [x] 磁盘只显示当前值 - Task 3

- [ ] 类型一致性检查：
  - [x] Agent `BandwidthIn/BandwidthOut` 字段类型与后端 `HeartbeatRequest` 一致
  - [x] 后端 `NodeStatusResponse` 字段与前端 `NodeStatusResponse` 类型一致

- [ ] 无占位符检查：所有代码步骤都包含完整实现
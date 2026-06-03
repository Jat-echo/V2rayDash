import { useState, useEffect, useRef } from 'react'
import { serverAPI, logAPI, Server, NodeStatusResponse, MetricPoint, BandwidthPoint } from '../../services/api'

const TIME_RANGES = [
  { label: '1H', value: '1h' },
  { label: '4H', value: '4h' },
  { label: '12H', value: '12h' },
  { label: '24H', value: '24h' },
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
        statusData.forEach((s: NodeStatusResponse) => statusMap.set(s.server_id, s))
      }
      setStatuses(statusMap)
    } catch (e) {
      console.error('[Monitor] loadData error:', e)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    loadData()
  }, [timeRange])

  const formatBytes = (bytes: number) => {
    if (bytes === 0) return '0 B'
    const k = 1024
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i]
  }

  return (
    <div className="monitor-container">
      <div className="bg-grid" />

      <header className="monitor-header animate-in">
        <div className="header-left">
          <h1>监控中心</h1>
          <p>// REAL-TIME SERVER TELEMETRY</p>
        </div>
        <div className="time-tabs">
          {TIME_RANGES.map(range => (
            <button
              key={range.value}
              className={`time-tab ${timeRange === range.value ? 'active' : ''}`}
              onClick={() => setTimeRange(range.value)}
            >
              {range.label}
            </button>
          ))}
        </div>
      </header>

      {servers.length === 0 ? (
        <div className="empty-state animate-in">
          <h3>暂无服务器</h3>
          <p>请先在服务器管理中添加服务器</p>
        </div>
      ) : (
        servers.map((server, index) => {
          const status = statuses.get(server.id)
          const isOnline = status?.current?.v2ray_status === 'running'
          const cpu = status?.current?.cpu_percent || 0
          const memory = status?.current?.memory_percent || 0
          const disk = status?.current?.disk_percent || 0
          const bandwidthIn = status?.current?.bandwidth_in || 0
          const bandwidthOut = status?.current?.bandwidth_out || 0

          return (
            <div key={server.id} className={`server-grid animate-in animate-delay-${index + 1}`}>
              {/* Left Panel - Server Info */}
              <div className="server-card">
                <div className="server-status-indicator">
                  <div className={`status-dot ${isOnline ? 'online' : 'offline'}`} />
                  <span className={`status-label ${isOnline ? 'online' : 'offline'}`}>
                    {isOnline ? 'ONLINE' : 'OFFLINE'}
                  </span>
                </div>
                <div className="server-name">{server.name}</div>
                <div className="server-ip">{server.ip}</div>

                <div className="stats-grid">
                  <div className="stat-item">
                    <div className="stat-label">CPU</div>
                    <div className="stat-value cpu">
                      {cpu.toFixed(1)}<span className="stat-unit">%</span>
                    </div>
                  </div>
                  <div className="stat-item">
                    <div className="stat-label">内存</div>
                    <div className="stat-value memory">
                      {memory.toFixed(1)}<span className="stat-unit">%</span>
                    </div>
                  </div>
                  <div className="stat-item">
                    <div className="stat-label">磁盘</div>
                    <div className="stat-value disk">
                      {disk.toFixed(0)}<span className="stat-unit">%</span>
                    </div>
                  </div>
                  <div className="stat-item">
                    <div className="stat-label">带宽</div>
                    <div className="stat-value bandwidth">
                      {formatBytes(bandwidthIn + bandwidthOut)}<span className="stat-unit">/s</span>
                    </div>
                  </div>
                </div>
              </div>

              {/* Right Panel - Metrics */}
              <div className="metrics-panel">
                {status ? (
                  <>
                    <MetricCard
                      title="CPU 使用率"
                      indicator="cpu"
                      value={`${cpu.toFixed(1)}%`}
                      highlight
                      data={status.metrics.cpu.map(p => ({ time: p.time, value: p.value }))}
                      color="var(--cyan)"
                    />
                    <MetricCard
                      title="内存使用率"
                      indicator="memory"
                      value={`${memory.toFixed(1)}%`}
                      color="var(--magenta)"
                      data={status.metrics.memory.map(p => ({ time: p.time, value: p.value }))}
                      color="var(--magenta)"
                    />
                    <MetricCard
                      title="带宽流量"
                      indicator="bandwidth"
                    >
                      <div className="bandwidth-chart">
                        <div className="bandwidth-item">
                          <div className="bandwidth-label">
                            <span className="bandwidth-arrow in">↓</span>
                            <span className="bandwidth-title">入站</span>
                          </div>
                          <ChartRenderer
                            data={status.metrics.bandwidth_in.map(p => ({ time: p.time, value: p.value }))}
                            color="var(--cyan)"
                          />
                        </div>
                        <div className="bandwidth-item">
                          <div className="bandwidth-label">
                            <span className="bandwidth-arrow out">↑</span>
                            <span className="bandwidth-title">出站</span>
                          </div>
                          <ChartRenderer
                            data={status.metrics.bandwidth_out.map(p => ({ time: p.time, value: p.value }))}
                            color="var(--magenta)"
                          />
                        </div>
                      </div>
                    </MetricCard>
                  </>
                ) : (
                  <div className="metric-card empty-card">
                    <div className="empty-content">
                      <div className="empty-icon">⏳</div>
                      <div>等待 Agent 上报状态...</div>
                    </div>
                  </div>
                )}
              </div>
            </div>
          )
        })
      )}
    </div>
  )
}

function MetricCard({ title, indicator, value, highlight, color, children, data }: {
  title: string
  indicator: string
  value?: string
  highlight?: boolean
  color?: string
  children?: React.ReactNode
  data?: { time: string; value: number }[]
}) {
  return (
    <div className="metric-card">
      <div className="metric-header">
        <div className="metric-title">
          <div className={`metric-indicator ${indicator}`} />
          {title}
        </div>
        {value && (
          <div className={`metric-value-current ${highlight ? 'highlight' : ''}`} style={color ? { color } : {}}>
            {value}
          </div>
        )}
      </div>
      {children || (data && <ChartRenderer data={data} color={color || 'var(--cyan)'} />)}
    </div>
  )
}

function ChartRenderer({ data, color }: { data: { time: string; value: number }[]; color: string }) {
  const containerRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!containerRef.current || !data || data.length === 0) return

    const chart = containerRef.current
    const trace = {
      x: data.map(p => p.time),
      y: data.map(p => p.value),
      type: 'scatter' as const,
      mode: 'lines' as const,
      line: { color, width: 2, shape: 'spline' as const },
      fill: 'tozeroy' as const,
      fillcolor: color.replace(')', ', 0.1)').replace('var(', 'rgba('),
    }

    const layout = {
      margin: { t: 10, r: 10, b: 30, l: 40 },
      paper_bgcolor: 'transparent',
      plot_bgcolor: 'transparent',
      font: { color: '#6a6a7a', family: 'JetBrains Mono, monospace' },
      xaxis: {
        showgrid: true,
        gridcolor: 'rgba(0, 255, 242, 0.05)',
        linecolor: 'rgba(0, 255, 242, 0.1)',
        tickcolor: 'rgba(0, 255, 242, 0.1)',
        ticks: 'outside',
        tickfont: { size: 10 },
      },
      yaxis: {
        showgrid: true,
        gridcolor: 'rgba(0, 255, 242, 0.05)',
        linecolor: 'rgba(0, 255, 242, 0.1)',
        tickcolor: 'rgba(0, 255, 242, 0.1)',
        ticks: 'outside',
        tickfont: { size: 10 },
        rangemode: 'tozero' as const,
      },
    }

    import('plotly.js-dist-min').then((Plotly) => {
      Plotly.newPlot(chart, [trace], layout, {
        responsive: true,
        displayModeBar: false,
        legend: { display: false },
      })
    })

    return () => {
      if (containerRef.current) {
        Plotly.purge(chart)
      }
    }
  }, [data, color])

  return <div ref={containerRef} className="chart-container" />
}
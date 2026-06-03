import { useState, useEffect, useRef } from 'react'
import { Segmented } from 'antd'
import { serverAPI, logAPI, Server, NodeStatusResponse, MetricPoint, BandwidthPoint } from '../../services/api'
import { formatBytes } from '../../utils/format'
import { FlagName } from '../../components/FlagName'

const TIME_RANGES = [
  { label: '1小时', value: '1h' },
  { label: '4小时', value: '4h' },
  { label: '12小时', value: '12h' },
  { label: '24小时', value: '24h' },
]

export default function Monitor() {
  const [servers, setServers] = useState<Server[]>([])
  const [statuses, setStatuses] = useState<Map<string, NodeStatusResponse>>(new Map())
  const [timeRange, setTimeRange] = useState('1h')

  const loadData = async () => {
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
    }
  }

  useEffect(() => {
    loadData()
  }, [timeRange])

  return (
    <div className="monitor-page">
      <style>{`
        .monitor-page {
          --bg-primary: #F5F5F5;
          --bg-card: #FFFFFF;
          --text-primary: #262626;
          --text-secondary: #595959;
          --text-muted: #8C8C8C;
          --border-color: #E8E8E8;
          --shadow: 0 1px 2px rgba(0,0,0,0.06);
        }
        .monitor-page .page-wrap {
          padding: 20px 24px;
          max-width: 1400px;
          margin: 0 auto;
        }
        .monitor-page .page-header {
          display: flex;
          justify-content: space-between;
          align-items: center;
          margin-bottom: 20px;
        }
        .monitor-page .page-title h1 {
          font-size: 20px;
          font-weight: 600;
          color: var(--text-primary);
          margin: 0;
        }
        .monitor-page .time-tabs .ant-segmented {
          background: #F5F5F5;
          padding: 3px;
          border-radius: 6px;
        }
        .monitor-page .time-tabs .ant-segmented-item {
          color: var(--text-secondary);
          font-size: 13px;
          padding: 4px 12px;
          height: 28px;
        }
        .monitor-page .time-tabs .ant-segmented-item-selected {
          background: var(--bg-card);
          color: var(--text-primary);
          box-shadow: var(--shadow);
        }
        .monitor-page .server-card {
          background: var(--bg-card);
          border: 1px solid var(--border-color);
          border-radius: 8px;
          box-shadow: var(--shadow);
          margin-bottom: 16px;
          overflow: hidden;
        }
        .monitor-page .server-header {
          padding: 14px 20px;
          border-bottom: 1px solid var(--border-color);
          display: flex;
          justify-content: space-between;
          align-items: center;
          background: #FAFAFA;
        }
        .monitor-page .server-info {
          display: flex;
          align-items: center;
          gap: 10px;
        }
        .monitor-page .status-dot {
          width: 8px;
          height: 8px;
          border-radius: 50%;
          flex-shrink: 0;
        }
        .monitor-page .status-dot.online { background: #52C41A; }
        .monitor-page .status-dot.offline { background: #D9D9D9; }
        .monitor-page .server-name {
          font-size: 14px;
          font-weight: 600;
          color: var(--text-primary);
        }
        .monitor-page .server-ip {
          font-size: 12px;
          color: var(--text-muted);
        }
        .monitor-page .server-badge {
          font-size: 12px;
          padding: 2px 8px;
          border-radius: 4px;
          background: #F6FFED;
          color: #52C41A;
          border: 1px solid #B7EB8F;
        }
        .monitor-page .server-badge.offline {
          background: #F5F5F5;
          color: #8C8C8C;
          border-color: #D9D9D9;
        }
        .monitor-page .metrics-row {
          display: grid;
          grid-template-columns: repeat(4, 1fr);
          border-top: none;
        }
        @media (max-width: 1100px) {
          .monitor-page .metrics-row { grid-template-columns: repeat(2, 1fr); }
        }
        @media (max-width: 700px) {
          .monitor-page .metrics-row { grid-template-columns: 1fr; }
          .monitor-page .page-header { flex-direction: column; gap: 12px; align-items: flex-start; }
        }
        .monitor-page .metric-cell {
          padding: 14px 16px;
          border-right: 1px solid var(--border-color);
          min-width: 0;
        }
        .monitor-page .metric-cell:last-child { border-right: none; }
        .monitor-page .metric-label {
          font-size: 12px;
          color: var(--text-secondary);
          margin-bottom: 6px;
        }
        .monitor-page .metric-value {
          font-size: 20px;
          font-weight: 500;
          color: var(--text-primary);
          font-family: 'JetBrains Mono', monospace;
          margin-bottom: 8px;
        }
        .monitor-page .metric-value .unit {
          font-size: 12px;
          color: var(--text-muted);
          font-weight: 400;
          margin-left: 2px;
        }
        .monitor-page .chart-area {
          height: 140px;
          margin: 0 -2px;
        }
        .monitor-page .empty-chart {
          height: 120px;
          display: flex;
          align-items: center;
          justify-content: center;
          color: var(--text-muted);
          font-size: 12px;
          background: #FAFAFA;
          border-radius: 4px;
        }
        @keyframes fadeInUp {
          from { opacity: 0; transform: translateY(8px); }
          to { opacity: 1; transform: translateY(0); }
        }
        .monitor-page .animate-in {
          animation: fadeInUp 0.4s ease-out;
        }
      `}</style>

      <div className="page-wrap">
        <div className="page-header animate-in">
          <div className="page-title">
            <h1>云监控</h1>
          </div>
          <div className="time-tabs">
            <Segmented
              options={TIME_RANGES}
              value={timeRange}
              onChange={(val) => setTimeRange(val as string)}
            />
          </div>
        </div>

        {servers.length === 0 ? (
          <div style={{ textAlign: 'center', padding: 60, color: '#8C8C8C', background: '#fff', border: '1px dashed #E8E8E8', borderRadius: 8 }}>
            暂无服务器
          </div>
        ) : (
          servers.map((server) => {
            const status = statuses.get(server.id)
            const isOnline = status?.current?.v2ray_status === 'running'
            const cpu = status?.current?.cpu_percent || 0
            const memory = status?.current?.memory_percent || 0

            return (
              <div key={server.id} className="server-card animate-in">
                <div className="server-header">
                  <div className="server-info">
                    <div className={`status-dot ${isOnline ? 'online' : 'offline'}`} />
                    <span className="server-name"><FlagName name={server.name} /></span>
                    <span className="server-ip">{server.ip}</span>
                  </div>
                  <div className={`server-badge ${isOnline ? '' : 'offline'}`}>
                    {isOnline ? '运行中' : '离线'}
                  </div>
                </div>

                {status ? (
                  <div className="metrics-row">
                    <div className="metric-cell">
                      <div className="metric-label">CPU使用率</div>
                      <div className="metric-value">
                        {cpu.toFixed(1)}<span className="unit">%</span>
                      </div>
                      <div className="chart-area">
                        <PlotlyLine
                          data={status.metrics.cpu.map((p: MetricPoint) => ({ time: p.time, value: p.value }))}
                          color="#1677FF"
                          type="percent"
                        />
                      </div>
                    </div>

                    <div className="metric-cell">
                      <div className="metric-label">内存使用率</div>
                      <div className="metric-value">
                        {memory.toFixed(1)}<span className="unit">%</span>
                      </div>
                      <div className="chart-area">
                        <PlotlyLine
                          data={status.metrics.memory.map((p: MetricPoint) => ({ time: p.time, value: p.value }))}
                          color="#722ED1"
                          type="percent"
                        />
                      </div>
                    </div>

                    <div className="metric-cell">
                      <div className="metric-label">入网带宽</div>
                      <div className="metric-value">
                        {formatBytes(status.current?.bandwidth_in || 0)}
                      </div>
                      <div className="chart-area">
                        <PlotlyLine
                          data={status.metrics.bandwidth_in.map((p: BandwidthPoint) => ({ time: p.time, value: p.value }))}
                          color="#1677FF"
                          type="bandwidth"
                        />
                      </div>
                    </div>

                    <div className="metric-cell">
                      <div className="metric-label">出网带宽</div>
                      <div className="metric-value">
                        {formatBytes(status.current?.bandwidth_out || 0)}
                      </div>
                      <div className="chart-area">
                        <PlotlyLine
                          data={status.metrics.bandwidth_out.map((p: BandwidthPoint) => ({ time: p.time, value: p.value }))}
                          color="#FA541C"
                          type="bandwidth"
                        />
                      </div>
                    </div>
                  </div>
                ) : (
                  <div className="empty-chart">等待数据...</div>
                )}
              </div>
            )
          })
        )}
      </div>
    </div>
  )
}

function PlotlyLine({ data, color, type = 'value' }: { data: { time: string; value: number }[]; color: string; type?: 'percent' | 'bandwidth' }) {
  const containerRef = useRef<HTMLDivElement>(null)

  const formatHoverValue = (y: number) => {
    if (type === 'percent') {
      return y.toFixed(2) + '%'
    }
    if (type === 'bandwidth') {
      // delta bytes / 30s = bytes per second
      const bytesPerSec = y / 30
      if (bytesPerSec === 0) return '0 B/s'
      const k = 1024
      const sizes = ['B/s', 'KB/s', 'MB/s', 'GB/s']
      const i = Math.floor(Math.log(Math.abs(bytesPerSec)) / Math.log(k))
      const idx = Math.min(i, sizes.length - 1)
      return (bytesPerSec / Math.pow(k, idx)).toFixed(2) + ' ' + sizes[idx]
    }
    return y.toFixed(2)
  }

  useEffect(() => {
    if (!containerRef.current || !data || data.length === 0) return

    const chart = containerRef.current

    const trace = {
      x: data.map(p => p.time),
      y: data.map(p => p.value),
      type: 'scatter' as const,
      mode: 'lines' as const,
      line: { color, width: 1.5, shape: 'spline' as const },
      fill: 'tozeroy' as const,
      fillcolor: color + '20',
      hovertemplate: '%{customdata}<extra></extra>',
      customdata: data.map(p => formatHoverValue(p.value)),
    }

    const layout = {
      margin: { t: 5, r: 5, b: 28, l: 38 },
      paper_bgcolor: 'transparent',
      plot_bgcolor: 'transparent',
      font: { color: '#8C8C8C', family: 'Noto Sans SC, sans-serif', size: 9 },
      xaxis: {
        showgrid: false,
        gridcolor: '#F0F0F0',
        linecolor: '#D9D9D9',
        tickcolor: '#D9D9D9',
        ticks: 'outside',
        tickfont: { size: 9 },
        showline: true,
        zeroline: false,
        hoverformat: '%H:%M',
      },
      yaxis: {
        showgrid: true,
        gridcolor: '#F5F5F5',
        linecolor: 'transparent',
        tickcolor: 'transparent',
        ticks: 'outside',
        tickfont: { size: 9 },
        showline: false,
        zeroline: false,
        rangemode: 'tozero' as const,
      },
      showlegend: false,
      hovermode: 'x unified',
      hoverlabel: {
        bgcolor: '#fff',
        bordercolor: color,
        borderwidth: 1,
        font: { size: 11, color: '#595959' },
      },
    }

    const config = {
      responsive: true,
      displayModeBar: false,
    }

    import('plotly.js-dist-min').then((Plotly) => {
      Plotly.newPlot(chart, [trace], layout, config)
    })

    return () => {
      if (containerRef.current) {
        try { Plotly.purge(chart) } catch {}
      }
    }
  }, [data, color, type])

  if (!data || data.length === 0) {
    return <div className="empty-chart">暂无数据</div>
  }

  return <div ref={containerRef} style={{ width: '100%', height: '100%' }} />
}
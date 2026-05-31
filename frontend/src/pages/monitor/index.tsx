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
        statusData.forEach((s: NodeStatusResponse) => statusMap.set(s.server_id, s))
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
      return <div style={{ height: 200, textAlign: 'center', color: '#999', lineHeight: '200px' }}>暂无数据</div>
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
                            status.metrics.cpu.map((p: MetricPoint) => ({ time: p.time, value: p.value })),
                            '#ee6666'
                          )}
                        </div>
                        {/* Memory Chart */}
                        <div style={{ marginBottom: 16 }}>
                          <h4>内存使用率 (%)</h4>
                          {renderLineChart(
                            status.metrics.memory.map((p: MetricPoint) => ({ time: p.time, value: p.value })),
                            '#3cb371'
                          )}
                        </div>
                        {/* Bandwidth Chart */}
                        <div style={{ marginBottom: 16 }}>
                          <h4>带宽流量 (入站/出站)</h4>
                          {renderLineChart([
                            ...status.metrics.bandwidth_in.map((p: BandwidthPoint) => ({ time: p.time, value: p.value, type: '入站' })),
                            ...status.metrics.bandwidth_out.map((p: BandwidthPoint) => ({ time: p.time, value: p.value, type: '出站' })),
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
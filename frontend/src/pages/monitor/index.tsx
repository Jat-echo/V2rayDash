import { useState, useEffect } from 'react'
import { Card, Row, Col, Statistic, Tag, Progress, message } from 'antd'
import { serverAPI, logAPI, Server, NodeStatus } from '../../services/api'

export default function Monitor() {
  const [servers, setServers] = useState<Server[]>([])
  const [statuses, setStatuses] = useState<Map<string, NodeStatus>>(new Map())
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    loadData()
    const interval = setInterval(loadData, 30000) // 30秒刷新
    return () => clearInterval(interval)
  }, [])

  const loadData = async () => {
    setLoading(true)
    try {
      const [serverData, statusData] = await Promise.all([
        serverAPI.list(),
        logAPI.getNodeStatuses()
      ])

      setServers(serverData || [])

      const statusMap = new Map<string, NodeStatus>()
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

  return (
    <div className="animate-in">
      {/* Page Header */}
      <div className="page-header">
        <h1>监控中心</h1>
        <p>查看服务器状态和性能指标</p>
      </div>

      {/* Server Cards with Status */}
      <Row gutter={16} style={{ marginBottom: 24 }}>
        {servers.map(server => {
          const status = statuses.get(server.id)
          return (
            <Col span={8} key={server.id} style={{ marginBottom: 16 }}>
              <Card
                className="morandi-card"
                title={
                  <span style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                    <span>{server.name}</span>
                    {status ? getV2rayTag(status.v2ray_status) : <Tag color="default">离线</Tag>}
                  </span>
                }
                loading={loading}
              >
                <Statistic title="IP 地址" value={server.ip} />
                <div style={{ marginTop: 16 }}>
                  <div style={{ color: 'var(--text-secondary)', marginBottom: 4 }}>
                    SSH: {server.ssh_port}
                  </div>
                  {status ? (
                    <>
                      <div style={{ marginTop: 16 }}>
                        <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 4 }}>
                          <span>CPU</span>
                          <span style={{ color: 'var(--morandi-terracotta)' }}>{status.cpu_percent.toFixed(1)}%</span>
                        </div>
                        <Progress percent={status.cpu_percent} showInfo={false} strokeColor="var(--morandi-dusty-rose)" size="small" />
                      </div>
                      <div style={{ marginTop: 12 }}>
                        <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 4 }}>
                          <span>内存</span>
                          <span style={{ color: 'var(--morandi-sage)' }}>{status.memory_percent.toFixed(1)}%</span>
                        </div>
                        <Progress percent={status.memory_percent} showInfo={false} strokeColor="var(--morandi-sage)" size="small" />
                      </div>
                      <div style={{ marginTop: 12 }}>
                        <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 4 }}>
                          <span>磁盘</span>
                          <span style={{ color: 'var(--morandi-sky)' }}>{status.disk_percent.toFixed(1)}%</span>
                        </div>
                        <Progress percent={status.disk_percent} showInfo={false} strokeColor="var(--morandi-sky)" size="small" />
                      </div>
                      <div style={{ marginTop: 12, display: 'flex', gap: 16 }}>
                        <div>
                          <div style={{ fontSize: 12, color: 'var(--text-muted)' }}>入站</div>
                          <div style={{ color: 'var(--morandi-lavender)' }}>{formatBytes(status.bandwidth_in)}</div>
                        </div>
                        <div>
                          <div style={{ fontSize: 12, color: 'var(--text-muted)' }}>出站</div>
                          <div style={{ color: 'var(--morandi-lavender)' }}>{formatBytes(status.bandwidth_out)}</div>
                        </div>
                      </div>
                      <div style={{ marginTop: 8, fontSize: 12, color: 'var(--text-muted)' }}>
                        上报时间: {new Date(status.reported_at).toLocaleString()}
                      </div>
                    </>
                  ) : (
                    <div style={{ marginTop: 16, textAlign: 'center', color: 'var(--text-muted)' }}>
                      等待 Agent 上报状态...
                    </div>
                  )}
                </div>
              </Card>
            </Col>
          )
        })}
      </Row>

      {servers.length === 0 && (
        <Card className="morandi-card">
          <div style={{ textAlign: 'center', padding: '40px 0', color: 'var(--text-muted)' }}>
            <div style={{ fontSize: 48, marginBottom: 16 }}>🖥️</div>
            <h3 style={{ color: 'var(--text-secondary)', marginBottom: 8 }}>暂无服务器</h3>
            <p>请先在服务器管理中添加服务器</p>
          </div>
        </Card>
      )}
    </div>
  )
}
import { useState, useEffect } from 'react'
import { Card, Row, Col, Statistic, Tag, message } from 'antd'
import { serverAPI, Server } from '../../services/api'

interface NodeStatus {
  server_id: string
  cpu_percent: number
  memory_percent: number
  disk_percent: number
  v2ray_status: string
  reported_at: string
}

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
      const data = await serverAPI.list()
      setServers(data || [])
      // 节点状态需要通过 agent 上报，目前暂无数据源
      // 如果后续实现了 agent 上报机制，可以在这里获取状态
      setStatuses(new Map())
    } catch (e) {
      message.error('加载失败')
    } finally {
      setLoading(false)
    }
  }

  const getStatusTag = (status: string) => {
    const color = status === 'online' ? 'green' : status === 'offline' ? 'red' : 'default'
    return <Tag color={color}>{status === 'online' ? '在线' : status === 'offline' ? '离线' : '未知'}</Tag>
  }

  return (
    <div className="animate-in">
      {/* Page Header */}
      <div className="page-header">
        <h1>监控中心</h1>
        <p>查看服务器状态和性能指标</p>
      </div>

      {/* Server Cards */}
      <Row gutter={16} style={{ marginBottom: 24 }}>
        {servers.map(server => (
          <Col span={8} key={server.id} style={{ marginBottom: 16 }}>
            <Card
              className="morandi-card"
              title={
                <span style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                  <span>{server.name}</span>
                  {getStatusTag(server.status)}
                </span>
              }
              loading={loading}
            >
              <Statistic title="IP 地址" value={server.ip} />
              <div style={{ marginTop: 16 }}>
                <div style={{ color: 'var(--text-secondary)', marginBottom: 4 }}>
                  SSH 端口: {server.ssh_port}
                </div>
                <div style={{ color: 'var(--text-muted)', fontSize: 12 }}>
                  用户: {server.ssh_user}
                </div>
              </div>
            </Card>
          </Col>
        ))}
      </Row>

      {/* Note about agent status */}
      <Card className="morandi-card">
        <div style={{ textAlign: 'center', padding: '40px 0', color: 'var(--text-muted)' }}>
          <div style={{ fontSize: 48, marginBottom: 16 }}>📊</div>
          <h3 style={{ color: 'var(--text-secondary)', marginBottom: 8 }}>节点状态监控</h3>
          <p>节点状态需要通过 Agent 上报。</p>
          <p>部署 Agent 后，服务器状态和性能指标将显示在这里。</p>
        </div>
      </Card>
    </div>
  )
}
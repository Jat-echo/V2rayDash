import { useState, useEffect } from 'react'
import { Card, Row, Col, Statistic, Table, Tag } from 'antd'
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

  useEffect(() => {
    loadServers()
    const interval = setInterval(loadServers, 30000) // 30秒刷新
    return () => clearInterval(interval)
  }, [])

  const loadServers = async () => {
    const data = await serverAPI.list()
    setServers(data)
  }

  const getStatusTag = (status: string) => {
    const color = status === 'online' ? 'green' : status === 'offline' ? 'red' : 'default'
    return <Tag color={color}>{status}</Tag>
  }

  return (
    <div>
      <Row gutter={16} style={{ marginBottom: 24 }}>
        {servers.map(server => (
          <Col span={6} key={server.id}>
            <Card title={server.name} extra={getStatusTag(server.status)}>
              <Statistic title="IP" value={server.ip} />
              <div style={{ marginTop: 16 }}>
                <div>SSH: {server.ssh_port}</div>
                <div>最后更新: {server.updated_at}</div>
              </div>
            </Card>
          </Col>
        ))}
      </Row>

      <Card title="节点状态">
        <Table
          dataSource={Array.from(statuses.values())}
          rowKey="server_id"
          columns={[
            { title: '服务器', dataIndex: 'server_id', render: (id: string) => servers.find(s => s.id === id)?.name || id },
            { title: 'CPU', dataIndex: 'cpu_percent', render: (v: number) => `${v?.toFixed(1)}%` },
            { title: '内存', dataIndex: 'memory_percent', render: (v: number) => `${v?.toFixed(1)}%` },
            { title: '硬盘', dataIndex: 'disk_percent', render: (v: number) => `${v?.toFixed(1)}%` },
            { title: 'V2ray', dataIndex: 'v2ray_status' },
            { title: '上报时间', dataIndex: 'reported_at' },
          ]}
        />
      </Card>
    </div>
  )
}
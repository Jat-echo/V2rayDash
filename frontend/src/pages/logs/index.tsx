import { useState, useEffect } from 'react'
import { Table, Card, Tag, message } from 'antd'
import { logAPI, OperationLog } from '../../services/api'

const actionNames: Record<string, string> = {
  'create_server': '创建服务器',
  'delete_server': '删除服务器',
  'create_subscription': '创建订阅',
  'delete_subscription': '删除订阅',
  'create_account': '创建账号',
  'delete_account': '删除账号',
}

export default function Logs() {
  const [logs, setLogs] = useState<OperationLog[]>([])
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    loadLogs()
  }, [])

  const loadLogs = async () => {
    setLoading(true)
    try {
      const data = await logAPI.list()
      setLogs(data || [])
    } catch (e) {
      message.error('加载日志失败')
      setLogs([])
    } finally {
      setLoading(false)
    }
  }

  const columns = [
    { title: '时间', dataIndex: 'created_at', width: 180 },
    { title: '操作人', dataIndex: 'operator', width: 100 },
    {
      title: '动作',
      dataIndex: 'action',
      width: 120,
      render: (v: string) => <Tag color="blue">{actionNames[v] || v}</Tag>
    },
    { title: '目标类型', dataIndex: 'target_type', width: 100 },
    { title: '目标ID', dataIndex: 'target_id', render: (v: string) => v ? v.substring(0, 8) + '...' : '-' },
    { title: 'IP地址', dataIndex: 'ip', width: 120 },
  ]

  return (
    <div className="animate-in">
      {/* Page Header */}
      <div className="page-header">
        <h1>操作日志</h1>
        <p>查看系统操作记录</p>
      </div>

      {/* Stats Grid */}
      <div className="stats-grid">
        <div className="stat-card animate-in animate-delay-1">
          <div className="stat-icon sky">📋</div>
          <div className="stat-content">
            <h3>{logs.length}</h3>
            <p>日志总数</p>
          </div>
        </div>
      </div>

      {/* Logs Table */}
      <Card className="morandi-card">
        <Table
          columns={columns}
          dataSource={logs}
          rowKey="id"
          loading={loading}
          pagination={{ pageSize: 10 }}
          locale={{ emptyText: '暂无日志记录' }}
        />
      </Card>
    </div>
  )
}
import { useState, useEffect } from 'react'
import { Table, Card, Tag, Tooltip, message } from 'antd'
import { logAPI, OperationLog } from '../../services/api'

const actionNames: Record<string, string> = {
  'create_server': '创建服务器',
  'delete_server': '删除服务器',
  'create_subscription': '创建订阅',
  'delete_subscription': '删除订阅',
  'create_account': '创建账号',
  'delete_account': '删除账号',
}

const targetTypeNames: Record<string, string> = {
  'server': '服务器',
  'subscription': '订阅',
  'account': '账号',
}

const actionTagStyle = (action: string): React.CSSProperties => {
  if (action.startsWith('create_')) {
    return {
      background: 'rgba(168,181,160,0.15)',
      color: '#4d6e48',
      border: '1px solid rgba(168,181,160,0.4)',
      borderRadius: 6,
    }
  }
  if (action.startsWith('delete_')) {
    return {
      background: 'rgba(196,131,106,0.12)',
      color: '#8a4a2e',
      border: '1px solid rgba(196,131,106,0.35)',
      borderRadius: 6,
    }
  }
  return {
    background: 'rgba(158,154,147,0.12)',
    color: '#5a5650',
    border: '1px solid rgba(158,154,147,0.3)',
    borderRadius: 6,
  }
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
    } catch (e: any) {
      message.error(`加载日志失败：${e?.message || '请检查网络连接'}`)
      setLogs([])
    } finally {
      setLoading(false)
    }
  }

  const columns = [
    {
      title: '时间',
      dataIndex: 'created_at',
      width: 180,
      render: (v: string) => v ? new Date(v).toLocaleString('zh-CN', {
        year: 'numeric', month: '2-digit', day: '2-digit',
        hour: '2-digit', minute: '2-digit', second: '2-digit',
      }) : '-',
    },
    { title: '操作人', dataIndex: 'operator', width: 100 },
    {
      title: '动作',
      dataIndex: 'action',
      width: 130,
      filters: Object.entries(actionNames).map(([k, v]) => ({ text: v, value: k })),
      onFilter: (value: any, record: OperationLog) => record.action === value,
      render: (v: string) => (
        <Tag style={actionTagStyle(v)}>{actionNames[v] || v}</Tag>
      ),
    },
    {
      title: '目标类型',
      dataIndex: 'target_type',
      width: 90,
      render: (v: string) => targetTypeNames[v] || v || '-',
    },
    {
      title: '目标 ID',
      dataIndex: 'target_id',
      render: (v: string) => v ? (
        <Tooltip title={v} placement="topLeft">
          <span style={{ fontFamily: 'monospace', cursor: 'default', color: 'var(--text-secondary)' }}>
            {v.substring(0, 8)}…
          </span>
        </Tooltip>
      ) : '-',
    },
    { title: 'IP 地址', dataIndex: 'ip', width: 130 },
  ]

  return (
    <div className="animate-in">
      <div className="page-header">
        <h1>操作日志</h1>
        <p>查看系统操作记录，共 {logs.length} 条</p>
      </div>

      <Card className="morandi-card">
        <Table
          columns={columns}
          dataSource={logs}
          rowKey="id"
          loading={loading}
          pagination={{ pageSize: 20, showSizeChanger: true, pageSizeOptions: ['10', '20', '50'] }}
          locale={{ emptyText: '暂无日志记录' }}
        />
      </Card>
    </div>
  )
}

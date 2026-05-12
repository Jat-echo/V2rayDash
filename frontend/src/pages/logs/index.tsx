import { useState, useEffect } from 'react'
import { Table, Select, DatePicker, Space } from 'antd'
import { logAPI, OperationLog } from '../../services/api'

const { RangePicker } = DatePicker

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
      setLogs(data)
    } catch (e) {
      // handle error
    } finally {
      setLoading(false)
    }
  }

  const columns = [
    { title: '时间', dataIndex: 'created_at' },
    { title: '操作人', dataIndex: 'operator' },
    { title: '动作', dataIndex: 'action' },
    { title: '目标类型', dataIndex: 'target_type' },
    { title: '目标ID', dataIndex: 'target_id', render: (v: string) => v?.slice(0, 8) || '-' },
    { title: 'IP', dataIndex: 'ip' },
  ]

  return (
    <div>
      <Space style={{ marginBottom: 16 }}>
        <RangePicker />
        <Select placeholder="操作类型" style={{ width: 120 }} allowClear>
          <Select.Option value="create_server">创建服务器</Select.Option>
          <Select.Option value="delete_server">删除服务器</Select.Option>
          <Select.Option value="create_subscription">创建账号</Select.Option>
        </Select>
      </Space>

      <Table columns={columns} dataSource={logs} rowKey="id" loading={loading} />
    </div>
  )
}
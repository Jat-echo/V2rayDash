import { useState, useEffect } from 'react'
import { Table, Button, Space, Modal, Form, Input, Select, message, Card, Tag } from 'antd'
import { CopyOutlined } from '@ant-design/icons'
import { subscriptionAPI, serverAPI, Subscription, Server } from '../../services/api'

export default function SubscriptionList() {
  const [subscriptions, setSubscriptions] = useState<Subscription[]>([])
  const [servers, setServers] = useState<Server[]>([])
  const [loading, setLoading] = useState(false)
  const [modalVisible, setModalVisible] = useState(false)
  const [linkModalVisible, setLinkModalVisible] = useState(false)
  const [currentLink, setCurrentLink] = useState('')
  const [currentEncoded, setCurrentEncoded] = useState('')
  const [form] = Form.useForm()

  useEffect(() => {
    loadData()
  }, [])

  const loadData = async () => {
    setLoading(true)
    try {
      const [subs, srvs] = await Promise.all([
        subscriptionAPI.list(),
        serverAPI.list(),
      ])
      setSubscriptions(subs || [])
      setServers(srvs || [])
    } catch (e) {
      message.error('加载失败')
      setSubscriptions([])
      setServers([])
    } finally {
      setLoading(false)
    }
  }

  const handleAdd = async (values: any) => {
    try {
      await subscriptionAPI.create(values)
      message.success('添加成功')
      setModalVisible(false)
      form.resetFields()
      loadData()
    } catch (e) {
      message.error('添加失败')
    }
  }

  const handleDelete = async (id: string) => {
    try {
      await subscriptionAPI.delete(id)
      message.success('删除成功')
      loadData()
    } catch (e) {
      message.error('删除失败')
    }
  }

  const handleGetLink = async (id: string) => {
    try {
      const { link, encoded } = await subscriptionAPI.getLink(id)
      setCurrentLink(link)
      setCurrentEncoded(encoded)
      setLinkModalVisible(true)
    } catch (e) {
      message.error('获取链接失败')
    }
  }

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text).then(() => {
      message.success('已复制到剪贴板')
    }).catch(() => {
      message.error('复制失败')
    })
  }

  const getServerName = (serverId: string) => {
    return servers.find(s => s.id === serverId)?.name || serverId
  }

  const columns = [
    { title: '名称', dataIndex: 'name' },
    { title: 'UUID', dataIndex: 'uuid', render: (v: string) => v ? v.substring(0, 8) + '...' : '-' },
    { title: '服务器', dataIndex: 'server_id', render: (id: string) => <Tag color="blue">{getServerName(id)}</Tag> },
    { title: '流量限制', dataIndex: 'traffic_limit', render: (v: number) => v ? `${(v/1024**3).toFixed(1)} GB` : '无限' },
    { title: '已用流量', dataIndex: 'traffic_used', render: (v: number) => `${(v/1024**3).toFixed(1)} GB` },
    { title: '状态', dataIndex: 'enable', render: (v: boolean) => v ? <Tag color="green">启用</Tag> : <Tag color="gray">禁用</Tag> },
    {
      title: '操作',
      render: (_: any, record: Subscription) => (
        <Space>
          <Button size="small" type="primary" onClick={() => handleGetLink(record.id)}>订阅链接</Button>
          <Button size="small" danger onClick={() => handleDelete(record.id)}>删除</Button>
        </Space>
      ),
    },
  ]

  return (
    <div className="animate-in">
      {/* Page Header */}
      <div className="page-header">
        <h1>订阅管理</h1>
        <p>创建和管理用户的订阅链接</p>
      </div>

      {/* Stats Grid */}
      <div className="stats-grid">
        <div className="stat-card animate-in animate-delay-1">
          <div className="stat-icon rose">📋</div>
          <div className="stat-content">
            <h3>{subscriptions.length}</h3>
            <p>订阅总数</p>
          </div>
        </div>
        <div className="stat-card animate-in animate-delay-2">
          <div className="stat-icon sage">✓</div>
          <div className="stat-content">
            <h3>{subscriptions.filter(s => s.enable).length}</h3>
            <p>启用中</p>
          </div>
        </div>
      </div>

      {/* Action Bar */}
      <Card className="morandi-card" style={{ marginBottom: 20 }}>
        <Button type="primary" onClick={() => setModalVisible(true)}>+ 添加订阅</Button>
      </Card>

      {/* Subscription Table */}
      <Card className="morandi-card">
        <Table columns={columns} dataSource={subscriptions} rowKey="id" loading={loading} pagination={{ pageSize: 10 }} />
      </Card>

      {/* Add Modal */}
      <Modal title="添加订阅" open={modalVisible} onCancel={() => setModalVisible(false)} footer={null}>
        <Form form={form} onFinish={handleAdd} layout="vertical" style={{ marginTop: 20 }}>
          <Form.Item name="name" label="名称" rules={[{ required: true }]}>
            <Input placeholder="例如: VIP用户" />
          </Form.Item>
          <Form.Item name="server_id" label="服务器" rules={[{ required: true }]}>
            <Select>
              {servers.map(s => <Select.Option key={s.id} value={s.id}>{s.name}</Select.Option>)}
            </Select>
          </Form.Item>
          <Form.Item name="traffic_limit" label="流量限制(GB)">
            <Input type="number" placeholder="留空表示无限流量" />
          </Form.Item>
          <Button type="primary" htmlType="submit">提交</Button>
        </Form>
      </Modal>

      {/* Link Modal */}
      <Modal
        title="订阅链接"
        open={linkModalVisible}
        onCancel={() => setLinkModalVisible(false)}
        footer={[
          <Button key="close" onClick={() => setLinkModalVisible(false)}>关闭</Button>,
        ]}
      >
        <div style={{ marginTop: 16 }}>
          <h4>原始链接</h4>
          <div style={{ display: 'flex', gap: 8, marginBottom: 16 }}>
            <Input value={currentLink} readOnly style={{ flex: 1 }} />
            <Button icon={<CopyOutlined />} onClick={() => copyToClipboard(currentLink)} />
          </div>

          <h4>Base64 编码链接 (适用于客户端)</h4>
          <div style={{ display: 'flex', gap: 8 }}>
            <Input value={currentEncoded} readOnly style={{ flex: 1 }} />
            <Button icon={<CopyOutlined />} onClick={() => copyToClipboard(currentEncoded)} />
          </div>

          <div style={{ marginTop: 16, fontSize: 12, color: 'var(--text-muted)' }}>
            <p>支持格式: ?format=vless (默认) | ?format=clash_meta | ?format=singbox</p>
          </div>
        </div>
      </Modal>
    </div>
  )
}
import { useState, useEffect } from 'react'
import { Table, Button, Space, Modal, Form, Input, Select, message } from 'antd'
import { subscriptionAPI, serverAPI, Subscription, Server } from '../../services/api'

export default function SubscriptionList() {
  const [subscriptions, setSubscriptions] = useState<Subscription[]>([])
  const [servers, setServers] = useState<Server[]>([])
  const [loading, setLoading] = useState(false)
  const [modalVisible, setModalVisible] = useState(false)
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
      setSubscriptions(subs)
      setServers(srvs)
    } catch (e) {
      message.error('加载失败')
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

  const handleGetLink = async (id: string) => {
    try {
      const { link } = await subscriptionAPI.getLink(id)
      message.success(`订阅链接: ${link}`)
    } catch (e) {
      message.error('获取链接失败')
    }
  }

  const columns = [
    { title: '名称', dataIndex: 'name' },
    { title: 'UUID', dataIndex: 'uuid', render: (v: string) => v.slice(0, 8) + '...' },
    { title: '服务器', dataIndex: 'server_id', render: (id: string) => servers.find(s => s.id === id)?.name || id },
    { title: '流量限制', dataIndex: 'traffic_limit', render: (v: number) => v ? `${(v/1024**3).toFixed(1)} GB` : '无限' },
    { title: '已用流量', dataIndex: 'traffic_used', render: (v: number) => `${(v/1024**3).toFixed(1)} GB` },
    { title: '状态', dataIndex: 'enable', render: (v: boolean) => v ? '启用' : '禁用' },
    {
      title: '操作',
      render: (_: any, record: Subscription) => (
        <Space>
          <Button size="small" onClick={() => handleGetLink(record.id)}>链接</Button>
          <Button size="small" danger onClick={() => subscriptionAPI.delete(record.id).then(loadData)}>删除</Button>
        </Space>
      ),
    },
  ]

  return (
    <div>
      <Space style={{ marginBottom: 16 }}>
        <Button type="primary" onClick={() => setModalVisible(true)}>添加账号</Button>
      </Space>

      <Table columns={columns} dataSource={subscriptions} rowKey="id" loading={loading} />

      <Modal title="添加账号" open={modalVisible} onCancel={() => setModalVisible(false)} footer={null}>
        <Form form={form} onFinish={handleAdd} layout="vertical">
          <Form.Item name="name" label="名称" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item name="server_id" label="服务器" rules={[{ required: true }]}>
            <Select>
              {servers.map(s => <Select.Option key={s.id} value={s.id}>{s.name}</Select.Option>)}
            </Select>
          </Form.Item>
          <Form.Item name="traffic_limit" label="流量限制(GB)">
            <Input type="number" />
          </Form.Item>
          <Button type="primary" htmlType="submit">提交</Button>
        </Form>
      </Modal>
    </div>
  )
}
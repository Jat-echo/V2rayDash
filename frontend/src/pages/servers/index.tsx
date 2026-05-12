import { useState, useEffect } from 'react'
import { Table, Button, Space, Modal, Form, Input, message } from 'antd'
import { serverAPI, Server } from '../../services/api'

export default function ServerList() {
  const [servers, setServers] = useState<Server[]>([])
  const [loading, setLoading] = useState(false)
  const [modalVisible, setModalVisible] = useState(false)
  const [form] = Form.useForm()

  useEffect(() => {
    loadServers()
  }, [])

  const loadServers = async () => {
    setLoading(true)
    try {
      const data = await serverAPI.list()
      setServers(data)
    } catch (e) {
      message.error('加载服务器列表失败')
    } finally {
      setLoading(false)
    }
  }

  const handleAdd = async (values: any) => {
    try {
      await serverAPI.create(values)
      message.success('添加成功')
      setModalVisible(false)
      form.resetFields()
      loadServers()
    } catch (e) {
      message.error('添加失败')
    }
  }

  const handleDelete = async (id: string) => {
    try {
      await serverAPI.delete(id)
      message.success('删除成功')
      loadServers()
    } catch (e) {
      message.error('删除失败')
    }
  }

  const columns = [
    { title: '名称', dataIndex: 'name' },
    { title: 'IP', dataIndex: 'ip' },
    { title: 'SSH端口', dataIndex: 'ssh_port' },
    { title: '状态', dataIndex: 'status' },
    { title: '创建时间', dataIndex: 'created_at' },
    {
      title: '操作',
      render: (_: any, record: Server) => (
        <Space>
          <Button size="small">安装</Button>
          <Button size="small" danger onClick={() => handleDelete(record.id)}>删除</Button>
        </Space>
      ),
    },
  ]

  return (
    <div>
      <Space style={{ marginBottom: 16 }}>
        <Button type="primary" onClick={() => setModalVisible(true)}>添加服务器</Button>
      </Space>

      <Table columns={columns} dataSource={servers} rowKey="id" loading={loading} />

      <Modal title="添加服务器" open={modalVisible} onCancel={() => setModalVisible(false)} footer={null}>
        <Form form={form} onFinish={handleAdd} layout="vertical">
          <Form.Item name="name" label="名称" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item name="ip" label="IP地址" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item name="ssh_port" label="SSH端口" initialValue={22}>
            <Input type="number" />
          </Form.Item>
          <Form.Item name="ssh_user" label="SSH用户" initialValue="root">
            <Input />
          </Form.Item>
          <Form.Item name="ssh_key" label="SSH私钥">
            <Input.TextArea rows={4} />
          </Form.Item>
          <Button type="primary" htmlType="submit">提交</Button>
        </Form>
      </Modal>
    </div>
  )
}
import { useState, useEffect } from 'react'
import { Table, Button, Space, Modal, Form, Input, Select, message, Popconfirm } from 'antd'
import { serverAPI, Server } from '../../services/api'

export default function ServerList() {
  const [servers, setServers] = useState<Server[]>([])
  const [loading, setLoading] = useState(false)
  const [modalVisible, setModalVisible] = useState(false)
  const [installModalVisible, setInstallModalVisible] = useState(false)
  const [installOutput, setInstallOutput] = useState('')
  const [eventSource, setEventSource] = useState<EventSource | null>(null)
  const [selectedServer, setSelectedServer] = useState<Server | null>(null)
  const [form] = Form.useForm()
  const [sshKeyType, setSshKeyType] = useState<string>('key')

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
      setSshKeyType('key')
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

  const handleInstall = (server: Server) => {
    setSelectedServer(server)
    setInstallOutput('')
    setInstallModalVisible(true)

    // Create SSE connection
    const es = new EventSource(`/api/servers/${server.id}/install`)
    es.onmessage = (e) => {
      setInstallOutput(prev => prev + e.data)
    }
    es.onerror = () => {
      es.close()
    }
    setEventSource(es)
  }

  const closeInstallModal = () => {
    if (eventSource) {
      eventSource.close()
      setEventSource(null)
    }
    setInstallModalVisible(false)
  }

  const copyOutput = () => {
    navigator.clipboard.writeText(installOutput)
    message.success('已复制到剪贴板')
  }

  const columns = [
    { title: '名称', dataIndex: 'name' },
    { title: 'IP', dataIndex: 'ip' },
    { title: 'SSH端口', dataIndex: 'ssh_port' },
    { title: 'SSH用户', dataIndex: 'ssh_user' },
    { title: '认证方式', dataIndex: 'ssh_key_type', render: (v: string) => v === 'password' ? '密码' : '密钥' },
    { title: '状态', dataIndex: 'status' },
    {
      title: '操作',
      render: (_: any, record: Server) => (
        <Space>
          <Button size="small" type="primary" onClick={() => handleInstall(record)}>安装</Button>
          <Popconfirm title="确定删除?" onConfirm={() => handleDelete(record.id)}>
            <Button size="small" danger>删除</Button>
          </Popconfirm>
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

      <Modal
        title="添加服务器"
        open={modalVisible}
        onCancel={() => {
          setModalVisible(false)
          form.resetFields()
          setSshKeyType('key')
        }}
        footer={null}
      >
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
          <Form.Item name="ssh_key_type" label="认证方式" initialValue="key">
            <Select onChange={(v) => setSshKeyType(v)}>
              <Select.Option value="key">SSH密钥</Select.Option>
              <Select.Option value="password">密码</Select.Option>
            </Select>
          </Form.Item>
          {sshKeyType === 'key' ? (
            <Form.Item name="ssh_key" label="SSH私钥">
              <Input.TextArea rows={4} placeholder="粘贴SSH私钥内容" />
            </Form.Item>
          ) : (
            <Form.Item name="ssh_password" label="SSH密码">
              <Input.Password />
            </Form.Item>
          )}
          <Button type="primary" htmlType="submit">提交</Button>
        </Form>
      </Modal>

      <Modal
        title={`安装 v2ray 到 ${selectedServer?.name || ''}`}
        open={installModalVisible}
        onCancel={closeInstallModal}
        footer={[
          <Button key="copy" onClick={copyOutput}>
            复制输出
          </Button>,
          <Button key="close" onClick={closeInstallModal}>
            关闭
          </Button>,
        ]}
      >
        <pre style={{
          background: '#1e1e1e',
          color: '#0f0',
          padding: 16,
          borderRadius: 4,
          maxHeight: 400,
          overflow: 'auto',
          fontFamily: 'monospace'
        }}>
          {installOutput || '正在连接...'}
        </pre>
      </Modal>
    </div>
  )
}
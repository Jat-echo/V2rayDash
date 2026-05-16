import { useState, useEffect } from 'react'
import { Table, Button, Space, Modal, Form, Input, Select, Switch, message, Popconfirm } from 'antd'
import { CopyOutlined, PlusOutlined, DeleteOutlined, EditOutlined } from '@ant-design/icons'

interface Template {
  id: string
  name: string
  description: string
  config: {
    core: string
    port: number
    uuid: string
    server_name: string
    protocols: string[]
    agent_enabled: boolean
    report_interval: number
  }
}

const PROTOCOL_OPTIONS = [
  { label: 'VLESS Reality Vision', value: 'vless_reality_vision' },
  { label: 'VLESS TCP Vision', value: 'vless_tcp_vision' },
  { label: 'Hysteria2', value: 'hysteria2' },
  { label: 'TUIC', value: 'tuic' },
]

const CORE_OPTIONS = [
  { label: 'sing-box', value: 'sing-box' },
  { label: 'xray-core', value: 'xray-core' },
]

export default function TemplateList() {
  const [templates, setTemplates] = useState<Template[]>([])
  const [loading, setLoading] = useState(false)
  const [modalVisible, setModalVisible] = useState(false)
  const [commandModalVisible, setCommandModalVisible] = useState(false)
  const [form] = Form.useForm()
  const [selectedTemplate, setSelectedTemplate] = useState<Template | null>(null)
  const [installCommand, setInstallCommand] = useState('')
  const [editingId, setEditingId] = useState<string | null>(null)

  useEffect(() => {
    loadTemplates()
  }, [])

  const loadTemplates = async () => {
    setLoading(true)
    try {
      const res = await fetch('/api/templates')
      const data = await res.json()
      setTemplates(data)
    } catch (e) {
      message.error('加载失败')
    } finally {
      setLoading(false)
    }
  }

  const handleAdd = () => {
    setEditingId(null)
    form.resetFields()
    setModalVisible(true)
  }

  const handleEdit = (record: Template) => {
    setEditingId(record.id)
    form.setFieldsValue({
      name: record.name,
      description: record.description,
      config: {
        core: record.config.core || 'sing-box',
        port: record.config.port || 443,
        server_name: record.config.server_name || 'download-installer.cdn.mozilla.net',
        protocols: record.config.protocols || [],
        agent_enabled: record.config.agent_enabled ?? true,
        report_interval: record.config.report_interval || 30,
      },
    })
    setModalVisible(true)
  }

  const handleSubmit = async (values: any) => {
    try {
      if (editingId) {
        // Update not implemented yet - could add later
        message.info('编辑功能暂未实现')
        return
      }
      await fetch('/api/templates', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(values),
      })
      message.success('添加成功')
      setModalVisible(false)
      form.resetFields()
      loadTemplates()
    } catch (e) {
      message.error('添加失败')
    }
  }

  const handleDelete = async (id: string) => {
    try {
      await fetch(`/api/templates/${id}`, { method: 'DELETE' })
      message.success('删除成功')
      loadTemplates()
    } catch (e) {
      message.error('删除失败')
    }
  }

  const generateInstallCommand = (template: Template) => {
    const serverId = crypto.randomUUID()
    const psk = btoa(Math.random().toString()).slice(0, 32)
    const baseURL = window.location.origin
    const templateName = template.name.toLowerCase().replace(/\s+/g, '-')
    return `curl -sL ${baseURL}/install.sh | bash -s -- \\
  --agent \\
  --template ${templateName} \\
  --url ${baseURL} \\
  --id ${serverId} \\
  --psk ${psk}`
  }

  const handleGenerateCommand = (template: Template) => {
    setSelectedTemplate(template)
    setInstallCommand(generateInstallCommand(template))
    setCommandModalVisible(true)
  }

  const copyCommand = () => {
    navigator.clipboard.writeText(installCommand)
    message.success('已复制到剪贴板')
  }

  const columns = [
    { title: '名称', dataIndex: 'name', key: 'name' },
    { title: '描述', dataIndex: 'description', key: 'description' },
    { title: '核心', dataIndex: ['config', 'core'], key: 'core' },
    { title: '端口', dataIndex: ['config', 'port'], key: 'port' },
    { title: 'Agent', dataIndex: ['config', 'agent_enabled'], key: 'agent_enabled', render: (v: boolean) => v ? '是' : '否' },
    {
      title: '操作',
      key: 'action',
      render: (_: any, record: Template) => (
        <Space>
          <Button size="small" icon={<EditOutlined />} onClick={() => handleEdit(record)}>编辑</Button>
          <Button size="small" type="primary" onClick={() => handleGenerateCommand(record)}>生成命令</Button>
          <Popconfirm title="确定删除?" onConfirm={() => handleDelete(record.id)}>
            <Button size="small" danger icon={<DeleteOutlined />}>删除</Button>
          </Popconfirm>
        </Space>
      ),
    },
  ]

  return (
    <div style={{ padding: 24 }}>
      <Space style={{ marginBottom: 16 }}>
        <Button type="primary" icon={<PlusOutlined />} onClick={handleAdd}>新建模板</Button>
      </Space>

      <Table columns={columns} dataSource={templates} rowKey="id" loading={loading} />

      <Modal
        title={editingId ? '编辑模板' : '新建模板'}
        open={modalVisible}
        onCancel={() => setModalVisible(false)}
        footer={null}
        width={600}
      >
        <Form form={form} onFinish={handleSubmit} layout="vertical">
          <Form.Item name="name" label="模板名称" rules={[{ required: true, message: '请输入模板名称' }]}>
            <Input />
          </Form.Item>
          <Form.Item name="description" label="描述">
            <Input.TextArea />
          </Form.Item>
          <Form.Item label="配置" name="config">
            <Form.Item noStyle>
              <Input.Group compact>
                <Form.Item name={['config', 'core']} label="核心" initialValue="sing-box" style={{ width: 200 }}>
                  <Select options={CORE_OPTIONS} />
                </Form.Item>
                <Form.Item name={['config', 'port']} label="端口" initialValue={443} style={{ width: 120 }}>
                  <Input type="number" />
                </Form.Item>
              </Input.Group>
            </Form.Item>
            <Form.Item noStyle>
              <Input.Group compact>
                <Form.Item name={['config', 'server_name']} label="Reality目标域名" initialValue="download-installer.cdn.mozilla.net" style={{ flex: 1 }}>
                  <Input />
                </Form.Item>
              </Input.Group>
            </Form.Item>
            <Form.Item name={['config', 'protocols']} label="协议" initialValue={['vless_reality_vision']}>
              <Select mode="multiple" options={PROTOCOL_OPTIONS} />
            </Form.Item>
            <Form.Item name={['config', 'agent_enabled']} label="启用Agent" valuePropName="checked" initialValue={true}>
              <Switch />
            </Form.Item>
            <Form.Item name={['config', 'report_interval']} label="上报间隔(秒)" initialValue={30}>
              <Input type="number" min={10} max={300} />
            </Form.Item>
          </Form.Item>
          <Button type="primary" htmlType="submit">提交</Button>
        </Form>
      </Modal>

      <Modal
        title="安装命令"
        open={commandModalVisible}
        onCancel={() => setCommandModalVisible(false)}
        footer={[
          <Button key="copy" icon={<CopyOutlined />} type="primary" onClick={copyCommand}>
            复制命令
          </Button>,
          <Button key="close" onClick={() => setCommandModalVisible(false)}>
            关闭
          </Button>,
        ]}
      >
        <pre style={{ background: '#f5f5f5', padding: 16, borderRadius: 4, overflow: 'auto' }}>
          {installCommand}
        </pre>
      </Modal>
    </div>
  )
}
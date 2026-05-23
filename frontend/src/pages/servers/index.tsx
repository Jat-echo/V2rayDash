import { useState, useEffect, useRef, useCallback } from 'react'
import { Table, Button, Space, Modal, Form, Input, Select, message, Popconfirm, Tag, Card, Alert } from 'antd'
import { serverAPI, templateAPI, accountAPI, Server, Template, TemplateConfig, Account } from '../../services/api'

// Convert ANSI escape codes to HTML with colors
function ansiToHtml(text: string): string {
  const lines = text.split('\n')
  const styledLines = lines.map(line => {
    let escaped = line
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
    escaped = escaped
      .replace(/\[1;31m/g, '<span style="color:#ff4d4f;font-weight:bold">')
      .replace(/\[31m/g, '<span style="color:#ff4d4f">')
      .replace(/\[1;36m/g, '<span style="color:#13c2c2;font-weight:bold">')
      .replace(/\[32m/g, '<span style="color:#52c41a">')
      .replace(/\[33m/g, '<span style="color:#faad14">')
      .replace(/\[34m/g, '<span style="color:#1677ff">')
      .replace(/\[35m/g, '<span style="color:#eb2f96">')
      .replace(/\[36m/g, '<span style="color:#13c2c2">')
      .replace(/\[1m/g, '<span style="font-weight:bold">')
      .replace(/\[0m/g, '</span>')
    const openCount = (escaped.match(/<span/g) || []).length
    const closeCount = (escaped.match(/<\/span>/g) || []).length
    if (openCount > closeCount) {
      escaped += '</span>'.repeat(openCount - closeCount)
    }
    return escaped
  })
  return styledLines.join('<br/>')
}

// Protocol name mapping
const protocolNames: Record<string, string> = {
  'vless_tcp': 'VLESS TCP',
  'vless_reality_vision': 'VLESS Reality',
  'vless_ws': 'VLESS WebSocket',
  'trojan': 'Trojan',
  'trojan_grpc': 'Trojan gRPC',
  'vmess_ws': 'VMess WS',
  'hysteria2': 'Hysteria2',
  'tuic': 'Tuic',
}

function getProtocolName(proto: string): string {
  return protocolNames[proto] || proto
}

const defaultConfig: TemplateConfig = {
  core: 'xray-core',
  uuid: '',
  protocols: ['vless_reality_vision'],
  agent_enabled: false,
  report_interval: 30,
}

export default function ServerList() {
  const [servers, setServers] = useState<Server[]>([])
  const [templates, setTemplates] = useState<Template[]>([])
  const [loading, setLoading] = useState(false)
  const [modalVisible, setModalVisible] = useState(false)
  const [configModalVisible, setConfigModalVisible] = useState(false)
  const [installModalVisible, setInstallModalVisible] = useState(false)
  const [installOutput, setInstallOutput] = useState('')
  const [selectedServer, setSelectedServer] = useState<Server | null>(null)
  const [selectedTemplate, setSelectedTemplate] = useState<Template | null>(null)
  const [installConfig, setInstallConfig] = useState<TemplateConfig>(defaultConfig)
  const [form] = Form.useForm()
  const [sshKeyType, setSshKeyType] = useState<string>('key')
  const [installing, setInstalling] = useState(false)
  const [accountModalVisible, setAccountModalVisible] = useState(false)
  const [accounts, setAccounts] = useState<Account[]>([])
  const [selectedServerForAccounts, setSelectedServerForAccounts] = useState<Server | null>(null)
  const [addAccountForm] = Form.useForm()
  const installOutputRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    loadServers()
    loadTemplates()
  }, [])

  // 自动滚动到最新输出
  useEffect(() => {
    if (installOutputRef.current) {
      installOutputRef.current.scrollTop = installOutputRef.current.scrollHeight
    }
  }, [installOutput])

  const loadServers = async () => {
    setLoading(true)
    try {
      const data = await serverAPI.list()
      setServers(data || [])
    } catch (e) {
      message.error('加载服务器列表失败')
      setServers([])
    } finally {
      setLoading(false)
    }
  }

  const loadTemplates = async () => {
    try {
      const data = await templateAPI.list()
      setTemplates(data || [])
    } catch (e) {
      setTemplates([])
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

  const handleInstallClick = (server: Server) => {
    setSelectedServer(server)
    setSelectedTemplate(null)
    setInstallConfig({
      ...defaultConfig,
    })
    setConfigModalVisible(true)
  }

  const handleTemplateChange = (templateId: number) => {
    const tmpl = templates.find(t => t.id === templateId)
    if (tmpl) {
      setSelectedTemplate(tmpl)
      setInstallConfig(tmpl.config)
    }
  }

  const confirmReinstall = () => {
    setConfigModalVisible(false)
    setInstallOutput('')
    setInstallModalVisible(true)
    setInstalling(true)

    fetch(`/api/servers/${selectedServer?.id}/install`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        core: installConfig.core,
        uuid: installConfig.uuid,
        protocols: installConfig.protocols,
      })
    }).then(response => {
      if (!response.ok) {
        setInstallOutput('\n❌ 连接失败，状态码: ' + response.status)
        setInstalling(false)
        return
      }

      // 使用 ReadableStream 实时读取 SSE 流
      const reader = response.body?.getReader()
      if (!reader) {
        setInstallOutput('\n❌ 浏览器不支持流式读取')
        setInstalling(false)
        return
      }

      const decoder = new TextDecoder()

      function read() {
        reader.read().then(({ done, value }) => {
          if (done) {
            setInstalling(false)
            // 检查安装结果
            if (installOutput.includes('✓') || installOutput.includes('安装完成')) {
              message.success('安装完成！')
            } else if (installOutput.includes('[ERROR]')) {
              message.error('安装失败，请检查输出')
            }
            return
          }

          const chunk = decoder.decode(value, { stream: true })
          setInstallOutput(prev => prev + chunk)
          read()
        }).catch(err => {
          setInstallOutput(prev => prev + `\n❌ 读取错误: ${err}\n`)
          setInstalling(false)
        })
      }

      read()
    }).catch(err => {
      setInstallOutput(`\n❌ 连接失败: ${err}\n`)
      setInstalling(false)
    })
  }

  const closeInstallModal = () => {
    setInstallModalVisible(false)
  }

  const copyOutput = () => {
    if (navigator.clipboard) {
      navigator.clipboard.writeText(installOutput).then(() => {
        message.success('已复制到剪贴板')
      }).catch(() => {
        message.error('复制失败')
      })
    }
  }

  const loadAccounts = async (serverId: string) => {
    try {
      const data = await accountAPI.listByServer(serverId)
      setAccounts(data || [])
    } catch (e) {
      setAccounts([])
    }
  }

  const handleOpenAccountModal = (server: Server) => {
    setSelectedServerForAccounts(server)
    loadAccounts(server.id)
    setAccountModalVisible(true)
  }

  const handleImportFromRemote = async () => {
    if (!selectedServerForAccounts) return
    try {
      const result = await accountAPI.import(selectedServerForAccounts.id)
      message.success(`成功导入 ${result.accounts?.length || 0} 个账号`)
      loadAccounts(selectedServerForAccounts.id)
    } catch (e) {
      message.error('导入失败')
    }
  }

  const handleAddAccount = async (values: any) => {
    if (!selectedServerForAccounts) return
    try {
      await accountAPI.create(selectedServerForAccounts.id, values)
      message.success('添加成功')
      addAccountForm.resetFields()
      loadAccounts(selectedServerForAccounts.id)
    } catch (e) {
      message.error('添加失败')
    }
  }

  const handleDeleteAccount = async (id: string) => {
    try {
      await accountAPI.delete(id)
      message.success('删除成功')
      if (selectedServerForAccounts) {
        loadAccounts(selectedServerForAccounts.id)
      }
    } catch (e) {
      message.error('删除失败')
    }
  }

  const handleDownloadSubscription = async (accountId: string, type: string) => {
    try {
      const content = await accountAPI.subscribe(accountId, type)
      const blob = new Blob([content], { type: 'text/plain' })
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = `subscription-${type}-${Date.now()}.txt`
      a.click()
      URL.revokeObjectURL(url)
      message.success('下载成功')
    } catch (e) {
      message.error('下载失败')
    }
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
          <Button size="small" type="primary" onClick={() => handleInstallClick(record)}>安装</Button>
          <Button size="small" onClick={() => handleOpenAccountModal(record)}>账号管理</Button>
          <Popconfirm title="确定删除?" onConfirm={() => handleDelete(record.id)}>
            <Button size="small" danger>删除</Button>
          </Popconfirm>
        </Space>
      ),
    },
  ]

  return (
    <div className="animate-in">
      {/* Page Header */}
      <div className="page-header">
        <h1>服务器管理</h1>
        <p>管理您的 V2ray 服务器和账号</p>
      </div>

      {/* Stats Grid */}
      <div className="stats-grid">
        <div className="stat-card animate-in animate-delay-1">
          <div className="stat-icon rose">⚡</div>
          <div className="stat-content">
            <h3>{servers.length}</h3>
            <p>服务器总数</p>
          </div>
        </div>
        <div className="stat-card animate-in animate-delay-2">
          <div className="stat-icon sage">✓</div>
          <div className="stat-content">
            <h3>{servers.filter(s => s.status === 'online').length}</h3>
            <p>在线服务器</p>
          </div>
        </div>
        <div className="stat-card animate-in animate-delay-3">
          <div className="stat-icon sky">📋</div>
          <div className="stat-content">
            <h3>{templates.length}</h3>
            <p>可用模板</p>
          </div>
        </div>
      </div>

      {/* Action Bar */}
      <Card className="morandi-card" style={{ marginBottom: 20 }}>
        <Button type="primary" onClick={() => setModalVisible(true)}>+ 添加服务器</Button>
      </Card>

      {/* Server Table */}
      <Card className="morandi-card">
        <Table columns={columns} dataSource={servers} rowKey="id" loading={loading} pagination={{ pageSize: 10 }} />
      </Card>

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

      {/* Config Modal */}
      <Modal
        title={`配置安装 - ${selectedServer?.name || ''}`}
        open={configModalVisible}
        onCancel={() => setConfigModalVisible(false)}
        onOk={confirmReinstall}
        okText="开始安装"
        cancelText="取消"
      >
        <Alert
          message="警告：重新安装会覆盖远程服务器上的所有配置"
          description="远程服务器的现有账号、协议配置会被清除。控制中心保存的账号不受影响。建议重新安装前先从远程导入账号。"
          type="warning"
          showIcon
          style={{ marginBottom: 16 }}
        />
        <Form layout="vertical">
          <Form.Item label="选择模板">
            <Select
              placeholder="选择已有模板或留空自定义"
              allowClear
              onChange={(v) => v ? handleTemplateChange(v) : setSelectedTemplate(null)}
            >
              {templates.map(t => (
                <Select.Option key={t.id} value={t.id}>{t.name}</Select.Option>
              ))}
            </Select>
          </Form.Item>

          <Form.Item label="协议类型">
            <Select
              mode="multiple"
              value={installConfig.protocols}
              onChange={(v) => setInstallConfig({ ...installConfig, protocols: v })}
            >
              <Select.Option value="vless_reality_vision">VLESS + Reality + Vision</Select.Option>
              <Select.Option value="vless_tcp_vision">VLESS + TCP + Vision</Select.Option>
              <Select.Option value="vmess_ws">VMess + WebSocket</Select.Option>
              <Select.Option value="trojan">Trojan</Select.Option>
              <Select.Option value="hysteria2">Hysteria2</Select.Option>
            </Select>
          </Form.Item>

          <Form.Item label="UUID (留空自动生成)">
            <Input
              value={installConfig.uuid}
              onChange={(e) => setInstallConfig({ ...installConfig, uuid: e.target.value })}
              placeholder="自动生成"
            />
          </Form.Item>

          <Form.Item label="核心">
            <Select
              value={installConfig.core}
              onChange={(v) => setInstallConfig({ ...installConfig, core: v })}
            >
              <Select.Option value="xray-core">Xray-core</Select.Option>
              <Select.Option value="sing-box">Sing-box</Select.Option>
            </Select>
          </Form.Item>
        </Form>
      </Modal>

      {/* Install Output Modal */}
      <Modal
        title={`安装 v2ray 到 ${selectedServer?.name || ''}`}
        open={installModalVisible}
        onCancel={closeInstallModal}
        width={900}
        footer={[
          <Button key="copy" onClick={copyOutput} disabled={installing}>
            复制输出
          </Button>,
          <Button key="close" onClick={closeInstallModal}>
            关闭
          </Button>,
        ]}
      >
        {installing && <div style={{ color: '#888', marginBottom: 8 }}>正在安装，请稍候...（这可能需要10-30秒）</div>}
        <pre
          ref={installOutputRef}
          style={{
          background: '#1e1e1e',
          color: '#ffffff',
          padding: 16,
          borderRadius: 8,
          height: 500,
          overflow: 'auto',
          fontFamily: 'monospace',
          fontSize: 13,
          whiteSpace: 'pre-wrap',
          wordBreak: 'break-all'
        }}
          dangerouslySetInnerHTML={{ __html: installOutput ? ansiToHtml(installOutput) : '准备中...' }}
        />
      </Modal>

      {/* Account Modal */}
      <Modal
        title={`账号管理 - ${selectedServerForAccounts?.name || ''}`}
        open={accountModalVisible}
        onCancel={() => setAccountModalVisible(false)}
        width={750}
        footer={null}
      >
        <div style={{ marginBottom: 16 }}>
          <Space>
            <Button type="primary" onClick={handleImportFromRemote}>从远程导入</Button>
            <Button onClick={() => addAccountForm.resetFields()}>清空</Button>
          </Space>
        </div>

        <Form form={addAccountForm} onFinish={handleAddAccount} layout="inline" style={{ marginBottom: 16 }}>
          <Form.Item name="email" label="备注" rules={[{ required: true }]}>
            <Input placeholder="user@example.com" style={{ width: 150 }} />
          </Form.Item>
          <Form.Item name="protocols" label="协议" rules={[{ required: true }]}>
            <Select mode="multiple" style={{ width: 200 }}>
              <Select.Option value="vless_tcp">VLESS TCP</Select.Option>
              <Select.Option value="vless_reality_vision">VLESS Reality Vision</Select.Option>
              <Select.Option value="trojan">Trojan</Select.Option>
            </Select>
          </Form.Item>
          <Form.Item>
            <Button type="primary" htmlType="submit">确定</Button>
          </Form.Item>
        </Form>

        <Table
          dataSource={accounts}
          rowKey="id"
          size="small"
          columns={[
            { title: '备注', dataIndex: 'email' },
            { title: 'UUID', dataIndex: 'uuid', render: (v: string) => v ? v.substring(0, 8) + '...' : '-' },
            { title: '协议', dataIndex: 'protocols', render: (p: string[]) => p?.map(v => <Tag key={v}>{getProtocolName(v)}</Tag>) },
            { title: '状态', dataIndex: 'enabled', render: (v: boolean) => v ? '启用' : '禁用' },
            {
              title: '操作',
              render: (_: any, record: Account) => (
                <Space>
                  <Popconfirm title="确定下载VLESS订阅?" onConfirm={() => handleDownloadSubscription(record.id, 'vless')}>
                    <Button size="small">VLESS</Button>
                  </Popconfirm>
                  <Popconfirm title="确定下载Clash订阅?" onConfirm={() => handleDownloadSubscription(record.id, 'clash_meta')}>
                    <Button size="small">Clash</Button>
                  </Popconfirm>
                  <Popconfirm title="确定删除?" onConfirm={() => handleDeleteAccount(record.id)}>
                    <Button size="small" danger>删除</Button>
                  </Popconfirm>
                </Space>
              ),
            },
          ]}
        />
      </Modal>
    </div>
  )
}
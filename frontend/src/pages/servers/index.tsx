import { useState, useEffect, useRef } from 'react'
import { Table, Button, Space, Modal, Form, Input, InputNumber, Select, message, Popconfirm, Tag, Card, Alert, Segmented } from 'antd'
import { serverAPI, accountAPI, logAPI, Server, Account, NodeStatusResponse, MetricPoint, BandwidthPoint } from '../../services/api'
import { formatBytes } from '../../utils/format'

// Convert ANSI escape codes to HTML with colors
function ansiToHtml(text: string): string {
  // Normalize CRLF from PTY output to avoid double blank lines
  const lines = text.replace(/\r\n/g, '\n').replace(/\r/g, '').split('\n')
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

const defaultConfig = {
  core: 'xray-core',
  uuid: '',
  protocols: ['vless_reality_vision'],
}

export default function ServerList() {
  const [servers, setServers] = useState<Server[]>([])
  const [loading, setLoading] = useState(false)
  const [modalVisible, setModalVisible] = useState(false)
  const [configModalVisible, setConfigModalVisible] = useState(false)
  const [installModalVisible, setInstallModalVisible] = useState(false)
  const [installOutput, setInstallOutput] = useState('')
  const [selectedServer, setSelectedServer] = useState<Server | null>(null)
  const [installConfig, setInstallConfig] = useState(defaultConfig)
  const [form] = Form.useForm()
  const [sshKeyType, setSshKeyType] = useState<string>('key')
  const [installing, setInstalling] = useState(false)
  const [accountModalVisible, setAccountModalVisible] = useState(false)
  const [accounts, setAccounts] = useState<Account[]>([])
  const [selectedServerForAccounts, setSelectedServerForAccounts] = useState<Server | null>(null)
  const [addAccountForm] = Form.useForm()
  const installOutputRef = useRef<HTMLDivElement>(null)
  const [editModalVisible, setEditModalVisible] = useState(false)
  const [editingServer, setEditingServer] = useState<Server | null>(null)
  const [editForm] = Form.useForm()
  const [editSshKeyType, setEditSshKeyType] = useState<string>('key')
  const [timeRange, setTimeRange] = useState('1h')
  const [statuses, setStatuses] = useState<Map<string, NodeStatusResponse>>(new Map())
  const [restartingXray, setRestartingXray] = useState<string | null>(null)

  const TIME_RANGES = [
    { label: '1H', value: '1h' },
    { label: '3H', value: '3h' },
    { label: '6H', value: '6h' },
    { label: '12H', value: '12h' },
    { label: '1天', value: '1d' },
    { label: '3天', value: '3d' },
  ]

  // Preload plotly when page loads
  useEffect(() => {
    plotlyPromise.then(() => {})
  }, [])

  useEffect(() => {
    loadServers()
  }, [timeRange])

  // 自动滚动到最新输出
  useEffect(() => {
    if (installOutputRef.current) {
      installOutputRef.current.scrollTop = installOutputRef.current.scrollHeight
    }
    // 检查安装结果
    if (!installing) return
    if (installOutput.includes('[ERROR]') || installOutput.includes('失败') || installOutput.includes('✗')) {
      message.error('安装失败，请查看输出')
    } else if (installOutput.includes('安装完成') || installOutput.includes('[OK]') || installOutput.includes('Reality配置已保存') || installOutput.includes('安装成功')) {
      message.success('安装完成！')
    }
  }, [installOutput, installing])

  const loadServers = async () => {
    setLoading(true)
    try {
      const [data, statusData] = await Promise.all([
        serverAPI.list(),
        logAPI.getNodeStatuses(timeRange)
      ])
      setServers(data || [])

      const statusMap = new Map<string, NodeStatusResponse>()
      if (statusData && statusData.length > 0) {
        statusData.forEach((s: NodeStatusResponse) => statusMap.set(s.server_id, s))
      }
      setStatuses(statusMap)
    } catch (e) {
      message.error('加载服务器列表失败')
      setServers([])
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

  const handleRestartXray = async (serverId: string, e: React.MouseEvent) => {
    e.stopPropagation()
    setRestartingXray(serverId)
    try {
      await serverAPI.restartXray(serverId)
      message.success('xray 已重启')
    } catch {
      message.error('重启失败')
    } finally {
      setRestartingXray(null)
    }
  }

  const handleEditClick = (server: Server) => {
    setEditingServer(server)
    setEditSshKeyType(server.ssh_key_type || 'key')
    editForm.setFieldsValue({
      name: server.name,
      ip: server.ip,
      ssh_port: server.ssh_port,
      ssh_user: server.ssh_user,
      ssh_key_type: server.ssh_key_type,
      ssh_key: server.ssh_key,
      ssh_password: server.ssh_password,
    })
    setEditModalVisible(true)
  }

  const handleEdit = async (values: any) => {
    if (!editingServer) return
    try {
      await serverAPI.update(editingServer.id, values)
      message.success('编辑成功')
      setEditModalVisible(false)
      editForm.resetFields()
      setEditSshKeyType('key')
      setEditingServer(null)
      loadServers()
    } catch (e) {
      message.error('编辑失败')
    }
  }

  const handleInstallClick = (server: Server) => {
    setSelectedServer(server)
    setInstallConfig({
      ...defaultConfig,
    })
    setConfigModalVisible(true)
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
    const text = installOutput
    if (navigator.clipboard) {
      navigator.clipboard.writeText(text).then(() => {
        message.success('已复制到剪贴板')
      }).catch(() => fallbackCopy(text))
    } else {
      fallbackCopy(text)
    }
  }

  const fallbackCopy = (text: string) => {
    const textarea = document.createElement('textarea')
    textarea.value = text
    textarea.style.cssText = 'position:fixed;opacity:0;top:0;left:0'
    document.body.appendChild(textarea)
    textarea.focus()
    textarea.select()
    try {
      document.execCommand('copy')
      message.success('已复制到剪贴板')
    } catch {
      message.error('复制失败，请手动选择文本复制')
    }
    document.body.removeChild(textarea)
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
      await accountAPI.create(selectedServerForAccounts.id, {
        ...values,
        protocols: ['vless_reality_vision'],
      })
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
      title: 'xray',
      render: (_: any, record: Server) => {
        const st = statuses.get(record.id)
        const v2ray = st?.current?.v2ray_status
        const restarts = st?.v2ray_restarts ?? 0
        if (!v2ray) return <Tag color="default">-</Tag>
        return (
          <Space size={4}>
            <Tag color={v2ray === 'running' ? 'green' : 'red'}>{v2ray === 'running' ? '运行中' : '已停止'}</Tag>
            {restarts > 0 && <Tag color="orange">重启 {restarts} 次</Tag>}
          </Space>
        )
      }
    },
    {
      title: '操作',
      render: (_: any, record: Server) => (
        <Space>
          <Button size="small" type="primary" onClick={(e) => { e.stopPropagation(); handleInstallClick(record); }}>安装</Button>
          <Button size="small" onClick={(e) => { e.stopPropagation(); handleEditClick(record); }}>编辑</Button>
          <Button size="small" onClick={(e) => { e.stopPropagation(); handleOpenAccountModal(record); }}>账号管理</Button>
          <Button
            size="small"
            loading={restartingXray === record.id}
            onClick={(e) => handleRestartXray(record.id, e)}
          >重启服务</Button>
          <Popconfirm title="确定删除?" onConfirm={() => handleDelete(record.id)}>
            <Button size="small" danger onClick={(e) => e.stopPropagation()}>删除</Button>
          </Popconfirm>
        </Space>
      ),
    },
  ]

  return (
    <div className="animate-in">
      {/* Page Header */}
      <div className="page-header" style={{ display: 'flex', alignItems: 'flex-start', justifyContent: 'space-between' }}>
        <div>
          <h1>服务器管理</h1>
          <p>管理您的 V2ray 服务器和账号</p>
        </div>
        <Button type="primary" onClick={() => setModalVisible(true)} style={{ marginTop: 8 }}>+ 添加服务器</Button>
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
      </div>

      {/* Server Table */}
      <Card className="morandi-card">
        <Table
          columns={columns}
          dataSource={servers}
          rowKey="id"
          loading={loading}
          pagination={{ pageSize: 10 }}
          expandable={{
            expandedRowRender: (record: Server) => {
              const status = statuses.get(record.id)
              const cpu = status?.current?.cpu_percent || 0
              const memory = status?.current?.memory_percent || 0
              if (!status) return <div className="empty-monitor">等待数据...</div>
              return (
                <div className="metrics-row">
                  <div className="metric-cell">
                    <div className="metric-label">CPU使用率</div>
                    <div className="metric-value">{cpu.toFixed(1)}%</div>
                    <div className="chart-area">
                      <PlotlyLine
                        data={status.metrics.cpu.map((p: MetricPoint) => ({ time: p.time, value: p.value }))}
                        color="#1677FF"
                        type="percent"
                      />
                    </div>
                  </div>
                  <div className="metric-cell">
                    <div className="metric-label">内存使用率</div>
                    <div className="metric-value">{memory.toFixed(1)}%</div>
                    <div className="chart-area">
                      <PlotlyLine
                        data={status.metrics.memory.map((p: MetricPoint) => ({ time: p.time, value: p.value }))}
                        color="#722ED1"
                        type="percent"
                      />
                    </div>
                  </div>
                  <div className="metric-cell">
                    <div className="metric-label">入网带宽</div>
                    <div className="metric-value">{formatBytes(status.current?.bandwidth_in || 0)}</div>
                    <div className="chart-area">
                      <PlotlyLine
                        data={status.metrics.bandwidth_in.map((p: BandwidthPoint) => ({ time: p.time, value: p.value }))}
                        color="#1677FF"
                        type="bandwidth"
                      />
                    </div>
                  </div>
                  <div className="metric-cell">
                    <div className="metric-label">出网带宽</div>
                    <div className="metric-value">{formatBytes(status.current?.bandwidth_out || 0)}</div>
                    <div className="chart-area">
                      <PlotlyLine
                        data={status.metrics.bandwidth_out.map((p: BandwidthPoint) => ({ time: p.time, value: p.value }))}
                        color="#FA541C"
                        type="bandwidth"
                      />
                    </div>
                  </div>
                </div>
              )
            },
            expandRowByClick: false,
          }}
        />
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
            <InputNumber min={1} max={65535} style={{ width: '100%' }} />
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
            <Input placeholder="账号名称" style={{ width: 200 }} />
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

      {/* Edit Server Modal */}
      <Modal
        title={`编辑服务器 - ${editingServer?.name || ''}`}
        open={editModalVisible}
        onCancel={() => {
          setEditModalVisible(false)
          editForm.resetFields()
          setEditSshKeyType('key')
          setEditingServer(null)
        }}
        footer={null}
      >
        <Form form={editForm} onFinish={handleEdit} layout="vertical">
          <Form.Item name="name" label="名称" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item name="ip" label="IP地址" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item name="ssh_port" label="SSH端口" rules={[{ required: true }]}>
            <InputNumber min={1} max={65535} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item name="ssh_user" label="SSH用户" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item name="ssh_key_type" label="认证方式" rules={[{ required: true }]}>
            <Select onChange={(v) => setEditSshKeyType(v)}>
              <Select.Option value="key">SSH密钥</Select.Option>
              <Select.Option value="password">密码</Select.Option>
            </Select>
          </Form.Item>
          {editSshKeyType === 'key' ? (
            <Form.Item name="ssh_key" label="SSH私钥">
              <Input.TextArea rows={4} placeholder="粘贴SSH私钥内容" />
            </Form.Item>
          ) : (
            <Form.Item name="ssh_password" label="SSH密码">
              <Input.Password />
            </Form.Item>
          )}
          <Button type="primary" htmlType="submit">保存</Button>
        </Form>
      </Modal>

      {/* Monitor Styles */}
      <style>{`
        .metrics-row {
          display: grid;
          grid-template-columns: repeat(4, 1fr);
        }
        @media (max-width: 1100px) {
          .metrics-row { grid-template-columns: repeat(2, 1fr); }
        }
        @media (max-width: 700px) {
          .metrics-row { grid-template-columns: 1fr; }
        }
        .metric-cell {
          padding: 12px;
          border-right: 1px solid #E8E8E8;
        }
        .metric-cell:last-child { border-right: none; }
        .metric-label {
          font-size: 12px;
          color: #595959;
          margin-bottom: 4px;
        }
        .metric-value {
          font-size: 18px;
          font-weight: 500;
          font-family: 'JetBrains Mono', monospace;
          color: #262626;
          margin-bottom: 8px;
        }
        .chart-area {
          height: 110px;
        }
        .empty-monitor {
          padding: 40px;
          text-align: center;
          color: #8C8C8C;
          font-size: 13px;
        }
      `}</style>
    </div>
  )
}

function PlotlyLine({ data, color, type = 'value' }: { data: { time: string; value: number }[]; color: string; type?: 'percent' | 'bandwidth' }) {
  const containerRef = useRef<HTMLDivElement>(null)

  const formatHoverValue = (y: number) => {
    if (type === 'percent') {
      return y.toFixed(2) + '%'
    }
    if (type === 'bandwidth') {
      const bytesPerSec = y / 30
      if (bytesPerSec === 0) return '0 B/s'
      const k = 1024
      const sizes = ['B/s', 'KB/s', 'MB/s', 'GB/s']
      const i = Math.floor(Math.log(Math.abs(bytesPerSec)) / Math.log(k))
      const idx = Math.min(i, sizes.length - 1)
      return (bytesPerSec / Math.pow(k, idx)).toFixed(2) + ' ' + sizes[idx]
    }
    return y.toFixed(2)
  }

  useEffect(() => {
    if (!containerRef.current || !data || data.length === 0) return

    const chart = containerRef.current
    let destroyed = false

    plotlyPromise.then((Plotly) => {
      if (destroyed || !containerRef.current) return

      const trace = {
        x: data.map(p => p.time),
        y: data.map(p => p.value),
        type: 'scatter' as const,
        mode: 'lines' as const,
        line: { color, width: 1.5, shape: 'spline' as const },
        fill: 'tozeroy' as const,
        fillcolor: color + '20',
        hovertemplate: '%{customdata}<extra></extra>',
        customdata: data.map(p => formatHoverValue(p.value)),
      }

      const layout = {
        margin: { t: 5, r: 5, b: 24, l: 38 },
        paper_bgcolor: 'transparent',
        plot_bgcolor: 'transparent',
        font: { color: '#8C8C8C', family: 'Noto Sans SC, sans-serif', size: 9 },
        xaxis: {
          showgrid: false,
          gridcolor: '#F0F0F0',
          linecolor: '#D9D9D9',
          tickcolor: '#D9D9D9',
          ticks: 'outside',
          tickfont: { size: 9 },
          showline: true,
          zeroline: false,
          tickformat: '%H:%M',
        },
        yaxis: {
          showgrid: true,
          gridcolor: '#F5F5F5',
          linecolor: 'transparent',
          tickcolor: 'transparent',
          ticks: 'outside',
          tickfont: { size: 9 },
          showline: false,
          zeroline: false,
          rangemode: 'tozero' as const,
        },
        showlegend: false,
        hovermode: 'x unified',
        hoverlabel: {
          bgcolor: '#fff',
          bordercolor: color,
          borderwidth: 1,
          font: { size: 11, color: '#595959' },
        },
      }

      const config = {
        responsive: true,
        displayModeBar: false,
      }

      Plotly.newPlot(chart, [trace], layout, config)
    })

    return () => {
      destroyed = true
      if (containerRef.current) {
        try {
          plotlyPromise.then((P) => P.purge(chart))
        } catch {}
      }
    }
  }, [data, color, type])

  if (!data || data.length === 0) {
    return <div style={{ height: '100%', display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#8C8C8C', fontSize: 11 }}>暂无数据</div>
  }

  return <div ref={containerRef} style={{ width: '100%', height: '100%' }} />
}

// Shared promise to avoid multiple imports
const plotlyPromise = import('plotly.js-dist-min')
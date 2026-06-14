import { useState, useEffect, useRef } from 'react'
import { Table, Button, Space, Modal, Form, Input, InputNumber, Select, message, Popconfirm, Tag, Card, Alert, Segmented, Tabs, Spin, Checkbox } from 'antd'
import { CloudUploadOutlined, EditOutlined, TeamOutlined, ReloadOutlined, DeleteOutlined, UploadOutlined, HddOutlined, CheckCircleOutlined, ThunderboltOutlined } from '@ant-design/icons'
import { serverAPI, accountAPI, logAPI, subscriptionAPI, Server, Account, Subscription, NodeStatusResponse, MetricPoint, BandwidthPoint } from '../../services/api'
import { formatBytes } from '../../utils/format'
import { FlagName } from '../../components/FlagName'

function getSubAssignStatus(
  sub: Subscription,
  serverId: string,
  serverAccounts: Account[],
): 'assigned' | 'existing' | 'new' {
  if (sub.accounts?.some(a => a.server_id === serverId)) return 'assigned'
  if (serverAccounts.some(a => a.email === sub.name)) return 'existing'
  return 'new'
}

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
  port: 443,
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
  const [syncingXray, setSyncingXray] = useState<string | null>(null)
  const [accountModalTab, setAccountModalTab] = useState<string>('accounts')
  const [assignSubs, setAssignSubs] = useState<Subscription[]>([])
  const [assignServerAccounts, setAssignServerAccounts] = useState<Account[]>([])
  const [assignLoading, setAssignLoading] = useState(false)
  const [assignSelected, setAssignSelected] = useState<string[]>([])
  const [assignSubmitting, setAssignSubmitting] = useState(false)
  const [deleteSelected, setDeleteSelected] = useState<string[]>([])
  const [deleteSubmitting, setDeleteSubmitting] = useState(false)
  const [latencies, setLatencies] = useState<Map<string, number>>(new Map())
  const [pinging, setPinging] = useState(false)
  const isPinging = useRef(false)

  const TIME_RANGES = [
    { label: '1小时', value: '1h' },
    { label: '3小时', value: '3h' },
    { label: '6小时', value: '6h' },
    { label: '1天', value: '1d' },
    { label: '7天', value: '7d' },
    { label: '30天', value: '30d' },
  ]

  // Preload plotly when page loads
  useEffect(() => {
    plotlyPromise.then(() => {})
  }, [])

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
      message.error(`加载服务器列表失败：${(e as any)?.message || '请检查网络连接'}`)
      setServers([])
    } finally {
      setLoading(false)
    }
  }

  const pingAll = async () => {
    if (isPinging.current) return
    isPinging.current = true
    setPinging(true)
    try {
      const data = await serverAPI.pingAll()
      setLatencies(new Map(data.results.map((r: { server_id: string; latency_ms: number }) => [r.server_id, r.latency_ms])))
    } catch {
      // 静默失败
    } finally {
      isPinging.current = false
      setPinging(false)
    }
  }

  useEffect(() => {
    loadServers()
  }, [timeRange])

  useEffect(() => {
    pingAll()
    const timer = setInterval(pingAll, 60_000)
    return () => clearInterval(timer)
  }, [])

  // 自动滚动到最新输出
  useEffect(() => {
    if (installOutputRef.current) {
      installOutputRef.current.scrollTop = installOutputRef.current.scrollHeight
    }
  }, [installOutput])

  // 安装结束后判断最终结果（installing: true→false 时触发）
  const prevInstalling = useRef(false)
  useEffect(() => {
    if (prevInstalling.current && !installing) {
      // 以 [ERROR] 或 连接失败 为真正的错误标志，忽略 Agent 等非核心步骤的"失败"字样
      if (installOutput.includes('[ERROR]') || installOutput.includes('❌')) {
        message.error('安装失败，请查看输出')
      } else if (installOutput.includes('[OK]') || installOutput.includes('Reality配置已保存') || installOutput.includes('✓ 安装完成')) {
        message.success('安装完成！')
      }
    }
    prevInstalling.current = installing
  }, [installing])

  const loadAssignData = async () => {
    if (!selectedServerForAccounts || assignLoading) return
    setAssignLoading(true)
    try {
      const [subs, serverAccs] = await Promise.all([
        subscriptionAPI.listFull(),
        accountAPI.listByServer(selectedServerForAccounts.id),
      ])
      setAssignSubs(subs || [])
      setAssignServerAccounts(serverAccs || [])
      setAssignSelected([])
    } catch {
      message.error('加载订阅数据失败')
    } finally {
      setAssignLoading(false)
    }
  }

  const handleAssign = async () => {
    if (!selectedServerForAccounts || assignSelected.length === 0) return
    setAssignSubmitting(true)
    let succeeded = 0, failed = 0
    try {
      for (const id of assignSelected) {
        try {
          await subscriptionAPI.addAccount(id, {
            server_id: selectedServerForAccounts.id,
            auto_create: true,
          })
          succeeded++
        } catch {
          failed++
        }
      }
      if (failed === 0) {
        message.success(`成功分配 ${succeeded} 个订阅`)
      } else {
        message.warning(`${succeeded} 个成功，${failed} 个失败`)
      }
      try {
        await serverAPI.syncXray(selectedServerForAccounts.id)
      } catch {
        message.warning('xray 同步失败，请手动点击同步按钮')
      }
      setAssignSelected([])
      await loadAssignData()
    } finally {
      setAssignSubmitting(false)
    }
  }

  const handleDeleteAssign = async () => {
    if (!selectedServerForAccounts || deleteSelected.length === 0) return
    setDeleteSubmitting(true)
    const serverId = selectedServerForAccounts.id
    try {
      const results = await Promise.allSettled(
        deleteSelected.map(subId => {
          const sub = assignSubs.find(s => s.id === subId)
          const acc = sub?.accounts?.find(a => a.server_id === serverId)
          if (!acc) return Promise.reject(new Error('account not found'))
          return Promise.all([
            subscriptionAPI.removeAccount(subId, acc.id),
            accountAPI.delete(acc.id),
          ])
        })
      )
      const succeeded = results.filter(r => r.status === 'fulfilled').length
      const failed = results.filter(r => r.status === 'rejected').length
      if (failed === 0) {
        message.success(`已删除 ${succeeded} 个关联`)
      } else {
        message.warning(`${succeeded} 个成功，${failed} 个失败`)
      }
      try {
        await serverAPI.syncXray(selectedServerForAccounts.id)
      } catch {
        message.warning('xray 同步失败，请手动点击同步按钮')
      }
      setDeleteSelected([])
      await loadAssignData()
    } finally {
      setDeleteSubmitting(false)
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
      message.error(`添加服务器失败：${(e as any)?.message || '请检查填写内容'}`)
    }
  }

  const handleDelete = async (id: string) => {
    try {
      await serverAPI.delete(id)
      message.success('删除成功')
      loadServers()
    } catch (e) {
      message.error(`删除失败：${(e as any)?.message || '请稍后重试'}`)
    }
  }

  const handleRestartXray = async (serverId: string, e: React.MouseEvent) => {
    e.stopPropagation()
    setRestartingXray(serverId)
    try {
      await serverAPI.restartXray(serverId)
      message.success('xray 已重启')
    } catch (err) {
      message.error(`xray 重启失败：${(err as any)?.message || '请稍后重试'}`)
    } finally {
      setRestartingXray(null)
    }
  }

  const handleSyncXray = async (serverId: string, e: React.MouseEvent) => {
    e.stopPropagation()
    setSyncingXray(serverId)
    try {
      await serverAPI.syncXray(serverId)
      message.success('xray 配置同步成功')
    } catch {
      message.error('xray 同步失败')
    } finally {
      setSyncingXray(null)
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

    const token = localStorage.getItem('admin_token') || ''
    fetch(`/api/servers/${selectedServer?.id}/install`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', 'Authorization': `Bearer ${token}` },
      body: JSON.stringify({
        core: installConfig.core,
        uuid: installConfig.uuid,
        protocols: installConfig.protocols,
        port: installConfig.port || 0,
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
    { title: '名称', dataIndex: 'name', render: (v: string, record: any) => <FlagName name={v} countryCode={record.country_code} /> },
    { title: 'IP', dataIndex: 'ip' },
    { title: 'SSH端口', dataIndex: 'ssh_port' },
    { title: 'SSH用户', dataIndex: 'ssh_user' },
    { title: '认证方式', dataIndex: 'ssh_key_type', render: (v: string) => v === 'password' ? '密码' : '密钥' },
    { title: '状态', dataIndex: 'status' },
    {
      key: 'latency',
      title: '延迟',
      render: (_: any, record: Server) => {
        if (!latencies.has(record.id)) return <span style={{ color: 'var(--text-muted)' }}>—</span>
        const ms = latencies.get(record.id)!
        if (ms === -1) return <Tag color="red">超时</Tag>
        if (ms <= 100) return <Tag color="green">{ms}ms</Tag>
        if (ms <= 200) return <Tag color="orange">{ms}ms</Tag>
        return <Tag color="red">{ms}ms</Tag>
      },
    },
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
          <Button size="small" type="primary" icon={<CloudUploadOutlined />} aria-label="安装 Agent" title="安装 Agent" onClick={(e) => { e.stopPropagation(); handleInstallClick(record); }} />
          <Button size="small" icon={<EditOutlined />} aria-label="编辑服务器" title="编辑" onClick={(e) => { e.stopPropagation(); handleEditClick(record); }} />
          <Button size="small" icon={<TeamOutlined />} aria-label="账号管理" title="账号管理" onClick={(e) => { e.stopPropagation(); handleOpenAccountModal(record); }} />
          <Button
            size="small"
            icon={<ReloadOutlined />}
            aria-label="重启 xray"
            title="重启 xray"
            loading={restartingXray === record.id}
            onClick={(e) => handleRestartXray(record.id, e)}
          />
          <Button
            size="small"
            icon={<UploadOutlined />}
            aria-label="同步 xray 配置"
            title="同步 xray 配置"
            loading={syncingXray === record.id}
            onClick={(e) => handleSyncXray(record.id, e)}
          />
          <Popconfirm title="确定删除?" onConfirm={() => handleDelete(record.id)}>
            <Button size="small" danger icon={<DeleteOutlined />} aria-label="删除服务器" title="删除" onClick={(e) => e.stopPropagation()} />
          </Popconfirm>
        </Space>
      ),
    },
  ]

  return (
    <div className="animate-in">
      {/* Page Header */}
      <div className="page-header-row">
        <div>
          <h1>服务器管理</h1>
          <p>管理您的 V2ray 服务器和账号</p>
        </div>
        <Space style={{ marginTop: 6 }}>
          <Button
            loading={pinging}
            onClick={pingAll}
            icon={<ThunderboltOutlined />}
          >
            测速
          </Button>
          <Button type="primary" onClick={() => setModalVisible(true)}>+ 添加服务器</Button>
        </Space>
      </div>

      {/* Stats Grid */}
      <div className="stats-grid">
        <div className="stat-card animate-in animate-delay-1">
          <div className="stat-icon rose"><HddOutlined style={{ fontSize: 22, color: 'var(--morandi-dusty-rose)' }} /></div>
          <div className="stat-content">
            <h3>{servers.length}</h3>
            <p>服务器总数</p>
          </div>
        </div>
        <div className="stat-card animate-in animate-delay-2">
          <div className="stat-icon sage"><CheckCircleOutlined style={{ fontSize: 22, color: 'var(--morandi-sage)' }} /></div>
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
                        color="#9DB4C0"
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
                        color="#B4A7C7"
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
                        color="#A8B5A0"
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
                        color="#C4836A"
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
            <Input autoComplete="off" />
          </Form.Item>
          <Form.Item name="ip" label="IP地址" rules={[{ required: true }]}>
            <Input autoComplete="off" />
          </Form.Item>
          <Form.Item name="ssh_port" label="SSH端口" initialValue={22}>
            <InputNumber min={1} max={65535} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item name="ssh_user" label="SSH用户" initialValue="root">
            <Input autoComplete="off" />
          </Form.Item>
          <Form.Item name="ssh_key_type" label="认证方式" initialValue="key">
            <Select onChange={(v) => setSshKeyType(v)}>
              <Select.Option value="key">SSH密钥</Select.Option>
              <Select.Option value="password">密码</Select.Option>
            </Select>
          </Form.Item>
          {sshKeyType === 'key' ? (
            <Form.Item name="ssh_key" label="SSH私钥">
              <Input.TextArea rows={4} placeholder="粘贴SSH私钥内容" autoComplete="off" />
            </Form.Item>
          ) : (
            <Form.Item name="ssh_password" label="SSH密码">
              <Input.Password autoComplete="new-password" />
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
              autoComplete="off"
            />
          </Form.Item>

          <Form.Item label="端口" extra="留空或清除则随机分配">
            <InputNumber
              min={1}
              max={65535}
              value={installConfig.port || undefined}
              onChange={(v) => setInstallConfig({ ...installConfig, port: v || 0 })}
              placeholder="随机"
              style={{ width: '100%' }}
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
        {installing && <div style={{ color: 'var(--text-secondary)', marginBottom: 8 }}>正在安装，请稍候...（这可能需要10-30秒）</div>}
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
        title={`账号管理 · ${selectedServerForAccounts?.name || ''}`}
        open={accountModalVisible}
        onCancel={() => {
          setAccountModalVisible(false)
          setAccountModalTab('accounts')
          setAssignSubs([])
          setAssignServerAccounts([])
          setAssignSelected([])
          setDeleteSelected([])
        }}
        width={750}
        footer={null}
      >
        <Tabs
          activeKey={accountModalTab}
          onChange={(key) => {
            setAccountModalTab(key)
            if (key === 'assign') loadAssignData()
          }}
          items={[
            {
              key: 'accounts',
              label: '账号',
              children: (
                <>
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
                </>
              ),
            },
            {
              key: 'assign',
              label: '关联订阅',
              children: (
                <Spin spinning={assignLoading}>
                  {(() => {
                    const serverId = selectedServerForAccounts?.id || ''
                    const assignedSubs: Subscription[] = []
                    const selectableSubs: Subscription[] = []
                    const statusCache = new Map<string, 'assigned' | 'existing' | 'new'>()
                    for (const sub of assignSubs) {
                      const status = getSubAssignStatus(sub, serverId, assignServerAccounts)
                      statusCache.set(sub.id, status)
                      if (status === 'assigned') assignedSubs.push(sub)
                      else selectableSubs.push(sub)
                    }
                    const allDeleteSelected = assignedSubs.length > 0 &&
                      assignedSubs.every(s => deleteSelected.includes(s.id))
                    const someDeleteSelected = assignedSubs.some(s => deleteSelected.includes(s.id)) && !allDeleteSelected
                    const allAssignSelected = selectableSubs.length > 0 &&
                      selectableSubs.every(s => assignSelected.includes(s.id))
                    const someAssignSelected = selectableSubs.some(s => assignSelected.includes(s.id)) && !allAssignSelected
                    return (
                      <div style={{ display: 'flex', gap: 0, minHeight: 200 }}>
                        {/* Left: assigned */}
                        <div style={{ flex: 1, paddingRight: 12, borderRight: '1px solid #f0f0f0', display: 'flex', flexDirection: 'column' }}>
                          <div style={{ marginBottom: 8, paddingBottom: 8, borderBottom: '1px solid #f0f0f0' }}>
                            <Checkbox
                              checked={allDeleteSelected}
                              indeterminate={someDeleteSelected}
                              onChange={e => setDeleteSelected(
                                e.target.checked ? assignedSubs.map(s => s.id) : []
                              )}
                            >
                              已关联（{assignedSubs.length} 个）
                            </Checkbox>
                          </div>
                          <div style={{ flex: 1, maxHeight: 280, overflowY: 'auto' }}>
                            {assignedSubs.map(sub => (
                              <div
                                key={sub.id}
                                style={{
                                  display: 'flex',
                                  alignItems: 'center',
                                  justifyContent: 'space-between',
                                  padding: '6px 0',
                                  borderBottom: '1px solid #f0f0f0',
                                }}
                              >
                                <Checkbox
                                  checked={deleteSelected.includes(sub.id)}
                                  onChange={e => setDeleteSelected(
                                    e.target.checked
                                      ? [...deleteSelected, sub.id]
                                      : deleteSelected.filter(id => id !== sub.id)
                                  )}
                                >
                                  <span style={{ fontWeight: 500 }}>{sub.name}</span>
                                  {sub.remark && (
                                    <span style={{ color: 'var(--text-secondary)', marginLeft: 6, fontSize: 12 }}>
                                      {sub.remark}
                                    </span>
                                  )}
                                </Checkbox>
                                <Tag>已分配</Tag>
                              </div>
                            ))}
                            {assignedSubs.length === 0 && !assignLoading && (
                              <div style={{ textAlign: 'center', color: 'var(--text-secondary)', padding: '24px 0', fontSize: 13 }}>
                                暂无已关联订阅
                              </div>
                            )}
                          </div>
                          <div style={{ marginTop: 12, textAlign: 'left' }}>
                            <Popconfirm
                              title={`确定解除关联并删除这 ${deleteSelected.length} 个账号吗？此操作不可恢复。`}
                              onConfirm={handleDeleteAssign}
                              disabled={deleteSelected.length === 0}
                              okText="确认删除"
                              cancelText="取消"
                              okButtonProps={{ danger: true }}
                            >
                              <Button
                                danger
                                loading={deleteSubmitting}
                                disabled={deleteSelected.length === 0}
                              >
                                删除关联（已选 {deleteSelected.length} 个）
                              </Button>
                            </Popconfirm>
                          </div>
                        </div>

                        {/* Right: assignable */}
                        <div style={{ flex: 1, paddingLeft: 12, display: 'flex', flexDirection: 'column' }}>
                          <div style={{ marginBottom: 8, paddingBottom: 8, borderBottom: '1px solid #f0f0f0' }}>
                            <Checkbox
                              checked={allAssignSelected}
                              indeterminate={someAssignSelected}
                              onChange={e => setAssignSelected(
                                e.target.checked ? selectableSubs.map(s => s.id) : []
                              )}
                            >
                              可分配（{selectableSubs.length} 个）
                            </Checkbox>
                          </div>
                          <div style={{ flex: 1, maxHeight: 280, overflowY: 'auto' }}>
                            {selectableSubs.map(sub => {
                              const status = statusCache.get(sub.id)!
                              const existingAcc = status === 'existing'
                                ? assignServerAccounts.find(a => a.email === sub.name)
                                : undefined
                              return (
                                <div
                                  key={sub.id}
                                  style={{
                                    display: 'flex',
                                    alignItems: 'center',
                                    justifyContent: 'space-between',
                                    padding: '6px 0',
                                    borderBottom: '1px solid #f0f0f0',
                                  }}
                                >
                                  <Checkbox
                                    checked={assignSelected.includes(sub.id)}
                                    onChange={e => setAssignSelected(
                                      e.target.checked
                                        ? [...assignSelected, sub.id]
                                        : assignSelected.filter(id => id !== sub.id)
                                    )}
                                  >
                                    <span style={{ fontWeight: 500 }}>{sub.name}</span>
                                    {sub.remark && (
                                      <span style={{ color: 'var(--text-secondary)', marginLeft: 6, fontSize: 12 }}>
                                        {sub.remark}
                                      </span>
                                    )}
                                  </Checkbox>
                                  {status === 'existing' && (
                                    <Tag style={{ background: 'rgba(168,181,160,0.15)', color: '#4d6e48', border: '1px solid rgba(168,181,160,0.4)', borderRadius: 6 }}>已有账号 · {existingAcc?.email}</Tag>
                                  )}
                                  {status === 'new' && (
                                    <Tag color="default">将新建 · {sub.name}</Tag>
                                  )}
                                </div>
                              )
                            })}
                            {selectableSubs.length === 0 && !assignLoading && (
                              <div style={{ textAlign: 'center', color: 'var(--text-secondary)', padding: '24px 0', fontSize: 13 }}>
                                全部已关联
                              </div>
                            )}
                          </div>
                          <div style={{ marginTop: 12, textAlign: 'right' }}>
                            <Button
                              type="primary"
                              loading={assignSubmitting}
                              disabled={assignSelected.length === 0}
                              onClick={handleAssign}
                            >
                              确认分配（已选 {assignSelected.length} 个）
                            </Button>
                          </div>
                        </div>
                      </div>
                    )
                  })()}
                </Spin>
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
            <Input autoComplete="off" />
          </Form.Item>
          <Form.Item name="ip" label="IP地址" rules={[{ required: true }]}>
            <Input autoComplete="off" />
          </Form.Item>
          <Form.Item name="ssh_port" label="SSH端口" rules={[{ required: true }]}>
            <InputNumber min={1} max={65535} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item name="ssh_user" label="SSH用户" rules={[{ required: true }]}>
            <Input autoComplete="off" />
          </Form.Item>
          <Form.Item name="ssh_key_type" label="认证方式" rules={[{ required: true }]}>
            <Select onChange={(v) => setEditSshKeyType(v)}>
              <Select.Option value="key">SSH密钥</Select.Option>
              <Select.Option value="password">密码</Select.Option>
            </Select>
          </Form.Item>
          {editSshKeyType === 'key' ? (
            <Form.Item name="ssh_key" label="SSH私钥">
              <Input.TextArea rows={4} placeholder="粘贴SSH私钥内容" autoComplete="off" />
            </Form.Item>
          ) : (
            <Form.Item name="ssh_password" label="SSH密码">
              <Input.Password autoComplete="new-password" />
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
          border-right: 1px solid var(--border-color);
        }
        .metric-cell:last-child { border-right: none; }
        .metric-label {
          font-size: 12px;
          color: var(--text-secondary);
          margin-bottom: 4px;
        }
        .metric-value {
          font-size: 18px;
          font-weight: 500;
          font-family: 'JetBrains Mono', monospace;
          color: var(--text-primary);
          margin-bottom: 8px;
        }
        .chart-area {
          height: 110px;
        }
        .empty-monitor {
          padding: 40px;
          text-align: center;
          color: var(--text-muted);
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
      const bytesPerSec = y / 60
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
        font: { color: '#9E9A93', family: 'Noto Sans SC, sans-serif', size: 9 },
        xaxis: {
          showgrid: false,
          gridcolor: '#EDE8E0',
          linecolor: '#E5E0D8',
          tickcolor: '#E5E0D8',
          ticks: 'outside',
          tickfont: { size: 9 },
          showline: true,
          zeroline: false,
          tickformat: '%H:%M',
        },
        yaxis: {
          showgrid: true,
          gridcolor: '#F0EDE8',
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
          font: { size: 11, color: '#6B6760' },
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
    return <div style={{ height: '100%', display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#9E9A93', fontSize: 11 }}>暂无数据</div>
  }

  return <div ref={containerRef} style={{ width: '100%', height: '100%' }} />
}

// Shared promise to avoid multiple imports
const plotlyPromise = import('plotly.js-dist-min')
import { useState, useEffect, useRef, useCallback } from 'react'
import { Table, Button, Space, Modal, Form, Input, Select, message, Card, Tag, Checkbox, Tabs, Popconfirm } from 'antd'
import { CopyOutlined, QrcodeOutlined, HolderOutlined } from '@ant-design/icons'
import { DndContext, closestCenter, DragEndEvent } from '@dnd-kit/core'
import { SortableContext, useSortable, verticalListSortingStrategy } from '@dnd-kit/sortable'
import { CSS } from '@dnd-kit/utilities'
import QRCode from 'qrcode'
import { subscriptionAPI, serverAPI, accountAPI, Subscription, Server, Account, AccountWithServer, AccountMapping } from '../../services/api'

interface SubscriptionWithAccounts extends Subscription {
  accounts?: AccountWithServer[]
}

export default function SubscriptionList() {
  const [subscriptions, setSubscriptions] = useState<SubscriptionWithAccounts[]>([])
  const [servers, setServers] = useState<Server[]>([])
  const [accounts, setAccounts] = useState<Record<string, Account[]>>({})
  const [loading, setLoading] = useState(false)
  const [modalVisible, setModalVisible] = useState(false)
  const [linkModalVisible, setLinkModalVisible] = useState(false)
  const [currentLink, setCurrentLink] = useState('')
  const [currentEncoded, setCurrentEncoded] = useState('')
  const [currentSubInfo, setCurrentSubInfo] = useState<SubscriptionWithAccounts | null>(null)
  const [form] = Form.useForm()
  const [selectedMappings, setSelectedMappings] = useState<AccountMapping[]>([])
  const [activeTab, setActiveTab] = useState('uri')
  const [manageModalVisible, setManageModalVisible] = useState(false)
  const [managedAccounts, setManagedAccounts] = useState<AccountWithServer[]>([])
  const [manageSubscriptionId, setManageSubscriptionId] = useState<string>('')
  const [addAccountModalVisible, setAddAccountModalVisible] = useState(false)

  useEffect(() => {
    loadData()
  }, [])

  const loadData = useCallback(async () => {
    setLoading(true)
    try {
      const [subs, srvs] = await Promise.all([
        subscriptionAPI.listFull(),
        serverAPI.list(),
      ])
      setSubscriptions(subs || [])

      const accountResults = await Promise.all(
        srvs.map(srv => accountAPI.listByServer(srv.id).then(accs => ({ srv, accs })))
      )
      const accountMap: Record<string, Account[]> = {}
      for (const { srv, accs } of accountResults) {
        accountMap[srv.id] = accs || []
      }
      setAccounts(accountMap)
      setServers(srvs || [])
    } catch (e) {
      message.error('加载失败')
      setSubscriptions([])
      setServers([])
    } finally {
      setLoading(false)
    }
  }, [])

  const handleAdd = async (values: any) => {
    if (selectedMappings.length === 0) {
      message.error('请至少选择一个账号')
      return
    }
    try {
      await subscriptionAPI.create({
        name: values.name,
        traffic_limit: values.traffic_limit ? values.traffic_limit * 1024 * 1024 * 1024 : 0,
        account_mappings: selectedMappings,
      })
      message.success('添加成功')
      setModalVisible(false)
      form.resetFields()
      setSelectedMappings([])
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
      const sub = subscriptions.find(s => s.id === id)
      if (sub) {
        setCurrentSubInfo(sub)
      }
      const { link, encoded } = await subscriptionAPI.getLink(id)
      setCurrentLink(link)
      setCurrentEncoded(encoded)
      setLinkModalVisible(true)
      setActiveTab('uri')
    } catch (e) {
      message.error('获取链接失败')
    }
  }

  const copyToClipboard = (text: string) => {
    if (navigator.clipboard && navigator.clipboard.writeText) {
      navigator.clipboard.writeText(text).then(() => {
        message.success('已复制到剪贴板')
      }).catch(() => {
        fallbackCopy(text)
      })
    } else {
      fallbackCopy(text)
    }
  }

  const fallbackCopy = (text: string) => {
    const textarea = document.createElement('textarea')
    textarea.value = text
    textarea.style.position = 'fixed'
    textarea.style.opacity = '0'
    document.body.appendChild(textarea)
    textarea.select()
    try {
      document.execCommand('copy')
      message.success('已复制到剪贴板')
    } catch (err) {
      message.error('复制失败')
    }
    document.body.removeChild(textarea)
  }

  const handleServerSelect = (serverId: string, checked: boolean) => {
    if (checked) {
      const serverAccounts = accounts[serverId] || []
      if (serverAccounts.length === 0) {
        message.warn('该服务器暂无账号，请先创建')
        return
      }
      if (selectedMappings.some(m => m.server_id === serverId)) {
        message.warn('该服务器已选择账号')
        return
      }
      setSelectedMappings([...selectedMappings, { server_id: serverId, account_id: serverAccounts[0].id }])
    } else {
      setSelectedMappings(selectedMappings.filter(m => m.server_id !== serverId))
    }
  }

  const handleAccountChange = (serverId: string, accountId: string) => {
    setSelectedMappings(selectedMappings.map(m =>
      m.server_id === serverId ? { ...m, account_id: accountId } : m
    ))
  }

  const openModal = () => {
    setSelectedMappings([])
    form.resetFields()
    setModalVisible(true)
  }

  const openManageModal = async (record: SubscriptionWithAccounts) => {
    setManageSubscriptionId(record.id)
    try {
      const accounts = await subscriptionAPI.getAccounts(record.id)
      setManagedAccounts(accounts || [])
      setManageModalVisible(true)
    } catch (e) {
      message.error('加载账号失败')
    }
  }

  const onDragEnd = async (result: DragEndEvent) => {
    if (!result.destination) return

    const items = Array.from(managedAccounts)
    const [reorderedItem] = items.splice(result.source.index, 1)
    items.splice(result.destination.index, 0, reorderedItem)

    const previousAccounts = managedAccounts
    setManagedAccounts(items)

    const newOrder = items.map((acc, idx) => ({ id: acc.id, sort_order: idx }))
    try {
      await subscriptionAPI.updateAccountOrder(manageSubscriptionId, newOrder)
    } catch (e) {
      message.error('保存顺序失败')
      setManagedAccounts(previousAccounts)
    }
  }

  const handleRemoveAccount = async (accountId: string) => {
    try {
      await subscriptionAPI.removeAccount(manageSubscriptionId, accountId)
      setManagedAccounts(managedAccounts.filter(acc => acc.id !== accountId))
      message.success('移除成功')
    } catch (e) {
      message.error('移除失败')
    }
  }

  const handleAddAccount = async (serverId: string, accountId: string) => {
    try {
      await subscriptionAPI.addAccount(manageSubscriptionId, { server_id: serverId, account_id: accountId })
      message.success('添加成功')
      const accounts = await subscriptionAPI.getAccounts(manageSubscriptionId)
      setManagedAccounts(accounts || [])
      setAddAccountModalVisible(false)
    } catch (e) {
      message.error('添加失败')
    }
  }

  const columns = [
    { title: '名称', dataIndex: 'name' },
    { title: 'UUID', dataIndex: 'uuid', render: (v: string) => v ? v.substring(0, 8) + '...' : '-' },
    {
      title: '服务器/账号',
      render: (_: any, record: SubscriptionWithAccounts) => {
        if (!record.accounts || record.accounts.length === 0) {
          return <Tag color="default">暂无可用节点，请联系管理员</Tag>
        }
        return (
          <Space direction="vertical" size={2}>
            {record.accounts.map(acc => (
              <Tag key={acc.id} color="blue">{acc.server_name} / {acc.email}</Tag>
            ))}
          </Space>
        )
      }
    },
    { title: '流量限制', dataIndex: 'traffic_limit', render: (v: number) => v ? `${(v/1024**3).toFixed(1)} GB` : '无限' },
    { title: '已用流量', dataIndex: 'traffic_used', render: (v: number) => `${(v/1024**3).toFixed(1)} GB` },
    { title: '状态', dataIndex: 'enable', render: (v: boolean) => v ? <Tag color="green">启用</Tag> : <Tag color="gray">禁用</Tag> },
    {
      title: '操作',
      render: (_: any, record: SubscriptionWithAccounts) => (
        <Space>
          <Button size="small" type="primary" onClick={() => handleGetLink(record.id)}>订阅链接</Button>
          <Button size="small" onClick={() => openManageModal(record)}>管理账号</Button>
          <Popconfirm
            title="确认删除"
            description="确定要删除这个订阅吗？"
            onConfirm={() => handleDelete(record.id)}
            okText="确认"
            cancelText="取消"
          >
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
        <h1>订阅管理</h1>
        <p>创建和管理用户的订阅链接，支持多服务器账号</p>
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
        <Button type="primary" onClick={openModal}>+ 添加订阅</Button>
      </Card>

      {/* Subscription Table */}
      <Card className="morandi-card">
        <Table columns={columns} dataSource={subscriptions} rowKey="id" loading={loading} pagination={{ pageSize: 10 }} />
      </Card>

      {/* Add Modal */}
      <Modal
        title="添加订阅"
        open={modalVisible}
        onCancel={() => setModalVisible(false)}
        footer={null}
        width={700}
      >
        <Form form={form} onFinish={handleAdd} layout="vertical" style={{ marginTop: 20 }}>
          <Form.Item name="name" label="名称" rules={[{ required: true }]}>
            <Input placeholder="例如: VIP用户" />
          </Form.Item>
          <Form.Item name="traffic_limit" label="流量限制(GB)">
            <Input type="number" placeholder="留空表示无限流量" />
          </Form.Item>

          <div style={{ marginBottom: 16 }}>
            <h4>选择服务器和账号（每个服务器只能选一个账号）</h4>
            <div style={{ maxHeight: 300, overflowY: 'auto' }}>
              {servers.map(server => {
                const serverAccounts = accounts[server.id] || []
                const isSelected = selectedMappings.some(m => m.server_id === server.id)
                const selectedMapping = selectedMappings.find(m => m.server_id === server.id)

                return (
                  <div key={server.id} style={{
                    border: '1px solid #d9d9d9',
                    borderRadius: 4,
                    padding: 12,
                    marginBottom: 8,
                    backgroundColor: isSelected ? '#f0f5ff' : 'white'
                  }}>
                    <Checkbox
                      checked={isSelected}
                      onChange={(e) => handleServerSelect(server.id, e.target.checked)}
                      style={{ marginBottom: 8 }}
                    >
                      <strong>{server.name}</strong> ({server.ip})
                    </Checkbox>

                    {isSelected && serverAccounts.length > 0 && (
                      <div style={{ marginLeft: 24 }}>
                        <Select
                          value={selectedMapping?.account_id || serverAccounts[0]?.id}
                          onChange={(v) => handleAccountChange(server.id, v)}
                          style={{ width: '100%' }}
                        >
                          {serverAccounts.map(acc => (
                            <Select.Option key={acc.id} value={acc.id}>
                              {acc.email} {acc.enabled ? '' : '(已禁用)'}
                            </Select.Option>
                          ))}
                        </Select>
                      </div>
                    )}

                    {isSelected && serverAccounts.length === 0 && (
                      <div style={{ marginLeft: 24, color: '#ff4d4f' }}>
                        该服务器暂无账号，请先创建
                      </div>
                    )}
                  </div>
                )
              })}
            </div>
          </div>

          <Button type="primary" htmlType="submit" disabled={selectedMappings.length === 0}>
            提交 ({selectedMappings.length} 个账号)
          </Button>
        </Form>
      </Modal>

      {/* Link Modal */}
      <Modal
        title="订阅链接"
        open={linkModalVisible}
        onCancel={() => setLinkModalVisible(false)}
        footer={null}
        width={600}
      >
        <Tabs activeKey={activeTab} onChange={setActiveTab} style={{ marginTop: 16 }}
          items={[
            {
              key: 'uri',
              label: '订阅 URI',
              children: <div style={{ marginBottom: 16 }}>
                <h4>订阅地址 (通用)</h4>
                <div style={{ display: 'flex', gap: 8, marginBottom: 12 }}>
                  <Input value={currentLink} readOnly style={{ flex: 1 }} />
                  <Button icon={<CopyOutlined />} onClick={() => copyToClipboard(currentLink)} />
                </div>
                <h4>Base64 编码 (ShadowRocket/Quantumult)</h4>
                <div style={{ display: 'flex', gap: 8 }}>
                  <Input value={currentEncoded} readOnly style={{ flex: 1 }} />
                  <Button icon={<CopyOutlined />} onClick={() => copyToClipboard(currentEncoded)} />
                </div>
              </div>
            },
            {
              key: 'qrcode',
              label: '二维码',
              children: <div style={{ textAlign: 'center' }}>
                {currentSubInfo?.accounts?.map((acc, idx) => (
                  <div key={acc.id} style={{ marginBottom: 24 }}>
                    <h4>{acc.server_name} - {acc.email}</h4>
                    <QRCodeDisplay
                      link={`${currentLink}&aid=${acc.id}&format=ss`}
                      label={`${acc.server_name} / ${acc.email}`}
                    />
                  </div>
                ))}
                {(!currentSubInfo?.accounts || currentSubInfo.accounts.length === 0) && (
                  <div style={{ padding: 40, color: '#999' }}>无账号信息</div>
                )}
              </div>
            },
          ]}
        />

        <div style={{ marginTop: 16, fontSize: 12, color: 'var(--text-muted)' }}>
          <p>支持格式: ?format=vless (默认) | ?format=clash_meta | ?format=singbox | ?format=ss (ShadowRocket)</p>
          <p>客户端扫码可直接导入单个账号，或使用上方订阅 URI 导入全部账号</p>
        </div>
      </Modal>

      {/* Manage Accounts Modal */}
      <Modal
        title="管理账号"
        open={manageModalVisible}
        onCancel={() => setManageModalVisible(false)}
        footer={null}
        width={600}
      >
        <div style={{ marginTop: 16 }}>
          <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'flex-end' }}>
            <Button type="primary" onClick={() => setAddAccountModalVisible(true)}>
              添加账号
            </Button>
          </div>

          {managedAccounts.length === 0 ? (
            <div style={{ padding: 40, textAlign: 'center', color: '#999' }}>
              暂无账号，请点击上方按钮添加
            </div>
          ) : (
            <DndContext collisionDetection={closestCenter} onDragEnd={onDragEnd}>
              <SortableContext items={managedAccounts.map(a => a.id)} strategy={verticalListSortingStrategy}>
                {managedAccounts.map((acc) => (
                  <SortableItem
                    key={acc.id}
                    id={acc.id}
                    account={acc}
                    onRemove={() => handleRemoveAccount(acc.id)}
                  />
                ))}
              </SortableContext>
            </DndContext>
          )}
        </div>
      </Modal>

      {/* Add Account Modal */}
      <Modal
        title="添加账号"
        open={addAccountModalVisible}
        onCancel={() => setAddAccountModalVisible(false)}
        footer={null}
        width={500}
      >
        <div style={{ marginTop: 16, maxHeight: 400, overflowY: 'auto' }}>
          {servers.map(server => {
            const serverAccounts = accounts[server.id] || []
            const linkedAccountIds = managedAccounts.map(a => a.id)
            // 同一服务器只能添加一个账号：如果该服务器已有账号在订阅中，就不再显示任何账号
            const hasLinkedAccountFromServer = managedAccounts.some(a => a.server_id === server.id)
            const availableAccounts = hasLinkedAccountFromServer ? [] : serverAccounts.filter(acc => !linkedAccountIds.includes(acc.id))

            return (
              <div key={server.id} style={{
                border: '1px solid #d9d9d9',
                borderRadius: 4,
                padding: 12,
                marginBottom: 8,
              }}>
                <strong>{server.name}</strong> ({server.ip})
                {availableAccounts.length === 0 ? (
                  <div style={{ marginTop: 8, color: '#999' }}>
                    {hasLinkedAccountFromServer ? '已添加此服务器的账号' : '无可用账号'}
                  </div>
                ) : (
                  <div style={{ marginTop: 8 }}>
                    {availableAccounts.map(acc => (
                      <div key={acc.id} style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 4 }}>
                        <span>{acc.email} {acc.enabled ? '' : '(已禁用)'}</span>
                        <Button size="small" type="primary" onClick={() => handleAddAccount(server.id, acc.id)}>
                          添加
                        </Button>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            )
          })}
        </div>
      </Modal>
    </div>
  )
}

// Sortable item component
function SortableItem({ id, account, onRemove }: { id: string; account: AccountWithServer; onRemove: () => void }) {
  const {
    attributes,
    listeners,
    setNodeRef,
    transform,
    transition,
    isDragging,
  } = useSortable({ id })

  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
    opacity: isDragging ? 0.5 : 1,
  }

  return (
    <div
      ref={setNodeRef}
      style={{
        ...style,
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'space-between',
        padding: '12px',
        marginBottom: 8,
        border: '1px solid #d9d9d9',
        borderRadius: 4,
        backgroundColor: 'white',
      }}
    >
      <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
        <div {...attributes} {...listeners} style={{ cursor: 'grab', padding: '0 8px' }}>
          <HolderOutlined />
        </div>
        <div>
          <div style={{ fontWeight: 'bold' }}>{account.server_name} / {account.email}</div>
          <div style={{ fontSize: 12, color: '#999' }}>ID: {account.id.substring(0, 8)}...</div>
        </div>
      </div>
      <Button size="small" danger onClick={onRemove}>移除</Button>
    </div>
  )
}

// QRCode display component
function QRCodeDisplay({ link, label }: { link: string; label: string }) {
  const [qrDataUrl, setQrDataUrl] = useState<string>('')
  const canvasRef = useRef<HTMLCanvasElement>(null)

  useEffect(() => {
    const generateQR = async () => {
      try {
        const canvas = canvasRef.current
        if (canvas) {
          await QRCode.toCanvas(canvas, link, {
            width: 200,
            margin: 2,
            color: {
              dark: '#000000',
              light: '#ffffff',
            },
          })
          setQrDataUrl(canvas.toDataURL())
        }
      } catch (err) {
        console.error('QR generation failed:', err)
      }
    }
    generateQR()
  }, [link])

  return (
    <div style={{ textAlign: 'center' }}>
      <canvas ref={canvasRef} style={{ display: 'none' }} />
      {qrDataUrl && (
        <div>
          <img
            src={qrDataUrl}
            alt="QR Code"
            style={{ width: 200, height: 200, border: '1px solid #d9d9d9', borderRadius: 8 }}
          />
          <div style={{ marginTop: 8, fontSize: 12, color: '#666' }}>{label}</div>
        </div>
      )}
    </div>
  )
}
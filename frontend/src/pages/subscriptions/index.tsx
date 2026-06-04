import { useState, useEffect, useRef, useCallback, lazy, Suspense } from 'react'
import { Table, Button, Space, Modal, Form, Input, Select, Switch, message, Card, Tag, Checkbox, Divider, Progress } from 'antd'
import { CopyOutlined, QrcodeOutlined, HolderOutlined, LinkOutlined, EditOutlined, DeleteOutlined } from '@ant-design/icons'
import { DndContext, closestCenter, DragEndEvent } from '@dnd-kit/core'
import { SortableContext, useSortable, verticalListSortingStrategy } from '@dnd-kit/sortable'
import { CSS } from '@dnd-kit/utilities'
import QRCode from 'qrcode'
import { subscriptionAPI, serverAPI, accountAPI, Subscription, Server, Account, AccountWithServer, AccountMapping, AccountTrafficSeries } from '../../services/api'
import { formatBytes } from '../../utils/format'
import { FlagName } from '../../components/FlagName'

interface SubscriptionWithAccounts extends Subscription {
  accounts?: AccountWithServer[]
}

const AUTO_CREATE = '__auto_create__'

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
  const [manageModalVisible, setManageModalVisible] = useState(false)
  const [managedAccounts, setManagedAccounts] = useState<AccountWithServer[]>([])
  const [manageSubscriptionId, setManageSubscriptionId] = useState<string>('')
  const [manageSubscriptionName, setManageSubscriptionName] = useState<string>('')
  const [editEnable, setEditEnable] = useState(true)
  const [editRemark, setEditRemark] = useState('')
  const [savingInfo, setSavingInfo] = useState(false)
  const [addAccountModalVisible, setAddAccountModalVisible] = useState(false)
  const [deleteModalVisible, setDeleteModalVisible] = useState(false)
  const [pendingDeleteSub, setPendingDeleteSub] = useState<SubscriptionWithAccounts | null>(null)
  const [orphanedAccounts, setOrphanedAccounts] = useState<AccountWithServer[]>([])
  const [alsoDeleteAccounts, setAlsoDeleteAccounts] = useState(true)
  const [deleting, setDeleting] = useState(false)

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
      setSubscriptions((subs || []).sort((a, b) =>
        new Date(b.created_at).getTime() - new Date(a.created_at).getTime()
      ))

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
      message.error('请至少选择一个服务器')
      return
    }
    try {
      // 为标记为 auto-create 的服务器自动创建账号（使用订阅名称作为账号名）
      const resolvedMappings = await Promise.all(
        selectedMappings.map(async (m) => {
          if (m.account_id !== AUTO_CREATE) return m
          const newAcc = await accountAPI.create(m.server_id, {
            email: values.name,
            protocols: ['vless_reality_vision'],
          })
          return { server_id: m.server_id, account_id: newAcc.id }
        })
      )
      await subscriptionAPI.create({
        name: values.name,
        remark: values.remark || '',
        traffic_limit: values.traffic_limit ? values.traffic_limit * 1024 * 1024 * 1024 : 0,
        account_mappings: resolvedMappings,
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

  const triggerDelete = (record: SubscriptionWithAccounts) => {
    const subAccounts = record.accounts || []
    if (subAccounts.length === 0) {
      // 无关联账号，直接普通确认
      setPendingDeleteSub(record)
      setOrphanedAccounts([])
      setAlsoDeleteAccounts(false)
      setDeleteModalVisible(true)
      return
    }
    // 找出只属于当前订阅、没有被其他订阅引用的账号
    const otherSubAccountIds = new Set(
      subscriptions
        .filter(s => s.id !== record.id)
        .flatMap(s => (s.accounts || []).map(a => a.id))
    )
    const orphans = subAccounts.filter(a => !otherSubAccountIds.has(a.id))
    setPendingDeleteSub(record)
    setOrphanedAccounts(orphans)
    setAlsoDeleteAccounts(orphans.length > 0)
    setDeleteModalVisible(true)
  }

  const handleDelete = async () => {
    if (!pendingDeleteSub) return
    setDeleting(true)
    try {
      if (alsoDeleteAccounts && orphanedAccounts.length > 0) {
        await Promise.all(orphanedAccounts.map(a => accountAPI.delete(a.id)))
      }
      await subscriptionAPI.delete(pendingDeleteSub.id)
      message.success('删除成功')
      setDeleteModalVisible(false)
      setPendingDeleteSub(null)
      setOrphanedAccounts([])
      loadData()
    } catch (e) {
      message.error('删除失败')
    } finally {
      setDeleting(false)
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
      if (selectedMappings.some(m => m.server_id === serverId)) return
      // 默认自动创建账号
      const accountId = AUTO_CREATE
      setSelectedMappings([...selectedMappings, { server_id: serverId, account_id: accountId }])
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
    setManageSubscriptionName(record.name)
    setEditEnable(record.enable)
    setEditRemark(record.remark || '')
    try {
      const accounts = await subscriptionAPI.getAccounts(record.id)
      setManagedAccounts(accounts || [])
      setManageModalVisible(true)
    } catch (e) {
      message.error('加载账号失败')
    }
  }

  const handleSaveInfo = async () => {
    setSavingInfo(true)
    try {
      await subscriptionAPI.update(manageSubscriptionId, { enable: editEnable, remark: editRemark })
      message.success('保存成功')
      loadData()
    } catch {
      message.error('保存失败')
    } finally {
      setSavingInfo(false)
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
      loadData()
    } catch (e) {
      message.error('移除失败')
    }
  }

  const handleAddAccount = async (serverId: string, accountId: string, autoCreate?: boolean) => {
    try {
      await subscriptionAPI.addAccount(manageSubscriptionId, {
        server_id: serverId,
        account_id: autoCreate ? undefined : accountId,
        auto_create: autoCreate,
      })
      message.success('添加成功')
      const updatedAccounts = await subscriptionAPI.getAccounts(manageSubscriptionId)
      setManagedAccounts(updatedAccounts || [])
      setAddAccountModalVisible(false)
      loadData()
    } catch (e) {
      message.error('添加失败')
    }
  }

  const columns = [
    {
      title: '名称',
      dataIndex: 'name',
      render: (name: string, record: SubscriptionWithAccounts) => (
        <div>
          <div style={{ fontWeight: 500 }}>{name}</div>
          {record.remark && <div style={{ fontSize: 12, color: 'var(--text-muted)' }}>{record.remark}</div>}
        </div>
      ),
    },
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
              <Tag key={acc.id} color="blue"><FlagName name={acc.server_name} /> / {acc.email}</Tag>
            ))}
          </Space>
        )
      }
    },
    {
      title: '流量使用',
      width: 200,
      sorter: (a: SubscriptionWithAccounts, b: SubscriptionWithAccounts) =>
        (a.traffic_used || 0) - (b.traffic_used || 0),
      render: (_: any, record: SubscriptionWithAccounts) => {
        const used = record.traffic_used || 0
        const limit = record.traffic_limit || 0
        const pct = limit > 0 ? Math.min(100, (used / limit) * 100) : 0
        return (
          <div style={{ minWidth: 140 }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: 12, marginBottom: 4 }}>
              <span>{formatBytes(used)}</span>
              <span style={{ color: '#999' }}>{limit > 0 ? formatBytes(limit) : '无限'}</span>
            </div>
            {limit > 0 && (
              <Progress
                percent={Math.round(pct)}
                size="small"
                strokeColor={pct > 90 ? '#ff4d4f' : pct > 75 ? '#faad14' : '#52c41a'}
                showInfo={false}
              />
            )}
          </div>
        )
      }
    },
    { title: '状态', dataIndex: 'enable', render: (v: boolean) => v ? <Tag color="green">启用</Tag> : <Tag color="gray">禁用</Tag> },
    {
      title: '操作',
      render: (_: any, record: SubscriptionWithAccounts) => (
        <Space>
          <Button size="small" type="primary" icon={<LinkOutlined />} title="订阅链接" onClick={() => handleGetLink(record.id)} />
          <Button
            size="small"
            icon={<CopyOutlined />}
            title="复制订阅链接"
            onClick={async () => {
              try {
                const { link } = await subscriptionAPI.getLink(record.id)
                copyToClipboard(link)
              } catch {
                message.error('获取链接失败')
              }
            }}
          />
          <Button size="small" icon={<EditOutlined />} title="编辑" onClick={() => openManageModal(record)} />
          <Button size="small" danger icon={<DeleteOutlined />} title="删除" onClick={() => triggerDelete(record)} />
        </Space>
      ),
    },
  ]

  return (
    <div className="animate-in">
      {/* Page Header */}
      <div className="page-header" style={{ display: 'flex', alignItems: 'flex-start', justifyContent: 'space-between' }}>
        <div>
          <h1>订阅管理</h1>
          <p>创建和管理用户的订阅链接，支持多服务器账号</p>
        </div>
        <Button type="primary" onClick={openModal} style={{ marginTop: 8 }}>+ 添加订阅</Button>
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

      {/* Subscription Table */}
      <Card className="morandi-card">
        <Table
          columns={columns}
          dataSource={subscriptions}
          rowKey="id"
          loading={loading}
          pagination={{ pageSize: 10 }}
          expandable={{ expandedRowRender: (record) => <TrafficDetail subId={record.id} accounts={record.accounts} /> }}
        />
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
          <Form.Item
            name="name"
            label="名称"
            rules={[
              { required: true, message: '请输入名称' },
              {
                validator: (_, value) => {
                  if (!value) return Promise.resolve()
                  const duplicate = subscriptions.some(s => s.name === value)
                  return duplicate ? Promise.reject('已存在同名订阅，请使用其他名称') : Promise.resolve()
                },
              },
            ]}
          >
            <Input placeholder="例如: VIP用户" />
          </Form.Item>
          <Form.Item name="remark" label="备注">
            <Input placeholder="可选，中文备注，方便识别用户" />
          </Form.Item>
          <Form.Item name="traffic_limit" label="流量限制(GB)">
            <Input type="number" placeholder="留空表示无限流量" />
          </Form.Item>

          <div style={{ marginBottom: 16 }}>
            <h4>选择服务器（每个服务器选一个账号，无账号时自动以订阅名创建）</h4>
            <div style={{ maxHeight: 300, overflowY: 'auto' }}>
              {servers.map(server => {
                const serverAccounts = accounts[server.id] || []
                const isSelected = selectedMappings.some(m => m.server_id === server.id)
                const selectedMapping = selectedMappings.find(m => m.server_id === server.id)
                const isAutoCreate = selectedMapping?.account_id === AUTO_CREATE

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
                      style={{ marginBottom: isSelected ? 8 : 0 }}
                    >
                      <strong><FlagName name={server.name} /></strong> ({server.ip})
                    </Checkbox>

                    {isSelected && (
                      <div style={{ marginLeft: 24 }}>
                        <Select
                          value={selectedMapping?.account_id}
                          onChange={(v) => handleAccountChange(server.id, v)}
                          style={{ width: '100%' }}
                        >
                          <Select.Option value={AUTO_CREATE}>
                            <span style={{ color: '#1677ff' }}>+ 自动创建账号（使用订阅名称）</span>
                          </Select.Option>
                          {serverAccounts.map(acc => (
                            <Select.Option key={acc.id} value={acc.id}>
                              {acc.email} {acc.enabled ? '' : '(已禁用)'}
                            </Select.Option>
                          ))}
                        </Select>
                        {isAutoCreate && (
                          <div style={{ marginTop: 4, fontSize: 12, color: '#888' }}>
                            将自动创建名为「订阅名称」的 VLESS Reality 账号
                          </div>
                        )}
                      </div>
                    )}
                  </div>
                )
              })}
            </div>
          </div>

          <Button type="primary" htmlType="submit" disabled={selectedMappings.length === 0}>
            提交 ({selectedMappings.length} 个服务器)
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
        <div style={{ marginTop: 16 }}>
          <h4 style={{ marginBottom: 8 }}>订阅地址 (通用)</h4>
          <div style={{ display: 'flex', gap: 8, marginBottom: 12 }}>
            <Input value={currentLink} readOnly style={{ flex: 1 }} />
            <Button icon={<CopyOutlined />} onClick={() => copyToClipboard(currentLink)} />
          </div>
          <h4 style={{ marginBottom: 8 }}>Base64 编码 (ShadowRocket/Quantumult)</h4>
          <div style={{ display: 'flex', gap: 8, marginBottom: 20 }}>
            <Input value={currentEncoded} readOnly style={{ flex: 1 }} />
            <Button icon={<CopyOutlined />} onClick={() => copyToClipboard(currentEncoded)} />
          </div>
          <div style={{ textAlign: 'center' }}>
            {currentLink ? (
              <>
                <QRCodeDisplay link={currentLink} label={currentSubInfo?.name || '订阅链接'} />
                <div style={{ marginTop: 8, fontSize: 12, color: '#999' }}>
                  包含 {currentSubInfo?.accounts?.length || 0} 个节点，客户端扫码导入全部节点
                </div>
              </>
            ) : null}
          </div>
          <div style={{ marginTop: 16, fontSize: 12, color: 'var(--text-muted)' }}>
            <p>支持格式: ?format=vless (默认) | ?format=clash_meta | ?format=singbox | ?format=ss (ShadowRocket)</p>
          </div>
        </div>
      </Modal>

      {/* Edit Modal */}
      <Modal
        title={`编辑订阅 · ${manageSubscriptionName}`}
        open={manageModalVisible}
        onCancel={() => setManageModalVisible(false)}
        footer={null}
        width={600}
      >
        <div style={{ marginTop: 16 }}>
          {/* 基本信息 */}
          <div style={{ display: 'flex', alignItems: 'center', gap: 24, marginBottom: 12 }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
              <span style={{ fontSize: 13, color: 'var(--text-muted)' }}>启用</span>
              <Switch checked={editEnable} onChange={setEditEnable} size="small" />
            </div>
            <Input
              value={editRemark}
              onChange={e => setEditRemark(e.target.value)}
              placeholder="备注（可选）"
              style={{ flex: 1 }}
              size="small"
            />
            <Button type="primary" size="small" loading={savingInfo} onClick={handleSaveInfo}>
              保存
            </Button>
          </div>

          <Divider style={{ margin: '12px 0' }} />

          {/* 账号管理 */}
          <div style={{ marginBottom: 12, display: 'flex', justifyContent: 'flex-end' }}>
            <Button type="primary" size="small" onClick={() => setAddAccountModalVisible(true)}>
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

      {/* Delete Confirmation Modal */}
      <Modal
        title="删除订阅"
        open={deleteModalVisible}
        onCancel={() => { setDeleteModalVisible(false); setPendingDeleteSub(null) }}
        onOk={handleDelete}
        okText="确认删除"
        cancelText="取消"
        okButtonProps={{ danger: true, loading: deleting }}
        width={480}
      >
        <p>确定要删除订阅 <strong>「{pendingDeleteSub?.name}」</strong> 吗？</p>
        {orphanedAccounts.length > 0 && (
          <>
            <p style={{ marginBottom: 8 }}>以下关联账号未被其他订阅引用：</p>
            <div style={{ background: '#fafafa', border: '1px solid #f0f0f0', borderRadius: 6, padding: '8px 12px', marginBottom: 12 }}>
              {orphanedAccounts.map(a => (
                <div key={a.id} style={{ padding: '4px 0', fontSize: 13 }}>
                  <Tag color="blue" style={{ marginRight: 6 }}><FlagName name={a.server_name} /></Tag>
                  {a.email}
                </div>
              ))}
            </div>
            <Checkbox
              checked={alsoDeleteAccounts}
              onChange={e => setAlsoDeleteAccounts(e.target.checked)}
            >
              同步删除以上关联账号
            </Checkbox>
          </>
        )}
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
            const hasLinkedAccountFromServer = managedAccounts.some(a => a.server_id === server.id)
            const availableAccounts = hasLinkedAccountFromServer ? [] : serverAccounts.filter(acc => !linkedAccountIds.includes(acc.id))
            const hasAutoNameAccount = serverAccounts.some(acc => acc.email === manageSubscriptionName)

            return (
              <div key={server.id} style={{
                border: '1px solid #d9d9d9',
                borderRadius: 4,
                padding: 12,
                marginBottom: 8,
              }}>
                <strong><FlagName name={server.name} /></strong> ({server.ip})
                {hasLinkedAccountFromServer ? (
                  <div style={{ marginTop: 8, color: '#999' }}>已添加此服务器的账号</div>
                ) : (
                  <div style={{ marginTop: 8 }}>
                    <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 4 }}>
                      <span style={{ color: '#1677ff', fontSize: 13 }}>
                        {hasAutoNameAccount ? `使用同名账号「${manageSubscriptionName}」` : `自动创建账号「${manageSubscriptionName}」`}
                      </span>
                      <Button size="small" type="primary" onClick={() => handleAddAccount(server.id, '', true)}>
                        {hasAutoNameAccount ? '关联' : '创建并关联'}
                      </Button>
                    </div>
                    {availableAccounts.length > 0 && (
                      <div style={{ borderTop: '1px dashed #f0f0f0', paddingTop: 6, marginTop: 6 }}>
                        {availableAccounts.map(acc => (
                          <div key={acc.id} style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 4 }}>
                            <span style={{ color: '#555', fontSize: 13 }}>{acc.email} {acc.enabled ? '' : '(已禁用)'}</span>
                            <Button size="small" onClick={() => handleAddAccount(server.id, acc.id)}>
                              关联
                            </Button>
                          </div>
                        ))}
                      </div>
                    )}
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

// Traffic monitoring detail row
const TRAFFIC_RANGES = [
  { label: '1小时', value: '1h' },
  { label: '3小时', value: '3h' },
  { label: '6小时', value: '6h' },
  { label: '1天', value: '1d' },
  { label: '7天', value: '7d' },
  { label: '30天', value: '30d' },
]

const CHART_W = 800
const CHART_H = 180
const PAD_L = 70
const PAD_R = 16
const PAD_T = 12
const PAD_B = 28

// Morandi-ish multi-series colors
const SERIES_COLORS = ['#5B8DB8', '#C87D55', '#6BA87C', '#9B6B9B', '#B8936A']

const pad2 = (n: number) => n.toString().padStart(2, '0')

// Format time label based on actual data span (not selected range)
function fmtTime(ms: number, spanMs: number): string {
  const d = new Date(ms)
  const hours = spanMs / 3_600_000
  if (hours < 2) return `${pad2(d.getHours())}:${pad2(d.getMinutes())}`
  if (hours < 48) return `${d.getMonth() + 1}/${d.getDate()} ${pad2(d.getHours())}:${pad2(d.getMinutes())}`
  return `${d.getMonth() + 1}/${d.getDate()}`
}

// Smooth bezier path
function smoothPath(pts: { x: number; y: number }[]): string {
  if (pts.length < 2) return ''
  let d = `M ${pts[0].x.toFixed(1)} ${pts[0].y.toFixed(1)}`
  for (let i = 1; i < pts.length - 1; i++) {
    const xc = ((pts[i].x + pts[i + 1].x) / 2).toFixed(1)
    const yc = ((pts[i].y + pts[i + 1].y) / 2).toFixed(1)
    d += ` Q ${pts[i].x.toFixed(1)} ${pts[i].y.toFixed(1)} ${xc} ${yc}`
  }
  d += ` L ${pts[pts.length - 1].x.toFixed(1)} ${pts[pts.length - 1].y.toFixed(1)}`
  return d
}

function TrafficDetail({ subId, accounts }: { subId: string; accounts?: AccountWithServer[] }) {
  const [range, setRange] = useState('3h')
  const [series, setSeries] = useState<AccountTrafficSeries[]>([])
  const [fetching, setFetching] = useState(false)
  const [hoverX, setHoverX] = useState<number | null>(null)
  const [svgW, setSvgW] = useState(0)
  const svgRef = useRef<SVGSVGElement>(null)
  const wrapRef = useRef<HTMLDivElement>(null)

  // Observe the wrapper div — div clientWidth is reliable, SVG bbox is not
  useEffect(() => {
    const el = wrapRef.current
    if (!el) return
    const update = () => {
      const w = el.clientWidth
      if (w > 0) setSvgW(w)
    }
    update()
    const obs = new ResizeObserver(update)
    obs.observe(el)
    return () => obs.disconnect()
  }, [])

  // Direct effect on [subId, range] — no useCallback indirection
  useEffect(() => {
    let cancelled = false
    setFetching(true)
    setHoverX(null)
    subscriptionAPI.getAccountTrafficLogs(subId, range)
      .then(data => { if (!cancelled) setSeries(data || []) })
      .catch(() => { if (!cancelled) setSeries([]) })
      .finally(() => { if (!cancelled) setFetching(false) })
    return () => { cancelled = true }
  }, [subId, range])

  const innerW = svgW - PAD_L - PAD_R
  const innerH = CHART_H - PAD_T - PAD_B

  // Compute delta series (traffic consumed per interval)
  const deltaSeries = series.map(s => ({
    ...s,
    deltas: s.points.map((p, i) => {
      const t = typeof p.time === 'string' ? new Date(p.time).getTime() : (p.time as unknown as Date).getTime()
      return {
        t,
        value: i === 0 ? 0 : Math.max(0, p.value - s.points[i - 1].value),
      }
    }),
  }))

  const totalDelta = deltaSeries.reduce((sum, s) =>
    sum + s.deltas.reduce((a, p) => a + p.value, 0), 0)

  // Overall time range from actual data
  const allTs = deltaSeries.flatMap(s => s.deltas.map(p => p.t)).filter(Boolean)
  const minT = allTs.length ? Math.min(...allTs) : 0
  const maxT = allTs.length ? Math.max(...allTs) : 1
  const spanMs = Math.max(maxT - minT, 1)

  // Y-axis
  const allValues = deltaSeries.flatMap(s => s.deltas.map(p => p.value))
  const maxVal = Math.max(...allValues, 1)
  const magnitude = Math.pow(10, Math.floor(Math.log10(maxVal)))
  const niceMax = Math.ceil(maxVal / magnitude) * magnitude

  // Map time → SVG X coordinate
  const toX = (t: number) => PAD_L + ((t - minT) / spanMs) * innerW
  const toY = (v: number) => PAD_T + innerH - (v / niceMax) * innerH

  // Build SVG points per series (each series uses its own timestamps)
  const seriesPts = deltaSeries.map(s =>
    s.deltas.map(p => ({ x: toX(p.t), y: toY(p.value), t: p.t, value: p.value }))
  )

  // X-axis: 6 evenly-spaced labels across [minT, maxT]
  const xLabels = allTs.length > 0
    ? Array.from({ length: 6 }, (_, i) => {
        const t = minT + (i / 5) * spanMs
        return { x: PAD_L + (i / 5) * innerW, label: fmtTime(t, spanMs) }
      })
    : []

  // Hover: find nearest point per series
  const hoverT = hoverX !== null ? minT + ((hoverX - PAD_L) / innerW) * spanMs : null
  const hoverPts = hoverT !== null
    ? seriesPts.map(pts => {
        if (!pts.length) return null
        return pts.reduce((best, p) =>
          Math.abs(p.t - hoverT) < Math.abs(best.t - hoverT) ? p : best
        )
      })
    : seriesPts.map(() => null)
  const hoverXPos = hoverPts.some(p => p !== null) && hoverT !== null
    ? toX(Math.max(minT, Math.min(maxT, hoverT)))
    : null

  const yLevels = [0, 0.25, 0.5, 0.75, 1.0]
  const hasData = allTs.length > 0

  // Build left panel: match deltaSeries (colors) with accounts (traffic_used)
  const leftItems = deltaSeries.map((s, i) => {
    const acc = accounts?.find(a => a.server_name === s.server_name && a.email === s.email)
    return { color: SERIES_COLORS[i % SERIES_COLORS.length], serverName: s.server_name, email: s.email, trafficUsed: acc?.traffic_used ?? 0, key: s.account_id }
  })

  return (
    <div style={{ padding: '8px 0', opacity: fetching ? 0.65 : 1, transition: 'opacity 0.2s' }}>
      {/* Header: title + legend (centered) + range tabs */}
      <div style={{ display: 'flex', alignItems: 'center', gap: 12, marginBottom: 10, padding: '0 16px' }}>
        <span style={{ fontWeight: 600, fontSize: 14, flexShrink: 0 }}>流量使用趋势</span>
        <span style={{ color: '#999', fontSize: 13, flexShrink: 0 }}>时间段内消耗: {formatBytes(totalDelta)}</span>
        {/* Node legend centered */}
        <div style={{ flex: 1, display: 'flex', justifyContent: 'center', flexWrap: 'wrap', gap: '4px 16px' }}>
          {leftItems.map(item => (
            <div key={item.key} style={{
              display: 'flex', alignItems: 'center', gap: 5,
              paddingLeft: 7, borderLeft: `3px solid ${item.color}`,
            }}>
              <Tag color="blue" style={{ margin: 0, fontSize: 11, padding: '0 4px', flexShrink: 0 }}>
                <FlagName name={item.serverName} />
              </Tag>
              <span style={{ color: '#555', fontSize: 12 }}>{item.email}</span>
              <span style={{ color: '#999', fontSize: 11, fontVariantNumeric: 'tabular-nums' }}>{formatBytes(item.trafficUsed)}</span>
            </div>
          ))}
        </div>
        <div style={{ display: 'flex', gap: 6, flexShrink: 0 }}>
          {TRAFFIC_RANGES.map(r => (
            <Button key={r.value} size="small"
              type={range === r.value ? 'primary' : 'default'}
              onClick={() => setRange(r.value)}
            >{r.label}</Button>
          ))}
        </div>
      </div>

      {/* Chart full width — wrapRef measures actual container pixel width */}
      <div ref={wrapRef} style={{ width: '100%' }}>
          {!hasData || svgW === 0 ? (
            <div style={{ color: '#bbb', padding: '24px 0', textAlign: 'center', fontSize: 13 }}>
              {!hasData ? '暂无流量数据（切换时间范围或等待下一次心跳上报后生效）' : ''}
            </div>
          ) : (
            <svg ref={svgRef} width={svgW} height={CHART_H}
              style={{ display: 'block', cursor: 'crosshair' }}
              onMouseMove={e => {
                setHoverX(e.clientX - svgRef.current!.getBoundingClientRect().left)
              }}
              onMouseLeave={() => setHoverX(null)}
            >
              <defs>
                {deltaSeries.map((s, i) => (
                  <linearGradient key={s.account_id} id={`fill-${subId}-${i}`} x1="0" y1="0" x2="0" y2="1">
                    <stop offset="0%" stopColor={SERIES_COLORS[i % SERIES_COLORS.length]} stopOpacity="0.18" />
                    <stop offset="100%" stopColor={SERIES_COLORS[i % SERIES_COLORS.length]} stopOpacity="0.01" />
                  </linearGradient>
                ))}
              </defs>

              {/* Y gridlines + labels */}
              {yLevels.map(level => {
                const y = PAD_T + innerH - level * innerH
                return (
                  <g key={level}>
                    <line x1={PAD_L} y1={y} x2={PAD_L + innerW} y2={y}
                      stroke={level === 0 ? '#ccc' : '#ebebeb'} strokeWidth="1" />
                    <text x={PAD_L - 6} y={y + 4} textAnchor="end" fontSize="11" fill="#aaa">
                      {formatBytes(level * niceMax)}
                    </text>
                  </g>
                )
              })}

              {/* X labels */}
              {xLabels.map((l, i) => (
                <text key={i} x={l.x} y={PAD_T + innerH + 18} textAnchor="middle" fontSize="11" fill="#aaa">
                  {l.label}
                </text>
              ))}

              {/* Series: fill + smooth line */}
              {seriesPts.map((pts, i) => {
                if (pts.length < 2) return null
                const color = SERIES_COLORS[i % SERIES_COLORS.length]
                const linePath = smoothPath(pts)
                const baseY = PAD_T + innerH
                const fillPath = `${linePath} L ${pts[pts.length-1].x.toFixed(1)} ${baseY} L ${pts[0].x.toFixed(1)} ${baseY} Z`
                return (
                  <g key={series[i].account_id}>
                    <path d={fillPath} fill={`url(#fill-${subId}-${i})`} />
                    <path d={linePath} fill="none" stroke={color} strokeWidth="2"
                      strokeLinecap="round" strokeLinejoin="round" />
                  </g>
                )
              })}

              {/* Hover overlay */}
              {hoverXPos !== null && (
                <g>
                  <line x1={hoverXPos} y1={PAD_T} x2={hoverXPos} y2={PAD_T + innerH}
                    stroke="#bbb" strokeWidth="1" strokeDasharray="4,3" />
                  {hoverPts.map((p, i) => {
                    if (!p) return null
                    return <circle key={i} cx={p.x} cy={p.y} r="4"
                      fill={SERIES_COLORS[i % SERIES_COLORS.length]} stroke="white" strokeWidth="1.5" />
                  })}
                  {(() => {
                    const tipW = 160, tipH = 18 + hoverPts.filter(Boolean).length * 18 + 6
                    const tipX = hoverXPos + 12 > svgW - tipW - PAD_R ? hoverXPos - tipW - 12 : hoverXPos + 12
                    const tipY = PAD_T + 4
                    const timeStr = hoverT !== null ? fmtTime(Math.max(minT, Math.min(maxT, hoverT)), spanMs) : ''
                    return (
                      <g>
                        <rect x={tipX} y={tipY} width={tipW} height={tipH} rx="5"
                          fill="white" stroke="#e8e8e8" strokeWidth="1"
                          style={{ filter: 'drop-shadow(0 2px 6px rgba(0,0,0,0.07))' }} />
                        <text x={tipX + 10} y={tipY + 14} fontSize="11" fill="#999">{timeStr}</text>
                        {hoverPts.map((p, i) => {
                          if (!p) return null
                          const color = SERIES_COLORS[i % SERIES_COLORS.length]
                          const row = hoverPts.slice(0, i).filter(Boolean).length
                          return (
                            <g key={i}>
                              <rect x={tipX + 10} y={tipY + 22 + row * 18} width={8} height={2} rx="1" fill={color} />
                              <text x={tipX + 22} y={tipY + 32 + row * 18} fontSize="11" fill="#555">
                                {deltaSeries[i].server_name}: {formatBytes(p.value)}
                              </text>
                            </g>
                          )
                        })}
                      </g>
                    )
                  })()}
                </g>
              )}
            </svg>
          )}
      </div>
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
          <div style={{ fontWeight: 'bold' }}><FlagName name={account.server_name} /> / {account.email}</div>
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
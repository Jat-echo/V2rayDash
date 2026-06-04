# 批量分配订阅 & 按钮图标化 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将服务器页和订阅页操作按钮改为图标按钮，并在账号管理弹窗中增加「关联订阅」Tab 支持批量分配服务器到多个订阅。

**Architecture:** 纯前端改动，复用现有 `subscriptionAPI.listFull()`、`accountAPI.listByServer()`、`subscriptionAPI.addAccount()` 三个接口，无需新增后端接口。在 servers/index.tsx 中新增状态和逻辑；账号管理 Modal 用 Tabs 拆分为「账号」和「关联订阅」两个 Tab。

**Tech Stack:** React 18, TypeScript, Ant Design 5, @ant-design/icons

---

## 文件变动

| 操作 | 文件 | 改动 |
|---|---|---|
| Modify | `frontend/src/pages/subscriptions/index.tsx` | 操作按钮图标化 |
| Modify | `frontend/src/pages/servers/index.tsx` | 操作按钮图标化 + Tabs + 关联订阅 Tab |

---

## Task 1: 订阅页操作按钮图标化

**Files:**
- Modify: `frontend/src/pages/subscriptions/index.tsx:3` (icons import)
- Modify: `frontend/src/pages/subscriptions/index.tsx:350-368` (action buttons)

- [ ] **Step 1: 扩展 icons import**

找到第 3 行：
```ts
import { CopyOutlined, QrcodeOutlined, HolderOutlined } from '@ant-design/icons'
```

替换为：
```ts
import { CopyOutlined, QrcodeOutlined, HolderOutlined, LinkOutlined, EditOutlined, DeleteOutlined } from '@ant-design/icons'
```

- [ ] **Step 2: 替换操作列按钮**

找到操作列 render（约第 350-368 行），当前内容：
```tsx
render: (_: any, record: SubscriptionWithAccounts) => (
  <Space>
    <Button size="small" type="primary" onClick={() => handleGetLink(record.id)}>订阅链接</Button>
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
    <Button size="small" onClick={() => openManageModal(record)}>编辑</Button>
    <Button size="small" danger onClick={() => triggerDelete(record)}>删除</Button>
  </Space>
),
```

替换为：
```tsx
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
```

- [ ] **Step 3: 编译验证**

```bash
cd /home/jat-id/Project/V2rayDash/frontend
npm run build 2>&1 | tail -5
```

预期：末尾含 `built in` 字样，无 TypeScript 错误。

- [ ] **Step 4: Commit**

```bash
cd /home/jat-id/Project/V2rayDash
git add frontend/src/pages/subscriptions/index.tsx
git commit -m "feat(frontend): convert subscription action buttons to icons"
```

---

## Task 2: 服务器页操作按钮图标化

**Files:**
- Modify: `frontend/src/pages/servers/index.tsx:1-5` (add icons import)
- Modify: `frontend/src/pages/servers/index.tsx:396-413` (action buttons)

- [ ] **Step 1: 新增 icons import**

找到第 1-5 行 imports，在第 5 行（`import { FlagName ...`）之前插入：
```ts
import { CloudUploadOutlined, EditOutlined, TeamOutlined, ReloadOutlined, DeleteOutlined } from '@ant-design/icons'
```

文件头部 imports 区域最终应为：
```ts
import { useState, useEffect, useRef } from 'react'
import { Table, Button, Space, Modal, Form, Input, InputNumber, Select, message, Popconfirm, Tag, Card, Alert, Segmented } from 'antd'
import { CloudUploadOutlined, EditOutlined, TeamOutlined, ReloadOutlined, DeleteOutlined } from '@ant-design/icons'
import { serverAPI, accountAPI, logAPI, Server, Account, NodeStatusResponse, MetricPoint, BandwidthPoint } from '../../services/api'
import { formatBytes } from '../../utils/format'
import { FlagName } from '../../components/FlagName'
```

- [ ] **Step 2: 替换操作列按钮**

找到操作列 render（约第 396-413 行），当前内容：
```tsx
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
```

替换为：
```tsx
{
  title: '操作',
  render: (_: any, record: Server) => (
    <Space>
      <Button size="small" type="primary" icon={<CloudUploadOutlined />} title="安装 Agent" onClick={(e) => { e.stopPropagation(); handleInstallClick(record); }} />
      <Button size="small" icon={<EditOutlined />} title="编辑" onClick={(e) => { e.stopPropagation(); handleEditClick(record); }} />
      <Button size="small" icon={<TeamOutlined />} title="账号管理" onClick={(e) => { e.stopPropagation(); handleOpenAccountModal(record); }} />
      <Button
        size="small"
        icon={<ReloadOutlined />}
        title="重启 xray"
        loading={restartingXray === record.id}
        onClick={(e) => handleRestartXray(record.id, e)}
      />
      <Popconfirm title="确定删除?" onConfirm={() => handleDelete(record.id)}>
        <Button size="small" danger icon={<DeleteOutlined />} title="删除" onClick={(e) => e.stopPropagation()} />
      </Popconfirm>
    </Space>
  ),
},
```

- [ ] **Step 3: 编译验证**

```bash
cd /home/jat-id/Project/V2rayDash/frontend
npm run build 2>&1 | tail -5
```

预期：末尾含 `built in`，无错误。

- [ ] **Step 4: Commit**

```bash
cd /home/jat-id/Project/V2rayDash
git add frontend/src/pages/servers/index.tsx
git commit -m "feat(frontend): convert server action buttons to icons"
```

---

## Task 3: 账号管理弹窗增加「关联订阅」Tab

**Files:**
- Modify: `frontend/src/pages/servers/index.tsx` (多处)

### 改动范围概览

1. antd import 新增 `Tabs, Spin, Checkbox`
2. API import 新增 `subscriptionAPI, Subscription`
3. 新增 5 个状态变量
4. 新增 `getSubAssignStatus` 模块级函数
5. 新增 `loadAssignData` + `handleAssign` 函数
6. 新增 `accountModalTab` 状态 + 重置逻辑
7. 替换 Account Modal JSX（改为 Tabs 结构）

- [ ] **Step 1: 扩展 antd import**

找到第 2 行：
```ts
import { Table, Button, Space, Modal, Form, Input, InputNumber, Select, message, Popconfirm, Tag, Card, Alert, Segmented } from 'antd'
```

替换为：
```ts
import { Table, Button, Space, Modal, Form, Input, InputNumber, Select, message, Popconfirm, Tag, Card, Alert, Segmented, Tabs, Spin, Checkbox } from 'antd'
```

- [ ] **Step 2: 扩展 API import**

找到第 4 行（服务器页 API import，`serverAPI, accountAPI, logAPI ...`）：
```ts
import { serverAPI, accountAPI, logAPI, Server, Account, NodeStatusResponse, MetricPoint, BandwidthPoint } from '../../services/api'
```

替换为：
```ts
import { serverAPI, accountAPI, logAPI, subscriptionAPI, Server, Account, Subscription, NodeStatusResponse, MetricPoint, BandwidthPoint } from '../../services/api'
```

- [ ] **Step 3: 新增模块级 helper 函数**

在文件顶部（`function ansiToHtml` 定义之前）插入：

```ts
function getSubAssignStatus(
  sub: Subscription,
  serverId: string,
  serverAccounts: Account[],
): 'assigned' | 'existing' | 'new' {
  if (sub.accounts?.some(a => a.server_id === serverId)) return 'assigned'
  if (serverAccounts.some(a => a.email === sub.name)) return 'existing'
  return 'new'
}
```

- [ ] **Step 4: 新增状态变量**

找到组件内现有状态变量区（约第 60-82 行，`const [restartingXray...` 之后），在其后插入：

```ts
const [accountModalTab, setAccountModalTab] = useState<string>('accounts')
const [assignSubs, setAssignSubs] = useState<Subscription[]>([])
const [assignServerAccounts, setAssignServerAccounts] = useState<Account[]>([])
const [assignLoading, setAssignLoading] = useState(false)
const [assignSelected, setAssignSelected] = useState<string[]>([])
const [assignSubmitting, setAssignSubmitting] = useState(false)
```

- [ ] **Step 5: 新增 loadAssignData 函数**

在组件内现有函数（`loadServers`、`handleAdd` 等）附近插入：

```ts
const loadAssignData = async () => {
  if (!selectedServerForAccounts) return
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
```

- [ ] **Step 6: 新增 handleAssign 函数**

紧接 `loadAssignData` 之后插入：

```ts
const handleAssign = async () => {
  if (!selectedServerForAccounts || assignSelected.length === 0) return
  setAssignSubmitting(true)
  try {
    const results = await Promise.allSettled(
      assignSelected.map(id =>
        subscriptionAPI.addAccount(id, { server_id: selectedServerForAccounts.id, auto_create: true })
      )
    )
    const succeeded = results.filter(r => r.status === 'fulfilled').length
    const failed = results.filter(r => r.status === 'rejected').length
    if (failed === 0) {
      message.success(`成功分配 ${succeeded} 个订阅`)
    } else {
      message.warning(`${succeeded} 个成功，${failed} 个失败`)
    }
    loadAssignData()
  } finally {
    setAssignSubmitting(false)
  }
}
```

- [ ] **Step 7: 替换 Account Modal JSX**

找到 `{/* Account Modal */}` 注释开始的整个 Modal（约第 640-691 行），替换为：

```tsx
{/* Account Modal */}
<Modal
  title={`账号管理 · ${selectedServerForAccounts?.name || ''}`}
  open={accountModalVisible}
  onCancel={() => { setAccountModalVisible(false); setAccountModalTab('accounts') }}
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
              const selectableSubs = assignSubs.filter(
                sub => getSubAssignStatus(sub, serverId, assignServerAccounts) !== 'assigned'
              )
              const allSelected = selectableSubs.length > 0 &&
                selectableSubs.every(s => assignSelected.includes(s.id))
              const someSelected = selectableSubs.some(s => assignSelected.includes(s.id)) && !allSelected
              return (
                <>
                  <div style={{ marginBottom: 8, paddingBottom: 8, borderBottom: '1px solid #f0f0f0' }}>
                    <Checkbox
                      checked={allSelected}
                      indeterminate={someSelected}
                      onChange={e => setAssignSelected(
                        e.target.checked ? selectableSubs.map(s => s.id) : []
                      )}
                    >
                      全选（{selectableSubs.length} 个可分配）
                    </Checkbox>
                  </div>
                  <div style={{ maxHeight: 320, overflowY: 'auto' }}>
                    {assignSubs.map(sub => {
                      const status = getSubAssignStatus(sub, serverId, assignServerAccounts)
                      const existingAcc = assignServerAccounts.find(a => a.email === sub.name)
                      const disabled = status === 'assigned'
                      return (
                        <div
                          key={sub.id}
                          style={{
                            display: 'flex',
                            alignItems: 'center',
                            justifyContent: 'space-between',
                            padding: '8px 0',
                            borderBottom: '1px solid #f0f0f0',
                          }}
                        >
                          <Checkbox
                            disabled={disabled}
                            checked={assignSelected.includes(sub.id)}
                            onChange={e => setAssignSelected(
                              e.target.checked
                                ? [...assignSelected, sub.id]
                                : assignSelected.filter(id => id !== sub.id)
                            )}
                          >
                            <span style={{ fontWeight: 500 }}>{sub.name}</span>
                            {sub.remark && (
                              <span style={{ color: '#999', marginLeft: 6, fontSize: 12 }}>
                                {sub.remark}
                              </span>
                            )}
                          </Checkbox>
                          {status === 'assigned' && <Tag>已分配</Tag>}
                          {status === 'existing' && (
                            <Tag color="blue">已有账号 · {existingAcc?.email}</Tag>
                          )}
                          {status === 'new' && (
                            <Tag color="default">将新建 · {sub.name}</Tag>
                          )}
                        </div>
                      )
                    })}
                    {assignSubs.length === 0 && !assignLoading && (
                      <div style={{ textAlign: 'center', color: '#999', padding: '24px 0' }}>
                        暂无订阅
                      </div>
                    )}
                  </div>
                  <div style={{ marginTop: 16, textAlign: 'right' }}>
                    <Button
                      type="primary"
                      loading={assignSubmitting}
                      disabled={assignSelected.length === 0}
                      onClick={handleAssign}
                    >
                      确认分配（已选 {assignSelected.length} 个）
                    </Button>
                  </div>
                </>
              )
            })()}
          </Spin>
        ),
      },
    ]}
  />
</Modal>
```

- [ ] **Step 8: 编译验证**

```bash
cd /home/jat-id/Project/V2rayDash/frontend
npm run build 2>&1 | tail -8
```

预期：末尾含 `built in`，无 TypeScript 错误。

- [ ] **Step 9: Commit**

```bash
cd /home/jat-id/Project/V2rayDash
git add frontend/src/pages/servers/index.tsx
git commit -m "feat(frontend): add assign-subscriptions tab to server account modal"
```

---

## 验收检查

完成所有 Task 后，启动前端开发服务器验证：

```bash
cd /home/jat-id/Project/V2rayDash/frontend && npm run dev
```

1. **订阅页**：操作列显示 4 个图标按钮，悬停显示 tooltip（订阅链接、复制订阅链接、编辑、删除）
2. **服务器页**：操作列显示 5 个图标按钮（安装 Agent、编辑、账号管理、重启 xray、删除）
3. **账号管理弹窗**：弹窗内有「账号」和「关联订阅」两个 Tab
4. **关联订阅 Tab**：切换时加载订阅列表，每行显示正确状态 Tag，「已分配」禁用，其余可勾选
5. **确认分配**：勾选订阅后点击按钮，等待后显示成功/失败消息，列表状态刷新

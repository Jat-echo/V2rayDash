# 批量删除关联订阅 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在「关联订阅」Tab 中将单列布局改为双列，左列展示已关联订阅（支持批量删除关联+账号），右列展示可分配订阅（保持现有逻辑）。

**Architecture:** 纯前端改动，仅修改 `servers/index.tsx`。新增 `deleteSelected`/`deleteSubmitting` 状态和 `handleDeleteAssign` 函数。将现有 IIFE 单列 JSX 替换为双列 flex 布局，左右列各有独立全选 + action 按钮。

**Tech Stack:** React 18, TypeScript, Ant Design 5（Spin, Checkbox, Button, Popconfirm, Tag）

---

## 文件变动

| 操作 | 文件 | 改动 |
|---|---|---|
| Modify | `frontend/src/pages/servers/index.tsx` | 新增状态、函数；替换 assign tab JSX |

---

## 当前文件关键位置参考

- 第 2 行：antd import（已有 Spin, Checkbox, Popconfirm）
- 第 93-99 行：assign 相关状态变量区
- 第 703-709 行：账号管理 Modal 的 onCancel
- 第 162-178 行：`loadAssignData` 函数
- 第 180-200 行：`handleAssign` 函数
- 第 770-858 行：key='assign' Tab 的 children（当前为 IIFE 单列）

---

## Task 1: 新增状态变量、handleDeleteAssign 函数，更新 onCancel

**Files:**
- Modify: `frontend/src/pages/servers/index.tsx`

- [ ] **Step 1: 在第 99 行之后插入两个新状态变量**

找到第 99 行：
```ts
  const [assignSubmitting, setAssignSubmitting] = useState(false)
```

在其后插入：
```ts
  const [deleteSelected, setDeleteSelected] = useState<string[]>([])
  const [deleteSubmitting, setDeleteSubmitting] = useState(false)
```

- [ ] **Step 2: 在 handleAssign 函数之后插入 handleDeleteAssign**

找到 `handleAssign` 函数末尾（约第 200 行的 `}`），在其后插入：

```ts
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
      setDeleteSelected([])
      await loadAssignData()
    } finally {
      setDeleteSubmitting(false)
    }
  }
```

- [ ] **Step 3: 更新 onCancel（第 703-709 行）**

找到当前 onCancel：
```tsx
        onCancel={() => {
          setAccountModalVisible(false)
          setAccountModalTab('accounts')
          setAssignSubs([])
          setAssignServerAccounts([])
          setAssignSelected([])
        }}
```

替换为（新增最后两行）：
```tsx
        onCancel={() => {
          setAccountModalVisible(false)
          setAccountModalTab('accounts')
          setAssignSubs([])
          setAssignServerAccounts([])
          setAssignSelected([])
          setDeleteSelected([])
        }}
```

- [ ] **Step 4: 编译验证**

```bash
cd /home/jat-id/Project/V2rayDash/frontend
npm run build 2>&1 | tail -5
```

预期：末尾含 `built in`，无 TypeScript 错误。

- [ ] **Step 5: Commit**

```bash
cd /home/jat-id/Project/V2rayDash
git add frontend/src/pages/servers/index.tsx
git commit -m "feat(frontend): add deleteSelected state and handleDeleteAssign function"
```

---

## Task 2: 替换 assign Tab JSX 为双列布局

**Files:**
- Modify: `frontend/src/pages/servers/index.tsx:770-858`

- [ ] **Step 1: 替换 assign tab children**

找到约第 769-858 行的整个 key='assign' item：
```tsx
            {
              key: 'assign',
              label: '关联订阅',
              children: (
                <Spin spinning={assignLoading}>
                  {(() => {
                    ...（当前 IIFE 单列内容）...
                  })()}
                </Spin>
              ),
            },
```

替换为：
```tsx
            {
              key: 'assign',
              label: '关联订阅',
              children: (
                <Spin spinning={assignLoading}>
                  {(() => {
                    const serverId = selectedServerForAccounts?.id || ''
                    const assignedSubs = assignSubs.filter(
                      sub => getSubAssignStatus(sub, serverId, assignServerAccounts) === 'assigned'
                    )
                    const selectableSubs = assignSubs.filter(
                      sub => getSubAssignStatus(sub, serverId, assignServerAccounts) !== 'assigned'
                    )
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
                                    <span style={{ color: '#999', marginLeft: 6, fontSize: 12 }}>
                                      {sub.remark}
                                    </span>
                                  )}
                                </Checkbox>
                                <Tag>已分配</Tag>
                              </div>
                            ))}
                            {assignedSubs.length === 0 && !assignLoading && (
                              <div style={{ textAlign: 'center', color: '#999', padding: '24px 0', fontSize: 13 }}>
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
                              const status = getSubAssignStatus(sub, serverId, assignServerAccounts)
                              const existingAcc = assignServerAccounts.find(a => a.email === sub.name)
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
                                      <span style={{ color: '#999', marginLeft: 6, fontSize: 12 }}>
                                        {sub.remark}
                                      </span>
                                    )}
                                  </Checkbox>
                                  {status === 'existing' && (
                                    <Tag color="blue">已有账号 · {existingAcc?.email}</Tag>
                                  )}
                                  {status === 'new' && (
                                    <Tag color="default">将新建 · {sub.name}</Tag>
                                  )}
                                </div>
                              )
                            })}
                            {selectableSubs.length === 0 && !assignLoading && (
                              <div style={{ textAlign: 'center', color: '#999', padding: '24px 0', fontSize: 13 }}>
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
```

- [ ] **Step 2: 编译验证**

```bash
cd /home/jat-id/Project/V2rayDash/frontend
npm run build 2>&1 | tail -5
```

预期：末尾含 `built in`，无 TypeScript 错误。

- [ ] **Step 3: Commit**

```bash
cd /home/jat-id/Project/V2rayDash
git add frontend/src/pages/servers/index.tsx
git commit -m "feat(frontend): split assign tab into two-column layout with batch delete support"
```

---

## 验收检查

启动前端验证：

```bash
cd /home/jat-id/Project/V2rayDash/frontend && npm run dev
```

1. 打开服务器页，点击账号管理图标，切换到「关联订阅」Tab
2. **左列「已关联」**：
   - 展示已分配的订阅，每行有 checkbox + 「已分配」Tag
   - 全选 checkbox 仅影响左列
   - 选中后「删除关联（已选 N 个）」按钮启用
   - 点击触发 Popconfirm 确认弹窗
   - 确认后成功消息，列表刷新（选中的订阅移至右列）
3. **右列「可分配」**：
   - 展示未分配订阅及状态 Tag（将新建/已有账号）
   - 全选 checkbox 仅影响右列
   - 「确认分配」功能与之前一致
4. **关闭弹窗**：再次打开后两列均无勾选状态

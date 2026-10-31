import {useEffect, useState} from 'react'
import {Alert, Button, Checkbox, Empty, Input, Modal, Popconfirm, Select, Space, Spin, Table, Tag, message} from 'antd'
import {BatchSubscriptionResult, RelayCombination, Server, Subscription, relayAPI, serverAPI, subscriptionAPI} from '../services/api'
import {apiError} from './NodeActions'
import {accountTargetKey, nodeTargetOptions, targetMapping} from '../utils/node-targets'
import './SubscriptionAssignments.css'

export default function BatchSubscriptionDrawer({open, target, onClose, onChange}: {open: boolean; target?: string; onClose: () => void; onChange: () => void}) {
  const [servers, setServers] = useState<Server[]>([])
  const [combinations, setCombinations] = useState<RelayCombination[]>([])
  const [subscriptions, setSubscriptions] = useState<Subscription[]>([])
  const [selectedTarget, setSelectedTarget] = useState<string>()
  const [search, setSearch] = useState('')
  const [selectedAdd, setSelectedAdd] = useState<string[]>([])
  const [selectedRemove, setSelectedRemove] = useState<string[]>([])
  const [results, setResults] = useState<BatchSubscriptionResult[]>([])
  const [loading, setLoading] = useState(false)
  const [listReady, setListReady] = useState(false)
  const [saving, setSaving] = useState<'add' | 'remove' | null>(null)
  const [error, setError] = useState('')
  useEffect(() => {
    if (!open) return
    let active = true
    setSelectedTarget(target); setResults([]); setSearch(''); setSelectedAdd([]); setSelectedRemove([]); setError(''); setLoading(true); setListReady(false)
    Promise.all([serverAPI.list(), relayAPI.list(), subscriptionAPI.listFull()]).then(([nodes, paths, subs]) => {
      if (active) {setServers(nodes); setCombinations(paths); setSubscriptions(subs); setListReady(true)}
    }).catch(err => {if (active) setError(apiError(err, '加载订阅失败，请关闭后重试'))}).finally(() => {if (active) setLoading(false)})
    return () => {active = false}
  }, [open, target])
  const options = nodeTargetOptions(servers, combinations)
  const label = options.flatMap(group => group.options).find(option => option.value === selectedTarget)?.label
  const visible = subscriptions.filter(sub => (sub.name + ' ' + (sub.remark || '')).toLowerCase().includes(search.trim().toLowerCase()))
  const linked = (sub: Subscription) => sub.accounts?.some(account => accountTargetKey(account) === selectedTarget)
  const assigned = visible.filter(linked), available = visible.filter(sub => !linked(sub))
  const submit = async (action: 'add' | 'remove') => {
    if (!selectedTarget) return
    const ids = action === 'add' ? selectedAdd : selectedRemove
    if (!ids.length || ids.length > 200) return
    setSaving(action); setError(''); setResults([])
    try {
      const response = await subscriptionAPI.batch({...targetMapping(selectedTarget), subscription_ids: ids, action})
      setResults(response.results)
      const failed = response.results.filter(item => item.status === 'failed')
      if (action === 'add') setSelectedAdd(failed.map(item => item.id!))
      else setSelectedRemove(failed.map(item => item.id!))
      if (failed.length) message.warning('已处理，' + failed.length + ' 项失败，可查看原因后重试')
      else message.success(action === 'add' ? '订阅已关联，所需账号已自动分配' : '关联已解除')
      onChange()
      try {setSubscriptions(await subscriptionAPI.listFull())}
      catch {setListReady(false);setError('关联操作已提交，列表刷新失败。请关闭后重新打开查看结果。')}
    } catch (err) {setError(apiError(err, '操作失败，请重试'))}
    finally {setSaving(null)}
  }
  const statusLabels = {created: '已新建', added: '已关联', removed: '已解除', skipped: '无须变更', failed: '失败'}
  return <Modal title="关联订阅" open={open} onCancel={onClose} footer={null} width={800} maskClosable={!saving} keyboard={!saving} closable={!saving}>
    {target ? <p className="subscription-target">{label || '正在加载节点…'}</p> : <Select aria-label="选择关联目标节点" loading={loading} disabled={!!saving} placeholder="选择直连节点或代理链路" value={selectedTarget} onChange={value => {setSelectedTarget(value); setSelectedAdd([]); setSelectedRemove([]); setResults([])}} options={options} showSearch optionFilterProp="label" style={{width:'100%',marginBottom:16}} />}
    <Input.Search aria-label="搜索订阅" placeholder="搜索订阅名称或备注" allowClear value={search} onChange={event => setSearch(event.target.value)} disabled={!!saving} style={{marginBottom:16}} />
    {error && <Alert style={{marginBottom:16}} type="error" showIcon message={error} />}
    <Spin spinning={loading}>
      {!selectedTarget ? <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="先选择目标节点，再勾选要关联的订阅。" /> : <div className="subscription-assignment-columns">
        {([{key:'remove',title:'已关联',items:assigned,selected:selectedRemove,setSelected:setSelectedRemove}, {key:'add',title:'可分配',items:available,selected:selectedAdd,setSelected:setSelectedAdd}] as const).map(panel => <section key={panel.key} className="subscription-assignment-panel" aria-label={panel.title + '订阅'}>
          <div className="subscription-assignment-header"><Checkbox disabled={!listReady || !!saving || !panel.items.length} checked={panel.items.length > 0 && panel.items.every(sub => panel.selected.includes(sub.id))} indeterminate={panel.items.some(sub => panel.selected.includes(sub.id)) && !panel.items.every(sub => panel.selected.includes(sub.id))} onChange={event => panel.setSelected(event.target.checked ? [...new Set([...panel.selected,...panel.items.map(sub => sub.id)])] : panel.selected.filter(id => !panel.items.some(sub => sub.id === id)))}>{panel.title}（{panel.items.length} 个）</Checkbox></div>
          <div className="subscription-assignment-list">{panel.items.map(sub => <div key={sub.id} className="subscription-assignment-row">
            <Checkbox disabled={!listReady || !!saving} checked={panel.selected.includes(sub.id)} onChange={event => panel.setSelected(event.target.checked ? [...panel.selected,sub.id] : panel.selected.filter(id => id !== sub.id))}><span>{sub.name}</span>{sub.remark && <span className="subscription-assignment-remark">{sub.remark}</span>}</Checkbox>
            <Tag>{panel.key === 'remove' ? '已关联' : '自动分配账号'}</Tag>
          </div>)}{!panel.items.length && !loading && <p className="subscription-assignment-empty">{search ? '没有匹配的订阅' : panel.key === 'remove' ? '暂无已关联订阅' : '全部已关联'}</p>}</div>
          <div className="subscription-assignment-footer">{panel.key === 'remove' ? <Popconfirm title={'解除这 ' + selectedRemove.length + ' 份订阅的关联？'} description="保留订阅及其他节点，仅清理不再被订阅使用的账号。" onConfirm={() => submit('remove')} disabled={!listReady || !selectedRemove.length || !!saving || selectedRemove.length > 200} okText="解除关联" cancelText="取消"><Button danger loading={saving === 'remove'} disabled={!listReady || !!saving || !selectedRemove.length || selectedRemove.length > 200}>解除关联（已选 {selectedRemove.length} 个）</Button></Popconfirm> : <Button type="primary" loading={saving === 'add'} disabled={!listReady || !!saving || !selectedAdd.length || selectedAdd.length > 200} onClick={() => void submit('add')}>确认分配（已选 {selectedAdd.length} 个）</Button>}</div>
        </section>)}
      </div>}
    </Spin>
    <p className="node-help">勾选已有订阅后，系统自动创建或复用账号。每批最多 200 份，保存后自动同步到相关节点。</p>
    {results.length > 0 && <div style={{marginTop:16}}><Space wrap><span>已处理 {results.filter(item => item.status !== 'failed').length} 项</span><span>失败 {results.filter(item => item.status === 'failed').length} 项</span></Space><Table size="small" rowKey={item => item.id || item.name} pagination={{pageSize:5}} dataSource={results} columns={[{title:'订阅',dataIndex:'name'}, {title:'结果',dataIndex:'status',render:(status:BatchSubscriptionResult['status']) => <Tag color={status === 'failed' ? 'red' : undefined}>{statusLabels[status]}</Tag>}, {title:'说明',dataIndex:'error'}]} /></div>}
  </Modal>
}

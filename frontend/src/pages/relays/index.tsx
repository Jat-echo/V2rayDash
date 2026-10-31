import {useCallback, useEffect, useState} from 'react'
import {Alert, Button, Drawer, Empty, Form, Input, Popconfirm, Select, Space, Table, message} from 'antd'
import {Link} from 'react-router-dom'
import {DeleteOutlined, EditOutlined, PlusOutlined, ReloadOutlined, TeamOutlined, UploadOutlined} from '@ant-design/icons'
import {CombinationRequest, RelayCombination, Server, relayAPI, serverAPI} from '../../services/api'
import {apiError, NodeState} from '../../components/NodeActions'
import BatchSubscriptionDrawer from '../../components/BatchSubscriptionDrawer'
import {FlagName} from '../../components/FlagName'
import {combinationPath, serverRole} from '../../utils/node-targets'
import './relays.css'

export default function RelayPage() {
  const [servers, setServers] = useState<Server[]>([])
  const [items, setItems] = useState<RelayCombination[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [open, setOpen] = useState(false)
  const [editing, setEditing] = useState<RelayCombination | null>(null)
  const [saving, setSaving] = useState(false)
  const [filter, setFilter] = useState('')
  const [batchTarget, setBatchTarget] = useState<string>()
  const [form] = Form.useForm()
  const load = useCallback(async () => {
    try {const [nodes, combinations] = await Promise.all([serverAPI.list(), relayAPI.list()]); setServers(nodes); setItems(combinations); setError('')}
    catch (err) {setError(apiError(err, '加载代理链路失败，请重试'))}
    finally {setLoading(false)}
  }, [])
  useEffect(() => {void load()}, [load])
  const publishing = servers.some(server => server.node?.state === 'publishing' || server.node?.state === 'pending')
  useEffect(() => {
    if (!publishing) return
    const timer = window.setInterval(() => {void load()}, 2000)
    return () => window.clearInterval(timer)
  }, [publishing, load])
  const relays = servers.filter(server => serverRole(server) === 'relay')
  const exits = servers.filter(server => serverRole(server) === 'exit')
  const edit = (item?: RelayCombination) => {
    setEditing(item || null); form.resetFields(); form.setFieldsValue(item || {relay_server_id:relays[0]?.id}); setOpen(true)
  }
  const save = async (values: CombinationRequest) => {
    setSaving(true)
    const relay = servers.find(server => server.id === values.relay_server_id)
    const exit = servers.find(server => server.id === values.exit_server_id)
    const data = {...values, name: values.name?.trim() || `${relay?.name} → ${exit?.name}`}
    try {if (editing) await relayAPI.update(editing.id, data); else await relayAPI.create(data); message.success('链路已保存，可在订阅中选择或批量分配'); setOpen(false); await load()}
    catch (err) {message.error(apiError(err, '保存链路失败'))}
    finally {setSaving(false)}
  }
  const remove = async (item: RelayCombination) => {
    try {await relayAPI.delete(item.id); message.success('链路已删除'); await load()}
    catch (err) {message.error(apiError(err, '删除链路失败'))}
  }
  const publish = async (item: RelayCombination) => {
    try {await relayAPI.publish(item.relay_server_id); message.info('正在同步该中转上的账号与落地配置'); await load()}
    catch (err) {message.error(apiError(err, '无法开始同步'))}
  }
  return <div className="relay-page">
    <header className="relay-heading"><div><h1>代理链路</h1><p>手动创建“中转 → 落地”链路，再分配给用户订阅。</p></div><Space><Button icon={<ReloadOutlined />} onClick={() => {setLoading(true); void load()}}>刷新</Button><Button type="primary" icon={<PlusOutlined />} disabled={!relays.length || !exits.length} onClick={() => edit()}>创建链路</Button></Space></header>
    {error && <Alert type="error" showIcon message={error} action={<Button onClick={() => void load()}>重试</Button>} />}
    {!loading && (!relays.length || !exits.length) && <Alert type="info" showIcon message="先区分节点角色" description={<span>在 <Link to="/servers">节点管理</Link> 中添加或编辑节点，设置为“落地节点”或“中转节点”。落地仍支持原有直连。</span>} />}
    <div className="relay-toolbar"><span>一个中转节点可连接多个落地节点，同一个落地节点也可被多个中转节点使用。</span><Select aria-label="筛选中转节点" style={{minWidth:200}} value={filter} onChange={setFilter} options={[{value:'',label:'全部中转节点'},...relays.map(server => ({value:server.id,label:server.name}))]} /></div>
    <section className="relay-node" aria-label="已创建的代理链路">
      <Table<RelayCombination> rowKey="id" loading={loading} dataSource={items.filter(item => !filter || item.relay_server_id === filter)} pagination={{pageSize:10}} scroll={{x:1000}} locale={{emptyText:<Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="还没有链路，点击“创建链路”手动添加，例如中转 A → 落地 a。" />}} columns={[
        {title:'链路',width:235,render:(_,item) => <div><strong>{item.name}</strong><div className="node-help">{combinationPath(item)}</div></div>},
        {title:'中转节点',width:220,render:(_,item) => {const server=servers.find(value => value.id===item.relay_server_id); return <div><div className="relay-server-heading"><FlagName name={item.relay_server_name} countryCode={server?.country_code} /><NodeState node={server?.node} /></div><div className="node-help">{server?.ip}</div></div>}},
        {title:'落地节点',width:220,render:(_,item) => {const server=servers.find(value => value.id===item.exit_server_id); return <div><div className="relay-server-heading"><FlagName name={item.exit_server_name} countryCode={server?.country_code} /><NodeState node={server?.node} /></div><div className="node-help">{server?.ip}</div></div>}},
        {title:'使用情况',width:110,render:(_,item) => <div>{item.subscription_count} 份订阅<div className="node-help">{item.account_count} 个账号</div></div>},
        {title:'操作',width:160,render:(_,item) => {
          const relay=servers.find(value => value.id===item.relay_server_id), exit=servers.find(value => value.id===item.exit_server_id)
          const busy=relay?.node?.state==='publishing' || exit?.node?.state==='publishing'
          const syncLabel=relay?.node?.state==='failed' || exit?.node?.state==='failed' ? '重试同步' : '立即同步'
          return <Space size={4} wrap><Button size="small" icon={<TeamOutlined />} aria-label="关联订阅" title="关联订阅" onClick={() => setBatchTarget(`relay:${item.id}`)} /><Button size="small" icon={<EditOutlined />} aria-label="编辑链路" title="编辑链路" onClick={() => edit(item)} /><Button size="small" icon={<UploadOutlined />} aria-label={syncLabel} title={syncLabel} disabled={!relay?.node || !exit?.node || busy} loading={relay?.node?.state==='publishing'} onClick={() => void publish(item)} /><Popconfirm title="删除这条代理链路？" description="已有账号使用的链路会保留，需要先移除账号。" onConfirm={() => remove(item)}><Button size="small" icon={<DeleteOutlined />} aria-label="删除链路" title="删除链路" danger disabled={item.account_count>0 || busy} /></Popconfirm></Space>
        }},
      ]} />
    </section>
    <p className="node-help relay-footnote">订阅可同时包含直连节点和多条代理链路。配置变更保存后自动同步，连续操作会合并处理；也可立即同步，失败后使用同步图标重试。</p>
    <Drawer title={editing ? '编辑代理链路' : '创建代理链路'} open={open} onClose={() => setOpen(false)} width={480}>
      <p className="relay-intro">选择中转与落地，系统自动生成连接配置。请先在节点的“服务配置”中读取落地配置。</p>
      <Form form={form} layout="vertical" onFinish={save}>
        <Form.Item name="name" label="链路名称" rules={[{max:100}]}><Input placeholder="可留空，使用“中转 → 落地”命名" maxLength={100} /></Form.Item>
        <Form.Item name="relay_server_id" label="中转节点" rules={[{required:true,message:'选择中转节点'}]}><Select options={relays.map(server => ({value:server.id,label:`${server.name} · ${server.ip}`}))} disabled={!!editing?.account_count} /></Form.Item>
        <Form.Item name="exit_server_id" label="落地节点" rules={[{required:true,message:'选择落地节点'}]}><Select options={exits.map(server => ({value:server.id,label:`${server.name} · ${server.ip}${!server.node ? '（请先读取配置）' : ''}`,disabled:!server.node}))} disabled={!!editing?.account_count} /></Form.Item>
        {!!editing?.account_count && <p className="node-help">链路已有账号，可修改名称。如需更换节点，请创建另一链路，再为订阅添加该节点。</p>}
        <Button type="primary" htmlType="submit" block loading={saving}>保存链路</Button>
      </Form>
    </Drawer>
    <BatchSubscriptionDrawer open={!!batchTarget} target={batchTarget} onClose={() => setBatchTarget(undefined)} onChange={() => void load()} />
  </div>
}

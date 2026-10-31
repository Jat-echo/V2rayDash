import { useState } from 'react'
import { Alert, Button, Descriptions, Drawer, Form, Select, Tooltip, message } from 'antd'
import { CheckCircleOutlined, ClockCircleOutlined, CloseCircleOutlined, FileTextOutlined, QuestionCircleOutlined, SettingOutlined, SyncOutlined } from '@ant-design/icons'
import { relayAPI, ManagedNode, Server } from '../services/api'
import './NodeActions.css'

export const nodeStateLabels: Record<ManagedNode['state'], string> = {
  adopted: '已读取', pending: '等待自动同步', publishing: '同步中', active: '已生效', failed: '同步失败',
}

export function NodeState({ node }: { node?: ManagedNode }) {
  const state = node?.state || 'unread'
  const label = node ? nodeStateLabels[node.state] : '未读取配置'
  const Icon = {unread: QuestionCircleOutlined, adopted: FileTextOutlined, pending: ClockCircleOutlined, publishing: SyncOutlined, active: CheckCircleOutlined, failed: CloseCircleOutlined}[state]
  return <Tooltip title={state === 'failed' && node?.last_error ? `${label}：${node.last_error}` : label} trigger={['hover', 'focus']}><span className={`node-state node-state--${state}`} role="img" aria-label={label} tabIndex={0}><Icon spin={state === 'publishing'} aria-hidden /></span></Tooltip>
}

export function apiError(error: unknown, fallback: string) {
  return (error as { response?: { data?: { error?: string } } })?.response?.data?.error || fallback
}

export default function NodeActions({ server, onChange }: { server: Server; onChange: () => void }) {
  const [open, setOpen] = useState(false)
  const [layout, setLayout] = useState(server.node?.layout || '')
  const [busy, setBusy] = useState<'adopt' | 'monitor' | null>(null)
  const publishing = server.node?.state === 'publishing'
  const adopt = async () => {
    setBusy('adopt')
    try {
      await relayAPI.adopt(server.id, layout)
      message.success('已导入现有服务和账号，远程配置保持不变')
      setOpen(false); onChange()
    } catch (error) { message.error(apiError(error, '读取配置失败，请检查节点认证信息')) }
    finally { setBusy(null) }
  }
  const monitor = async () => {
    setBusy('monitor')
    try { await relayAPI.monitor(server.id); message.success('监控已接入，约一分钟后可查看节点指标'); onChange() }
    catch (error) { message.error(apiError(error, '监控接入失败')) }
    finally { setBusy(null) }
  }
  return <>
    <Button size="small" icon={<SettingOutlined />} aria-label="服务配置" title="服务配置" onClick={event => {event.stopPropagation();setOpen(true)}} disabled={publishing || busy !== null} />
    <Drawer title={`服务配置 · ${server.name}`} open={open} onClose={() => setOpen(false)} width={480}>
      <Alert type="info" showIcon message="读取已有配置" description="已手工部署的节点首次使用中转管理时，需要先读取现有端口、账号和 Reality 参数。读取不会重装服务；后续配置变更保存后自动同步，也可使用列表中的“立即同步”或“重试同步”。" />
      <Descriptions column={1} style={{ margin: '24px 0' }} items={[
        { key: 'ip', label: '节点', children: server.ip },
        { key: 'role', label: '当前角色', children: server.role === 'relay' ? '中转节点' : '落地节点' },
        { key: 'state', label: '配置状态', children: <NodeState node={server.node} /> },
        ...(server.node ? [{ key: 'updated', label: '更新时间', children: new Date(server.node.updated_at).toLocaleString() }] : []),
      ]} />
      {server.node?.last_error && <Alert style={{ marginBottom: 20 }} type="error" showIcon message="上次操作失败" description={server.node.last_error} />}
      {server.node && <div style={{ marginBottom: 24 }}><Button loading={busy === 'monitor'} disabled={busy !== null && busy !== 'monitor'} onClick={monitor}>安装 / 更新监控 Agent</Button><p className="node-help">只安装监控，不重装代理服务。同步配置不会安装 Agent。控制中心公网地址需可通过 HTTPS 访问。</p></div>}
      <Form layout="vertical" onFinish={adopt}>
        <Form.Item label="已有服务类型">
          <Select value={layout} onChange={setLayout} options={[
            { value: '', label: '自动识别' }, { value: 'relay', label: '中转服务 · xray-relay' },
            { value: 'exit', label: '落地服务 · xray-exit' }, { value: 'legacy', label: '现有 v2ray-agent 服务' },
          ]} />
        </Form.Item>
        <Button type="primary" htmlType="submit" loading={busy === 'adopt'} block>{server.node ? '重新读取配置' : '读取现有配置'}</Button>
      </Form>
      <p className="node-help">先读取落地配置，再读取中转配置，可识别已有账号的落地绑定。仅需在首次接入或节点配置被手工修改后读取。</p>
    </Drawer>
  </>
}

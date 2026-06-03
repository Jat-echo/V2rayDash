import { useState, useEffect, useCallback } from 'react'
import { Card, Form, Input, Button, message } from 'antd'
import { AimOutlined, SaveOutlined, ReloadOutlined, LockOutlined } from '@ant-design/icons'
import { authAPI } from '../../services/api'

interface SystemStatus {
  cpu_percent: number
  mem_percent: number
  mem_used_mb: number
  mem_total_mb: number
  disk_percent: number
  disk_used_gb: number
  disk_total_gb: number
}

// 所有 fetch 都带上 auth token（设置页的接口受认证保护）
function authedFetch(url: string, opts?: RequestInit) {
  const token = localStorage.getItem('admin_token') || ''
  return fetch(url, {
    ...opts,
    headers: { Authorization: `Bearer ${token}`, ...opts?.headers },
  })
}

// ── 环形指标卡 ────────────────────────────────────────────────────────────────
function MetricRing({
  label, value, sub, color,
}: { label: string; value: number; sub?: string; color: string }) {
  const R = 40
  const C = 2 * Math.PI * R   // 251.3
  const ARC = C * 0.75         // 188.5  (270°)
  const pct = Math.min(value / 100, 1)
  const filled = pct * ARC
  // 根据负载调整颜色
  const fill = pct > 0.85 ? 'var(--morandi-terracotta)'
              : pct > 0.68 ? 'var(--morandi-sand)'
              : color

  return (
    <div className="st-ring-card">
      <svg width="104" height="104" viewBox="0 0 104 104">
        {/* 轨道 */}
        <circle cx="52" cy="52" r={R} fill="none"
          stroke="rgba(60,55,48,0.10)" strokeWidth="6" strokeLinecap="round"
          strokeDasharray={`${ARC} ${C - ARC}`}
          transform="rotate(135 52 52)"
        />
        {/* 值 */}
        <circle cx="52" cy="52" r={R} fill="none"
          stroke={fill} strokeWidth="6" strokeLinecap="round"
          strokeDasharray={`${filled} ${C - filled}`}
          transform="rotate(135 52 52)"
          style={{ transition: 'stroke-dasharray 1s cubic-bezier(.4,0,.2,1), stroke .4s' }}
        />
        {/* 数值文字 */}
        <text x="52" y="48" textAnchor="middle"
          fill="var(--text-primary)" fontSize="16" fontWeight="600">
          {value.toFixed(1)}
        </text>
        <text x="52" y="62" textAnchor="middle"
          fill="var(--text-muted)" fontSize="10">
          %
        </text>
      </svg>
      <p className="st-ring-label">{label}</p>
      {sub && <p className="st-ring-sub">{sub}</p>}
    </div>
  )
}

// ── 主组件 ─────────────────────────────────────────────────────────────────────
export default function Settings() {
  const [urlForm] = Form.useForm()
  const [pwdForm] = Form.useForm()
  const [saving, setSaving] = useState(false)
  const [fetchingIP, setFetchingIP] = useState(false)
  const [modified, setModified] = useState(false)
  const [pwdSaving, setPwdSaving] = useState(false)
  const [sys, setSys] = useState<SystemStatus | null>(null)
  const [sysLoading, setSysLoading] = useState(false)

  const fetchSys = useCallback(async () => {
    setSysLoading(true)
    try {
      const r = await authedFetch('/api/settings/system-status')
      if (r.ok) setSys(await r.json())
    } catch {}
    finally { setSysLoading(false) }
  }, [])

  useEffect(() => {
    ;(async () => {
      try {
        const r = await authedFetch('/api/settings/public-url')
        const d = await r.json()
        urlForm.setFieldsValue({ public_url: d.value || 'http://localhost:8080' })
      } catch {}
    })()
    fetchSys()
    const t = setInterval(fetchSys, 10000)
    return () => clearInterval(t)
  }, [fetchSys, urlForm])

  const handleGetIP = async () => {
    setFetchingIP(true)
    try {
      const r = await authedFetch('/api/settings/public-ip')
      const d = await r.json()
      if (d.ip) {
        urlForm.setFieldsValue({ public_url: `http://${d.ip}:8080` })
        setModified(true)
        message.success(`检测到公网 IP：${d.ip}`)
      } else message.error('获取公网IP失败')
    } catch { message.error('网络错误') }
    finally { setFetchingIP(false) }
  }

  const handleSaveUrl = async (v: { public_url: string }) => {
    setSaving(true)
    try {
      const r = await authedFetch('/api/settings/public-url', {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ value: v.public_url }),
      })
      if (r.ok) { message.success('已保存'); setModified(false) }
      else message.error('保存失败')
    } catch { message.error('网络错误') }
    finally { setSaving(false) }
  }

  const handleChangePwd = async (v: { old: string; next: string; confirm: string }) => {
    if (v.next !== v.confirm) { message.error('两次密码不一致'); return }
    if (v.next.length < 6) { message.error('新密码至少6位'); return }
    setPwdSaving(true)
    try {
      await authAPI.changePassword(v.old, v.next)
      message.success('密码已修改，即将退出登录')
      pwdForm.resetFields()
      setTimeout(() => { localStorage.removeItem('admin_token'); window.location.reload() }, 1500)
    } catch (e: any) {
      message.error(e?.response?.data?.error || '修改失败')
    } finally { setPwdSaving(false) }
  }

  const memSub = sys
    ? `${sys.mem_used_mb >= 1024 ? (sys.mem_used_mb / 1024).toFixed(1) + ' GB' : sys.mem_used_mb + ' MB'} / ${sys.mem_total_mb >= 1024 ? (sys.mem_total_mb / 1024).toFixed(1) + ' GB' : sys.mem_total_mb + ' MB'}`
    : ''
  const diskSub = sys ? `${sys.disk_used_gb.toFixed(1)} GB / ${sys.disk_total_gb.toFixed(1)} GB` : ''

  return (
    <div className="animate-in">
      {/* 页头：与其他页面统一 */}
      <div className="page-header">
        <h1>系统设置</h1>
        <p>配置控制中心访问地址、查看服务器资源状态</p>
      </div>

      {/* ── 访问地址 ── */}
      <Card className="morandi-card" style={{ maxWidth: 680, marginBottom: 20 }}>
        <p className="st-section-title">控制中心访问地址</p>
        <p className="st-section-desc">代理节点通过此地址进行心跳上报和配置同步</p>
        <Form
          form={urlForm}
          layout="vertical"
          onFinish={handleSaveUrl}
          onValuesChange={() => setModified(true)}
          style={{ marginTop: 20 }}
        >
          <Form.Item
            name="public_url"
            label="公网地址"
            rules={[
              { required: true, message: '请输入地址' },
              { pattern: /^https?:\/\/.+/, message: '须以 http:// 或 https:// 开头' },
            ]}
          >
            <Input
              size="large"
              placeholder="http://公网IP:8080 或 https://域名"
              suffix={modified ? <span style={{ color: 'var(--morandi-dusty-rose)', fontSize: 12 }}>已修改</span> : null}
            />
          </Form.Item>
          <div style={{ fontSize: 13, color: 'var(--text-muted)', marginBottom: 20 }}>
            支持 <code style={{ background: 'var(--bg-secondary)', padding: '1px 5px', borderRadius: 4 }}>http://IP:PORT</code> 或 <code style={{ background: 'var(--bg-secondary)', padding: '1px 5px', borderRadius: 4 }}>https://域名</code>
          </div>
          <div style={{ display: 'flex', gap: 10, justifyContent: 'flex-end' }}>
            <Button icon={<AimOutlined />} onClick={handleGetIP} loading={fetchingIP}>
              一键获取公网 IP
            </Button>
            <Button
              type="primary" htmlType="submit"
              icon={<SaveOutlined />} loading={saving}
              disabled={!modified}
            >
              保存配置
            </Button>
          </div>
        </Form>
      </Card>

      {/* ── 服务器资源 ── */}
      <Card
        className="morandi-card"
        style={{ maxWidth: 680, marginBottom: 20 }}
        title={
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
            <span>控制中心服务器资源</span>
            <Button
              type="text" size="small"
              icon={<ReloadOutlined spin={sysLoading} />}
              onClick={fetchSys}
            />
          </div>
        }
      >
        {sys ? (
          <div className="st-rings">
            <MetricRing
              label="CPU 使用率"
              value={sys.cpu_percent}
              color="var(--morandi-dusty-rose)"
            />
            <div className="st-ring-divider" />
            <MetricRing
              label="内存使用率"
              value={sys.mem_percent}
              sub={memSub}
              color="var(--morandi-sky)"
            />
            <div className="st-ring-divider" />
            <MetricRing
              label="磁盘使用率"
              value={sys.disk_percent}
              sub={diskSub}
              color="var(--morandi-sage)"
            />
          </div>
        ) : (
          <div style={{ textAlign: 'center', padding: '32px 0', color: 'var(--text-muted)', fontSize: 14 }}>
            {sysLoading ? '加载中…' : '暂无数据，请刷新重试'}
          </div>
        )}
      </Card>

      {/* ── 修改密码 ── */}
      <Card className="morandi-card" style={{ maxWidth: 680 }}>
        <p className="st-section-title">修改登录密码</p>
        <p className="st-section-desc">修改后需要重新登录</p>
        <Form
          form={pwdForm}
          layout="vertical"
          onFinish={handleChangePwd}
          style={{ marginTop: 20 }}
        >
          <Form.Item name="old" label="原密码" rules={[{ required: true }]}>
            <Input.Password prefix={<LockOutlined />} placeholder="当前密码" />
          </Form.Item>
          <Form.Item name="next" label="新密码" rules={[{ required: true, min: 6, message: '至少6个字符' }]}>
            <Input.Password prefix={<LockOutlined />} placeholder="至少6个字符" />
          </Form.Item>
          <Form.Item name="confirm" label="确认新密码" rules={[{ required: true }]}>
            <Input.Password prefix={<LockOutlined />} placeholder="再次输入新密码" />
          </Form.Item>
          <div style={{ display: 'flex', justifyContent: 'flex-end' }}>
            <Button type="primary" htmlType="submit" loading={pwdSaving}>
              确认修改密码
            </Button>
          </div>
        </Form>
      </Card>

      <style>{`
        .st-section-title {
          font-size: 15px;
          font-weight: 600;
          color: var(--text-primary);
          margin: 0 0 4px;
        }
        .st-section-desc {
          font-size: 13px;
          color: var(--text-muted);
          margin: 0;
        }

        /* ── 环形区域 ── */
        .st-rings {
          display: flex;
          align-items: center;
          justify-content: center;
          padding: 12px 0 8px;
          gap: 0;
        }
        .st-ring-card {
          flex: 1;
          display: flex;
          flex-direction: column;
          align-items: center;
          gap: 6px;
          padding: 0 8px;
        }
        .st-ring-divider {
          width: 1px;
          height: 72px;
          background: var(--border-color);
          flex-shrink: 0;
        }
        .st-ring-label {
          font-size: 12px;
          color: var(--text-secondary);
          margin: 0;
          text-align: center;
        }
        .st-ring-sub {
          font-size: 11px;
          color: var(--text-muted);
          margin: 0;
          text-align: center;
        }
      `}</style>
    </div>
  )
}

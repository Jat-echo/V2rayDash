import { BrowserRouter, Routes, Route, Navigate, NavLink } from 'react-router-dom'
import { Layout, Spin, Dropdown, Avatar } from 'antd'
import { useState, useEffect, lazy, Suspense } from 'react'
import { UserOutlined, LogoutOutlined } from '@ant-design/icons'
import LoginPage from './pages/login'
import SetupPage from './pages/setup'
import { authAPI } from './services/api'

const ServerList = lazy(() => import('./pages/servers'))
const SubscriptionList = lazy(() => import('./pages/subscriptions'))
const Logs = lazy(() => import('./pages/logs'))
const Settings = lazy(() => import('./pages/settings'))
const ClientDownload = lazy(() => import('./pages/clients'))

const { Content } = Layout

const ServerIcon = () => (
  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
    <rect x="2" y="3" width="20" height="6" rx="1"/>
    <rect x="2" y="13" width="20" height="6" rx="1"/>
    <circle cx="6" cy="5" r="1" fill="currentColor"/>
    <circle cx="6" cy="15" r="1" fill="currentColor"/>
  </svg>
)

const SubIcon = () => (
  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
    <path d="M4 4h16v16H4z"/>
    <path d="M4 9h16M9 4v16"/>
  </svg>
)

const LogIcon = () => (
  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
    <path d="M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8z"/>
    <path d="M14 2v6h6M8 13h8M8 17h8M8 9h2"/>
  </svg>
)

const SettingsIcon = () => (
  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
    <circle cx="12" cy="12" r="3"/>
    <path d="M12 1v2M12 21v2M4.22 4.22l1.42 1.42M18.36 18.36l1.42 1.42M1 12h2M21 12h2M4.22 19.78l1.42-1.42M18.36 5.64l1.42-1.42"/>
  </svg>
)

const ClientIcon = () => (
  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
    <rect x="2" y="3" width="20" height="14" rx="2"/>
    <path d="M8 21h8M12 17v4"/>
    <path d="M9 10l2 2 4-4"/>
  </svg>
)

const navItems = [
  { path: '/servers', label: '服务器', icon: <ServerIcon /> },
  { path: '/subscriptions', label: '订阅', icon: <SubIcon /> },
  { path: '/logs', label: '日志', icon: <LogIcon /> },
  { path: '/clients', label: '客户端', icon: <ClientIcon /> },
  { path: '/settings', label: '设置', icon: <SettingsIcon /> },
]

type AuthState = 'loading' | 'setup' | 'login' | 'ok'

function App() {
  const [collapsed, setCollapsed] = useState(false)
  const [authState, setAuthState] = useState<AuthState>('loading')
  const [username, setUsername] = useState('')

  useEffect(() => {
    checkAuth()
  }, [])

  const checkAuth = async () => {
    try {
      const status = await authAPI.status()
      if (status.setup_required) {
        setAuthState('setup')
        return
      }
      const token = localStorage.getItem('admin_token')
      const storedUser = localStorage.getItem('admin_username')
      if (!token) {
        setAuthState('login')
        return
      }
      // 验证 token 是否仍然有效
      const res = await fetch('/api/servers', { headers: { Authorization: `Bearer ${token}` } })
      if (res.status === 401) {
        localStorage.removeItem('admin_token')
        localStorage.removeItem('admin_username')
        setAuthState('login')
        return
      }
      setUsername(storedUser || '')
      setAuthState('ok')
    } catch {
      setAuthState('login')
    }
  }

  const handleLogin = (user: string) => {
    setUsername(user)
    setAuthState('ok')
  }

  const handleLogout = async () => {
    try { await authAPI.logout() } catch {}
    localStorage.removeItem('admin_token')
    localStorage.removeItem('admin_username')
    setAuthState('login')
  }

  if (authState === 'loading') {
    return (
      <div style={{ minHeight: '100vh', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
        <Spin size="large" />
      </div>
    )
  }

  if (authState === 'setup') return <SetupPage onComplete={handleLogin} />
  if (authState === 'login') return <LoginPage onLogin={handleLogin} />

  return (
    <BrowserRouter>
      <Layout className="app-layout">
        <Layout.Sider
          width={240}
          collapsedWidth={72}
          collapsed={collapsed}
          className="app-sidebar"
          style={{
            background: 'var(--bg-sidebar)',
            position: 'fixed',
            height: '100vh',
            left: 0,
            top: 0,
            zIndex: 100,
          }}
        >
          <div className="app-logo" style={{ display: 'flex', alignItems: 'center', gap: collapsed ? 0 : 10, justifyContent: collapsed ? 'center' : 'flex-start' }}>
            <img src="/favicon.svg" alt="logo" style={{ width: 28, height: 28, flexShrink: 0, borderRadius: 6 }} />
            {!collapsed && <h1 style={{ margin: 0 }}>V2ray<span>Dash</span></h1>}
          </div>
          <nav className="app-nav">
            {navItems.map(item => (
              <NavLink
                key={item.path}
                to={item.path}
                className={({ isActive }) => isActive ? 'active' : ''}
              >
                {item.icon}
                {!collapsed && <span>{item.label}</span>}
              </NavLink>
            ))}
          </nav>
          {/* 用户信息 + 退出 */}
          <div style={{
            position: 'absolute', bottom: 0, left: 0, right: 0,
            padding: collapsed ? '12px 0' : '12px 16px',
            borderTop: '1px solid rgba(255,255,255,0.08)',
            display: 'flex', alignItems: 'center', gap: 10,
            justifyContent: collapsed ? 'center' : 'space-between',
          }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: 8, overflow: 'hidden' }}>
              <Avatar size={32} icon={<UserOutlined />} style={{ background: '#c9a9a6', flexShrink: 0 }} />
              {!collapsed && <span style={{ color: 'rgba(255,255,255,0.85)', fontSize: 13, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{username}</span>}
            </div>
            {!collapsed && (
              <LogoutOutlined
                onClick={handleLogout}
                style={{ color: 'rgba(255,255,255,0.5)', cursor: 'pointer', fontSize: 16, flexShrink: 0 }}
                title="退出登录"
              />
            )}
          </div>
        </Layout.Sider>

        <Layout className="app-main" style={{ marginLeft: collapsed ? 72 : 240 }}>
          <Content className="app-content" style={{ padding: '28px 32px' }}>
            <Suspense fallback={<div style={{ display: 'flex', justifyContent: 'center', padding: 60 }}><Spin size="large" /></div>}>
              <Routes>
                <Route path="/" element={<Navigate to="/servers" replace />} />
                <Route path="/servers" element={<ServerList />} />
                <Route path="/subscriptions" element={<SubscriptionList />} />
                <Route path="/logs" element={<Logs />} />
                <Route path="/clients" element={<ClientDownload />} />
                <Route path="/settings" element={<Settings />} />
              </Routes>
            </Suspense>
          </Content>
        </Layout>
      </Layout>
    </BrowserRouter>
  )
}

export default App

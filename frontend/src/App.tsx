import { BrowserRouter, Routes, Route, Navigate, NavLink } from 'react-router-dom'
import { Layout } from 'antd'
import { useState } from 'react'
import ServerList from './pages/servers'
import SubscriptionList from './pages/subscriptions'
import Monitor from './pages/monitor'
import Logs from './pages/logs'
import TemplateList from './pages/templates'

const { Content } = Layout

// Icons as inline SVGs for consistency
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

const MonitorIcon = () => (
  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
    <path d="M3 12h4l3-9 4 18 3-9h4"/>
  </svg>
)

const LogIcon = () => (
  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
    <path d="M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8z"/>
    <path d="M14 2v6h6M8 13h8M8 17h8M8 9h2"/>
  </svg>
)

const TemplateIcon = () => (
  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
    <rect x="3" y="3" width="18" height="18" rx="2"/>
    <path d="M3 9h18M9 21V9"/>
  </svg>
)

const navItems = [
  { path: '/servers', label: '服务器', icon: <ServerIcon /> },
  { path: '/subscriptions', label: '订阅', icon: <SubIcon /> },
  { path: '/templates', label: '模板', icon: <TemplateIcon /> },
  { path: '/monitor', label: '监控', icon: <MonitorIcon /> },
  { path: '/logs', label: '日志', icon: <LogIcon /> },
]

function App() {
  const [collapsed, setCollapsed] = useState(false)

  return (
    <BrowserRouter>
      <Layout className="app-layout">
        {/* Sidebar */}
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
          <div className="app-logo">
            <h1>V2<span>Dash</span></h1>
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
        </Layout.Sider>

        {/* Main Content */}
        <Layout className="app-main" style={{ marginLeft: collapsed ? 72 : 240 }}>
          <Content className="app-content" style={{ padding: '28px 32px' }}>
            <Routes>
              <Route path="/" element={<Navigate to="/servers" replace />} />
              <Route path="/servers" element={<ServerList />} />
              <Route path="/subscriptions" element={<SubscriptionList />} />
              <Route path="/monitor" element={<Monitor />} />
              <Route path="/logs" element={<Logs />} />
              <Route path="/templates" element={<TemplateList />} />
            </Routes>
          </Content>
        </Layout>
      </Layout>
    </BrowserRouter>
  )
}

export default App
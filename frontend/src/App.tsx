import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { Layout } from 'antd'
import ServerList from './pages/servers'
import SubscriptionList from './pages/subscriptions'
import Monitor from './pages/monitor'
import Logs from './pages/logs'

const { Header, Content } = Layout

function App() {
  return (
    <BrowserRouter>
      <Layout style={{ minHeight: '100vh' }}>
        <Header style={{ color: '#fff', fontSize: '18px', padding: '0 24px' }}>
          V2ray 管理平台
        </Header>
        <Content style={{ padding: '24px' }}>
          <Routes>
            <Route path="/" element={<Navigate to="/servers" replace />} />
            <Route path="/servers" element={<ServerList />} />
            <Route path="/subscriptions" element={<SubscriptionList />} />
            <Route path="/monitor" element={<Monitor />} />
            <Route path="/logs" element={<Logs />} />
          </Routes>
        </Content>
      </Layout>
    </BrowserRouter>
  )
}

export default App
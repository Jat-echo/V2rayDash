import { useState } from 'react'
import { Form, Input, Button, message } from 'antd'
import { UserOutlined, LockOutlined } from '@ant-design/icons'
import { authAPI } from '../../services/api'

interface Props {
  onLogin: (username: string) => void
}

export default function LoginPage({ onLogin }: Props) {
  const [loading, setLoading] = useState(false)
  const [form] = Form.useForm()

  const handleLogin = async (values: { username: string; password: string }) => {
    setLoading(true)
    try {
      const res = await authAPI.login(values.username, values.password)
      if (res.token) {
        localStorage.setItem('admin_token', res.token)
        localStorage.setItem('admin_username', res.username)
        onLogin(res.username)
      } else {
        message.error(res.error || '登录失败')
      }
    } catch {
      message.error('登录失败，请检查网络')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div style={{
      minHeight: '100vh', display: 'flex', alignItems: 'center', justifyContent: 'center',
      background: 'linear-gradient(135deg, #f5f0e8 0%, #ede8e0 100%)',
    }}>
      <div style={{
        background: '#fff', borderRadius: 16, padding: '48px 40px', width: 380,
        boxShadow: '0 8px 32px rgba(60,55,48,0.12)',
      }}>
        <div style={{ textAlign: 'center', marginBottom: 36 }}>
          <h1 style={{ fontSize: 28, fontWeight: 700, color: '#3d3a36', margin: 0 }}>
            V2<span style={{ color: '#c9a9a6' }}>Dash</span>
          </h1>
          <p style={{ color: '#9e9a93', marginTop: 8, fontSize: 14 }}>请登录以继续</p>
        </div>
        <Form form={form} onFinish={handleLogin} layout="vertical" size="large">
          <Form.Item name="username" rules={[{ required: true, message: '请输入用户名' }]}>
            <Input prefix={<UserOutlined style={{ color: '#c9a9a6' }} />} placeholder="用户名" style={{ borderRadius: 10 }} />
          </Form.Item>
          <Form.Item name="password" rules={[{ required: true, message: '请输入密码' }]}>
            <Input.Password prefix={<LockOutlined style={{ color: '#c9a9a6' }} />} placeholder="密码" style={{ borderRadius: 10 }} />
          </Form.Item>
          <Button type="primary" htmlType="submit" block loading={loading} style={{ borderRadius: 10, height: 44, marginTop: 8 }}>
            登录
          </Button>
        </Form>
      </div>
    </div>
  )
}

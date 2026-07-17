import { useState } from 'react'
import { Form, Input, Button, message } from 'antd'
import { UserOutlined, LockOutlined } from '@ant-design/icons'
import { authAPI } from '../../services/api'

interface Props {
  onComplete: (username: string) => void
}

export default function SetupPage({ onComplete }: Props) {
  const [loading, setLoading] = useState(false)
  const [form] = Form.useForm()

  const handleSetup = async (values: { username: string; password: string; confirm: string }) => {
    if (values.password !== values.confirm) {
      message.error('两次密码不一致')
      return
    }
    setLoading(true)
    try {
      const res = await authAPI.setup(values.username, values.password)
      if (res.token) {
        localStorage.setItem('admin_token', res.token)
        localStorage.setItem('admin_username', res.username)
        message.success('账号创建成功！')
        onComplete(res.username)
      } else {
        message.error(res.error || '创建失败')
      }
    } catch {
      message.error('创建失败，请检查网络')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div style={{
      minHeight: '100vh', display: 'flex', alignItems: 'center', justifyContent: 'center',
      padding: 16,
      background: 'linear-gradient(135deg, #f5f0e8 0%, #ede8e0 100%)',
    }}>
      <div style={{
        background: '#fff', borderRadius: 16, padding: '48px 40px', width: '100%', maxWidth: 420,
        boxShadow: '0 8px 32px rgba(60,55,48,0.12)',
      }}>
        <div style={{ textAlign: 'center', marginBottom: 36 }}>
          <h1 style={{ fontSize: 28, fontWeight: 700, color: '#3d3a36', margin: 0 }}>
            V2<span style={{ color: '#c9a9a6' }}>Dash</span>
          </h1>
          <p style={{ color: 'var(--text-secondary)', marginTop: 8 }}>首次使用，请创建管理员账号</p>
        </div>
        <Form form={form} onFinish={handleSetup} layout="vertical" size="large">
          <Form.Item name="username" label="用户名" rules={[{ required: true, message: '请输入用户名（至少3个字符）', min: 3 }]}>
            <Input prefix={<UserOutlined style={{ color: '#c9a9a6' }} />} placeholder="管理员用户名" style={{ borderRadius: 10 }} />
          </Form.Item>
          <Form.Item name="password" label="密码" rules={[{ required: true, message: '请输入密码（至少6个字符）', min: 6 }]}>
            <Input.Password prefix={<LockOutlined style={{ color: '#c9a9a6' }} />} placeholder="至少6个字符" style={{ borderRadius: 10 }} />
          </Form.Item>
          <Form.Item name="confirm" label="确认密码" rules={[{ required: true, message: '请再次输入密码' }]}>
            <Input.Password prefix={<LockOutlined style={{ color: '#c9a9a6' }} />} placeholder="再次输入密码" style={{ borderRadius: 10 }} />
          </Form.Item>
          <Button type="primary" htmlType="submit" block loading={loading} style={{ borderRadius: 10, height: 44, marginTop: 8 }}>
            创建管理员账号
          </Button>
        </Form>
      </div>
    </div>
  )
}

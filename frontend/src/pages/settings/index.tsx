import { useState, useEffect } from 'react'
import { Card, Form, Input, Button, message, Typography, Space, Tooltip } from 'antd'
import { InfoCircleOutlined, GlobalOutlined, SaveOutlined, AimOutlined } from '@ant-design/icons'

const { Title, Text } = Typography

interface PublicURLSetting {
  value: string
}

export default function Settings() {
  const [loading, setLoading] = useState(false)
  const [fetchingIP, setFetchingIP] = useState(false)
  const [form] = Form.useForm()
  const [hasChanges, setHasChanges] = useState(false)

  useEffect(() => {
    loadSettings()
  }, [])

  const loadSettings = async () => {
    try {
      const res = await fetch('/api/settings/public-url')
      const data: PublicURLSetting = await res.json()
      const url = data.value || 'http://localhost:8080'
      form.setFieldsValue({ public_url: url })
    } catch (e) {
      console.error('Failed to load settings', e)
      form.setFieldsValue({ public_url: 'http://localhost:8080' })
    }
  }

  const getPublicIP = async () => {
    setFetchingIP(true)
    try {
      const res = await fetch('/api/settings/public-ip')
      const data = await res.json()
      if (data.ip) {
        form.setFieldsValue({ public_url: `http://${data.ip}:8080` })
        setHasChanges(true)
        message.success({
          content: (
            <span>
              检测到公网IP: <strong>{data.ip}</strong>
            </span>
          ),
        })
      } else {
        message.error('获取公网IP失败，请检查网络连接')
      }
    } catch (e) {
      message.error('获取公网IP失败')
    } finally {
      setFetchingIP(false)
    }
  }

  const handleSave = async (values: { public_url: string }) => {
    setLoading(true)
    try {
      const res = await fetch('/api/settings/public-url', {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ value: values.public_url }),
      })
      if (res.ok) {
        message.success({
          content: (
            <span>
              保存成功，当前控制中心地址: <strong>{values.public_url}</strong>
            </span>
          ),
        })
        setHasChanges(false)
      } else {
        message.error('保存失败')
      }
    } catch (e) {
      message.error('保存失败')
    } finally {
      setLoading(false)
    }
  }

  const handleValuesChange = () => {
    setHasChanges(true)
  }

  return (
    <div className="animate-in">
      {/* Page Header */}
      <div className="page-header" style={{ marginBottom: 32 }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
          <div className="settings-icon">
            <GlobalOutlined />
          </div>
          <div>
            <h1 style={{ fontSize: 26, fontWeight: 600, marginBottom: 4 }}>系统设置</h1>
            <Text type="secondary" style={{ fontSize: 14 }}>
              配置控制中心的访问地址和基本参数
            </Text>
          </div>
        </div>
      </div>

      {/* Main Settings Card */}
      <Card
        className="morandi-card settings-main-card"
        style={{
          maxWidth: 680,
          borderRadius: 16,
          overflow: 'hidden',
        }}
      >
        {/* Card Header */}
        <div className="settings-card-header">
          <div className="header-content">
            <Title level={5} style={{ margin: 0, color: '#fff' }}>
              控制中心访问地址
            </Title>
            <Text style={{ color: 'rgba(255,255,255,0.75)', fontSize: 13 }}>
              代理节点通过此地址访问控制中心
            </Text>
          </div>
          <div className="header-decoration">
            <svg width="120" height="80" viewBox="0 0 120 80" fill="none">
              <circle cx="100" cy="40" r="60" fill="rgba(255,255,255,0.05)" />
              <circle cx="80" cy="30" r="40" fill="rgba(255,255,255,0.03)" />
            </svg>
          </div>
        </div>

        {/* Card Body */}
        <div className="settings-card-body">
          <Form
            form={form}
            layout="vertical"
            onFinish={handleSave}
            onValuesChange={handleValuesChange}
            requiredMark={false}
          >
            <Form.Item
              name="public_url"
              label={
                <Space>
                  <span>公网访问地址</span>
                  <Tooltip title="代理节点通过此地址访问控制中心进行心跳报告和配置同步">
                    <InfoCircleOutlined style={{ color: '#9e9a93' }} />
                  </Tooltip>
                </Space>
              }
              rules={[
                { required: true, message: '请输入控制中心地址' },
                {
                  pattern: /^https?:\/\/.+/,
                  message: '地址必须以 http:// 或 https:// 开头',
                },
              ]}
              style={{ marginBottom: 24 }}
            >
              <Input
                size="large"
                placeholder="http://公网IP:端口 或 https://域名"
                suffix={
                  hasChanges && (
                    <span style={{ color: '#c9a9a6', fontSize: 12 }}>已修改</span>
                  )
                }
                style={{
                  borderRadius: 10,
                  height: 48,
                  fontSize: 15,
                }}
              />
            </Form.Item>

            {/* Helper Text */}
            <div className="url-helper">
              <InfoCircleOutlined style={{ marginRight: 8, color: '#9e9a93' }} />
              <Text type="secondary" style={{ fontSize: 13 }}>
                格式支持: <code>http://IP:端口</code> 或 <code>https://域名</code>
              </Text>
            </div>

            {/* Action Buttons */}
            <div className="settings-actions">
              <Button
                size="large"
                icon={<AimOutlined />}
                onClick={getPublicIP}
                loading={fetchingIP}
                style={{
                  borderRadius: 10,
                  height: 48,
                  paddingLeft: 24,
                  paddingRight: 24,
                }}
              >
                一键获取公网IP
              </Button>

              <Button
                type="primary"
                size="large"
                htmlType="submit"
                icon={<SaveOutlined />}
                loading={loading}
                disabled={!hasChanges}
                style={{
                  borderRadius: 10,
                  height: 48,
                  paddingLeft: 28,
                  paddingRight: 28,
                }}
              >
                保存设置
              </Button>
            </div>
          </Form>
        </div>
      </Card>

      {/* Info Cards Row */}
      <div className="settings-info-row" style={{ display: 'flex', gap: 20, marginTop: 24 }}>
        {/* Agent Info Card */}
        <Card
          className="morandi-card info-card"
          style={{
            flex: 1,
            maxWidth: 320,
            borderRadius: 12,
          }}
        >
          <div style={{ display: 'flex', alignItems: 'flex-start', gap: 14 }}>
            <div className="info-icon" style={{
              width: 44,
              height: 44,
              borderRadius: 10,
              background: 'linear-gradient(135deg, #B4A7C7 0%, #9A8BB4 100%)',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              color: '#fff',
              fontSize: 18,
            }}>
              📡
            </div>
            <div>
              <Title level={5} style={{ fontSize: 14, marginBottom: 6 }}>
                Agent 配置
              </Title>
              <Text type="secondary" style={{ fontSize: 12, lineHeight: 1.6 }}>
                安装代理后，节点会从此地址获取控制中心配置并进行心跳上报
              </Text>
            </div>
          </div>
        </Card>

        {/* Subscription Info Card */}
        <Card
          className="morandi-card info-card"
          style={{
            flex: 1,
            maxWidth: 320,
            borderRadius: 12,
          }}
        >
          <div style={{ display: 'flex', alignItems: 'flex-start', gap: 14 }}>
            <div className="info-icon" style={{
              width: 44,
              height: 44,
              borderRadius: 10,
              background: 'linear-gradient(135deg, #9DB4C0 0%, #7A9AAE 100%)',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              color: '#fff',
              fontSize: 18,
            }}>
              🔗
            </div>
            <div>
              <Title level={5} style={{ fontSize: 14, marginBottom: 6 }}>
                订阅地址
              </Title>
              <Text type="secondary" style={{ fontSize: 12, lineHeight: 1.6 }}>
                客户端通过此地址获取订阅内容，支持 Clash Meta / VLESS 格式
              </Text>
            </div>
          </div>
        </Card>
      </div>

      {/* Current Status */}
      <Card
        className="morandi-card"
        style={{
          marginTop: 20,
          maxWidth: 680,
          borderRadius: 12,
          background: 'linear-gradient(135deg, #F5F0E8 0%, #F0EDE8 100%)',
        }}
      >
        <div style={{ display: 'flex', alignItems: 'center', gap: 16 }}>
          <div style={{
            width: 48,
            height: 48,
            borderRadius: 12,
            background: 'rgba(168, 181, 160, 0.2)',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
          }}>
            <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="#A8B5A0" strokeWidth="2">
              <path d="M22 11.08V12a10 10 0 11-5.93-9.14" />
              <polyline points="22,4 12,14.01 9,11.01" />
            </svg>
          </div>
          <div>
            <Text strong style={{ fontSize: 14, display: 'block', marginBottom: 4 }}>
              当前配置状态
            </Text>
            <Text type="secondary" style={{ fontSize: 13 }}>
              控制中心已正确配置，代理节点可以正常连接
            </Text>
          </div>
        </div>
      </Card>

      <style>{`
        .settings-icon {
          width: 52px;
          height: 52px;
          border-radius: 14px;
          background: linear-gradient(135deg, #C9A9A6 0%, #B89591 100%);
          display: flex;
          align-items: center;
          justify-content: center;
          font-size: 24px;
          color: #fff;
        }

        .settings-main-card {
          border: none;
          box-shadow: 0 4px 24px rgba(60, 55, 48, 0.1);
        }

        .settings-card-header {
          background: linear-gradient(135deg, #3D3A36 0%, #4A4740 100%);
          padding: 24px 28px;
          display: flex;
          justify-content: space-between;
          align-items: flex-start;
        }

        .header-content {
          z-index: 1;
        }

        .header-decoration {
          position: absolute;
          right: 0;
          top: 0;
          pointer-events: none;
        }

        .settings-card-body {
          padding: 28px;
          background: #fff;
        }

        .url-helper {
          background: #F5F0E8;
          padding: 12px 16px;
          border-radius: 8px;
          margin-bottom: 24px;
          font-size: 13px;
        }

        .url-helper code {
          background: rgba(201, 169, 166, 0.2);
          padding: 2px 6px;
          border-radius: 4px;
          font-family: 'SF Mono', Monaco, monospace;
          color: #6B6760;
        }

        .settings-actions {
          display: flex;
          gap: 12px;
          justify-content: flex-end;
        }

        .info-card {
          transition: all 0.3s ease;
        }

        .info-card:hover {
          transform: translateY(-2px);
          box-shadow: 0 6px 20px rgba(60, 55, 48, 0.1);
        }

        @keyframes fadeInUp {
          from {
            opacity: 0;
            transform: translateY(16px);
          }
          to {
            opacity: 1;
            transform: translateY(0);
          }
        }

        .animate-in {
          animation: fadeInUp 0.4s ease forwards;
        }
      `}</style>
    </div>
  )
}
import { LockOutlined, UserOutlined } from '@ant-design/icons'
import { Button, Form, Input } from 'antd'

interface LoginViewProps {
  loading: boolean
  onFinish: (values: { username: string; password: string }) => void | Promise<void>
}

export default function LoginView({ loading, onFinish }: LoginViewProps) {
  return (
    <main className="login-page">
      <section className="login-page__brand">
        <div className="login-brand">
          <span>E</span>
          <strong>etcd manager</strong>
        </div>
        <div className="login-hero">
          <h1>
            掌控集群，<br />
            从容管理配置。
          </h1>
          <p className="login-hero__description">统一管理 etcd 集群、服务发现与配置生命周期，让基础设施状态清晰可见。</p>
          <div className="login-features">
            <span>集群监控</span>
            <span>权限控制</span>
            <span>审计追踪</span>
          </div>
        </div>
      </section>
      <section className="login-page__form-panel">
        <div className="login-form">
          <span className="login-form__eyebrow">Welcome back</span>
          <h2>登录管理中心</h2>
          <p>使用管理员分配的账号继续</p>
          <Form layout="vertical" size="large" onFinish={onFinish}>
            <Form.Item name="username" className="login-form__field" rules={[{ required: true, message: '请输入用户名' }]}>
              <Input aria-label="用户名" prefix={<UserOutlined />} placeholder="请输入用户名" autoComplete="username" />
            </Form.Item>
            <Form.Item name="password" className="login-form__field" rules={[{ required: true, message: '请输入密码' }]}>
              <Input.Password aria-label="密码" prefix={<LockOutlined />} placeholder="请输入密码" autoComplete="current-password" />
            </Form.Item>
            <Button type="primary" htmlType="submit" loading={loading} block>
              安全登录
            </Button>
          </Form>
          <small className="login-form__security">连接已加密 · 请勿共享账号凭据</small>
        </div>
      </section>
    </main>
  )
}

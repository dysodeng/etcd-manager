import { renderToStaticMarkup } from 'react-dom/server'
import { describe, expect, it, vi } from 'vitest'
import LoginView from './LoginView'

describe('LoginView', () => {
  it('renders product identity, credentials, and the single primary action', () => {
    const html = renderToStaticMarkup(<LoginView loading={false} onFinish={vi.fn()} />)
    expect(html).toContain('login-page__brand')
    expect(html).toContain('登录管理中心')
    expect(html).toContain('用户名')
    expect(html).toContain('密码')
    expect(html).toContain('安全登录')
  })
})

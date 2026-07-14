import { renderToStaticMarkup } from 'react-dom/server'
import { describe, expect, it, vi } from 'vitest'
import type { Environment, UserProfile } from '@/types'
import AppHeader from './AppHeader'
import AppSidebar from './AppSidebar'

const user: UserProfile = {
  user_id: 'admin-1',
  username: 'admin',
  is_super: true,
  role: null,
}

const environment: Environment = {
  id: 'env-1',
  name: 'Production',
  key_prefix: '/prod/',
  config_prefix: 'config/',
  gateway_prefix: 'gateway/',
  grpc_prefix: 'grpc/',
  description: '生产环境',
  sort_order: 1,
  created_at: '',
  updated_at: '',
}

describe('application shell', () => {
  it('renders grouped navigation and brand identity', () => {
    const html = renderToStaticMarkup(
      <AppSidebar user={user} pathname="/cluster" onNavigate={vi.fn()} />,
    )
    expect(html).toContain('app-sidebar')
    expect(html).toContain('etcd manager')
    expect(html).toContain('资源管理')
    expect(html).toContain('服务治理')
    expect(html).toContain('系统管理')
  })

  it('renders the current environment and account', () => {
    const html = renderToStaticMarkup(
      <AppHeader
        environments={[environment]}
        current={environment}
        user={user}
        canManageEnvironment
        themeMode="light"
        onEnvironmentChange={vi.fn()}
        onManageEnvironment={vi.fn()}
        onThemeChange={vi.fn()}
        onChangePassword={vi.fn()}
        onLogout={vi.fn()}
      />,
    )
    expect(html).toContain('app-header')
    expect(html).toContain('Production')
    expect(html).toContain('admin')
  })
})

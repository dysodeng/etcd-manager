import type { UserProfile } from '@/types'

export interface MenuItemConfig {
  key: string
  label: string
  section: 'resources' | 'services' | 'system'
  module?: string
  superOnly?: boolean
}

// 菜单配置（不含 icon，icon 在 AppSidebar 中绑定）
export const menuItemConfigs: MenuItemConfig[] = [
  { key: '/cluster', label: '集群信息', section: 'resources', module: 'cluster' },
  { key: '/kv', label: 'KV 管理', section: 'resources', module: 'kv' },
  { key: '/config', label: '配置中心', section: 'resources', module: 'config' },
  { key: '/gateway', label: '网关服务', section: 'services', module: 'gateway' },
  { key: '/grpc', label: 'gRPC 服务', section: 'services', module: 'grpc' },
  { key: '/users', label: '用户管理', section: 'system', module: 'users' },
  { key: '/roles', label: '角色管理', section: 'system', superOnly: true },
  { key: '/audit', label: '审计日志', section: 'system', module: 'audit_logs' },
]

const sections = [
  { key: 'resources', label: '资源管理' },
  { key: 'services', label: '服务治理' },
  { key: 'system', label: '系统管理' },
] as const

function canRead(user: UserProfile | null, module: string): boolean {
  if (!user) return false
  if (user.is_super) return true
  return Boolean(
    user.role?.permissions.some(
      (permission) => permission.module === module && (permission.can_read || permission.can_write),
    ),
  )
}

function isSuper(user: UserProfile | null): boolean {
  return user?.is_super === true
}

// 获取用户有权限访问的菜单项
export function getVisibleMenuKeys(user: UserProfile | null): string[] {
  return menuItemConfigs
    .filter((item) => {
      if (item.superOnly) return isSuper(user)
      if (item.module) return canRead(user, item.module)
      return true
    })
    .map((item) => item.key)
}

export function getVisibleMenuGroups(user: UserProfile | null) {
  const visible = new Set(getVisibleMenuKeys(user))
  return sections
    .map((section) => ({
      ...section,
      items: menuItemConfigs.filter((item) => item.section === section.key && visible.has(item.key)),
    }))
    .filter((section) => section.items.length > 0)
}

// 获取用户登录后应该跳转的默认页面
export function getDefaultRoute(user: UserProfile | null): string {
  const keys = getVisibleMenuKeys(user)
  return keys[0] ?? '/cluster'
}

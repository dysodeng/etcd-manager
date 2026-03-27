import type { UserProfile } from '@/types'
import { canRead, isSuper } from '@/stores/auth'

export interface MenuItemConfig {
  key: string
  label: string
  module?: string
  superOnly?: boolean
}

// 菜单配置（不含 icon，icon 在 MainLayout 中绑定）
export const menuItemConfigs: MenuItemConfig[] = [
  { key: '/cluster', label: '集群信息', module: 'cluster' },
  { key: '/kv', label: 'KV 管理', module: 'kv' },
  { key: '/config', label: '配置中心', module: 'config' },
  { key: '/gateway', label: '网关服务', module: 'gateway' },
  { key: '/grpc', label: 'gRPC 服务', module: 'grpc' },
  { key: '/users', label: '用户管理', module: 'users' },
  { key: '/roles', label: '角色管理', superOnly: true },
  { key: '/audit', label: '审计日志', module: 'audit_logs' },
]

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

// 获取用户登录后应该跳转的默认页面
export function getDefaultRoute(user: UserProfile | null): string {
  const keys = getVisibleMenuKeys(user)
  return keys[0] ?? '/cluster'
}

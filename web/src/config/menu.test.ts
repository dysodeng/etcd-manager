import { describe, expect, it } from 'vitest'
import type { UserProfile } from '@/types'
import { getDefaultRoute, getVisibleMenuGroups, getVisibleMenuKeys } from './menu'

const roleUser: UserProfile = {
  user_id: 'user-1',
  username: 'reader',
  is_super: false,
  role: {
    id: 'role-1',
    name: 'reader',
    permissions: [{ module: 'config', can_read: true, can_write: false }],
    environment_ids: [],
  },
}

describe('menu permissions', () => {
  it('shows only modules granted to a role user', () => {
    expect(getVisibleMenuKeys(roleUser)).toEqual(['/config'])
    expect(getDefaultRoute(roleUser)).toBe('/config')
  })

  it('shows every menu to a super admin', () => {
    const superUser: UserProfile = { ...roleUser, is_super: true, role: null }
    expect(getVisibleMenuKeys(superUser)).toHaveLength(8)
  })

  it('keeps visible menu items in the approved console groups', () => {
    const superUser: UserProfile = { ...roleUser, is_super: true, role: null }
    expect(getVisibleMenuGroups(superUser).map((group) => [group.label, group.items.map((item) => item.key)])).toEqual([
      ['资源管理', ['/cluster', '/kv', '/config']],
      ['服务治理', ['/gateway', '/grpc']],
      ['系统管理', ['/users', '/roles', '/audit']],
    ])
  })

  it('uses cluster as the empty-permission fallback', () => {
    expect(getDefaultRoute(null)).toBe('/cluster')
  })
})

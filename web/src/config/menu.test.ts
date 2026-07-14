import { describe, expect, it, vi } from 'vitest'
import type { UserProfile } from '@/types'
import { getDefaultRoute, getVisibleMenuKeys } from './menu'

vi.mock('@/stores/auth', () => ({
  canRead: (user: UserProfile | null, module: string) =>
    Boolean(
      user?.is_super ||
        user?.role?.permissions.some(
          (permission) => permission.module === module && (permission.can_read || permission.can_write),
        ),
    ),
  isSuper: (user: UserProfile | null) => user?.is_super === true,
}))

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

  it('uses cluster as the empty-permission fallback', () => {
    expect(getDefaultRoute(null)).toBe('/cluster')
  })
})

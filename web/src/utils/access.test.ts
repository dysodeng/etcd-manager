import { describe, expect, it } from 'vitest'
import type { UserProfile } from '@/types'
import { canRead, canWrite, isSuper } from './access'

const roleUser: UserProfile = {
  user_id: 'user-1',
  username: 'operator',
  is_super: false,
  role: {
    id: 'role-1',
    name: 'operator',
    permissions: [
      { module: 'config', can_read: true, can_write: false },
      { module: 'gateway', can_read: false, can_write: true },
    ],
    environment_ids: ['env-1'],
  },
}

describe('access helpers', () => {
  it('treats read and write grants as readable access', () => {
    expect(canRead(roleUser, 'config')).toBe(true)
    expect(canRead(roleUser, 'gateway')).toBe(true)
    expect(canRead(roleUser, 'audit_logs')).toBe(false)
  })

  it('requires an explicit write grant for role users', () => {
    expect(canWrite(roleUser, 'config')).toBe(false)
    expect(canWrite(roleUser, 'gateway')).toBe(true)
  })

  it('grants all access only to super users', () => {
    const superUser: UserProfile = { ...roleUser, is_super: true, role: null }

    expect(canRead(superUser, 'audit_logs')).toBe(true)
    expect(canWrite(superUser, 'environments')).toBe(true)
    expect(isSuper(superUser)).toBe(true)
    expect(isSuper(roleUser)).toBe(false)
    expect(canRead(null, 'config')).toBe(false)
  })
})

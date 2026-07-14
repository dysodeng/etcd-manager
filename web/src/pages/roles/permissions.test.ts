import { describe, expect, it } from 'vitest'
import { updatePermissionState } from './permissions'

describe('updatePermissionState', () => {
  it('enables read when write is enabled', () => {
    expect(updatePermissionState({}, 'kv', 'can_write').kv).toEqual({ can_read: true, can_write: true })
  })

  it('disables write when read is disabled', () => {
    const state = { kv: { can_read: true, can_write: true } }
    expect(updatePermissionState(state, 'kv', 'can_read').kv).toEqual({ can_read: false, can_write: false })
  })

  it('does not enable write when read is enabled', () => {
    expect(updatePermissionState({}, 'kv', 'can_read').kv).toEqual({ can_read: true, can_write: false })
  })
})

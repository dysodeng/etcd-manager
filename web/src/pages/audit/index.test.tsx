// @vitest-environment jsdom

import { afterEach, beforeAll, beforeEach, describe, expect, it, vi } from 'vitest'
import { cleanup, render, screen, waitFor } from '@testing-library/react'
import type { RolePermission, UserProfile } from '@/types'
import AuditPage from './index'

const boundary = vi.hoisted(() => ({
  user: null as UserProfile | null,
  list: vi.fn(),
}))

vi.mock('@/stores/auth', async () => {
  const access = await vi.importActual<typeof import('@/utils/access')>('@/utils/access')
  return {
    ...access,
    useAuthStore: () => ({ user: boundary.user }),
  }
})

vi.mock('@/api/audit', () => ({
  auditApi: { list: boundary.list },
}))

const roleUser = (permissions: RolePermission[]): UserProfile => ({
  user_id: 'user-1',
  username: 'operator',
  is_super: false,
  role: {
    id: 'role-1',
    name: 'operator',
    permissions,
    environment_ids: [],
  },
})

describe('AuditPage access', () => {
  beforeAll(() => {
    const getComputedStyle = window.getComputedStyle.bind(window)
    vi.spyOn(window, 'getComputedStyle').mockImplementation((element) => getComputedStyle(element))
    Object.defineProperty(window, 'matchMedia', {
      configurable: true,
      value: vi.fn().mockImplementation((query: string) => ({
        matches: false,
        media: query,
        onchange: null,
        addListener: vi.fn(),
        removeListener: vi.fn(),
        addEventListener: vi.fn(),
        removeEventListener: vi.fn(),
        dispatchEvent: vi.fn(),
      })),
    })
  })

  beforeEach(() => {
    boundary.list.mockReset()
    boundary.list.mockResolvedValue({ list: [], total: 0, page: 1, page_size: 20 })
  })

  afterEach(cleanup)

  it('renders 403 and does not fetch for an unauthorized direct visit', () => {
    boundary.user = roleUser([{ module: 'config', can_read: true, can_write: false }])

    render(<AuditPage />)

    expect(screen.getByText('无权访问')).toBeTruthy()
    expect(screen.queryByRole('button', { name: /搜索/ })).toBeNull()
    expect(boundary.list).not.toHaveBeenCalled()
  })

  it('renders controls and fetches for an authorized direct visit', async () => {
    boundary.user = roleUser([{ module: 'audit_logs', can_read: true, can_write: false }])

    render(<AuditPage />)

    expect(screen.getByRole('button', { name: /搜索/ })).toBeTruthy()
    await waitFor(() => {
      expect(boundary.list).toHaveBeenCalledWith({ page: 1, page_size: 20 })
    })
  })
})

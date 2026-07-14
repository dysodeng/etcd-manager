// @vitest-environment jsdom

import { afterEach, beforeAll, beforeEach, describe, expect, it, vi } from 'vitest'
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import { renderToStaticMarkup } from 'react-dom/server'
import { message } from 'antd'
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

  it('starts with table loading on the first render before effects run', () => {
    boundary.user = roleUser([{ module: 'audit_logs', can_read: true, can_write: false }])

    const html = renderToStaticMarkup(<AuditPage />)

    expect(html).toContain('ant-spin-spinning')
  })

  it('renders a retryable error state when the initial request fails', async () => {
    boundary.user = roleUser([{ module: 'audit_logs', can_read: true, can_write: false }])
    boundary.list.mockRejectedValueOnce(new Error('审计服务暂不可用'))

    render(<AuditPage />)

    expect(await screen.findByText('审计服务暂不可用')).toBeTruthy()
    boundary.list.mockResolvedValueOnce({ list: [], total: 0, page: 1, page_size: 20 })
    screen.getByRole('button', { name: '重新加载' }).click()
    await waitFor(() => expect(boundary.list).toHaveBeenCalledTimes(2))
  })

  it('renders the designed empty state after an empty response', async () => {
    boundary.user = roleUser([{ module: 'audit_logs', can_read: true, can_write: false }])

    render(<AuditPage />)

    expect(await screen.findByText('暂无审计日志')).toBeTruthy()
  })

  it('keeps loaded rows and shows message feedback when a later request fails', async () => {
    boundary.user = roleUser([{ module: 'audit_logs', can_read: true, can_write: false }])
    boundary.list
      .mockResolvedValueOnce({
        list: [{
          id: 'log-1',
          user_id: 'user-1',
          username: 'operator',
          action: 'update',
          resource_type: 'config',
          resource_key: '/app/key',
          detail: 'updated config',
          ip: '127.0.0.1',
          created_at: '2026-01-01T00:00:00Z',
          updated_at: '2026-01-01T00:00:00Z',
        }],
        total: 1,
        page: 1,
        page_size: 20,
      })
      .mockRejectedValueOnce(new Error('刷新审计日志失败'))
    const errorMessage = vi.spyOn(message, 'error').mockImplementation(() => undefined as never)

    render(<AuditPage />)
    expect(await screen.findByText('/app/key')).toBeTruthy()
    fireEvent.click(screen.getByRole('button', { name: /重置/ }))

    await waitFor(() => expect(errorMessage).toHaveBeenCalledWith('刷新审计日志失败'))
    expect(screen.getByText('/app/key')).toBeTruthy()
    errorMessage.mockRestore()
  })
})

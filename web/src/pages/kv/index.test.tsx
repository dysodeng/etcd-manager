// @vitest-environment jsdom

import { afterEach, beforeAll, beforeEach, describe, expect, it, vi } from 'vitest'
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import { renderToStaticMarkup } from 'react-dom/server'
import { message } from 'antd'
import type { KVItem, UserProfile } from '@/types'
import KVPage from './index'

const boundary = vi.hoisted(() => ({
  list: vi.fn(),
  create: vi.fn(),
  update: vi.fn(),
  delete: vi.fn(),
}))

const user: UserProfile = { user_id: 'admin-1', username: 'admin', is_super: true, role: null }
const item: KVItem = { key: '/app/key', value: 'value', version: 1, create_revision: 1, mod_revision: 1 }

vi.mock('@/stores/auth', async () => {
  const access = await vi.importActual<typeof import('@/utils/access')>('@/utils/access')
  return {
    ...access,
    useAuthStore: (selector: (state: { user: UserProfile }) => unknown) => selector({ user }),
  }
})

vi.mock('@/api/kv', () => ({ kvApi: boundary }))

describe('KVPage async states', () => {
  beforeAll(() => {
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
    boundary.list.mockReset().mockResolvedValue([item])
  })

  afterEach(cleanup)

  it('starts with table loading on the first render before effects run', () => {
    const html = renderToStaticMarkup(<KVPage />)
    expect(html).toContain('ant-spin-spinning')
  })

  it('keeps loaded rows and shows message feedback when refresh fails', async () => {
    boundary.list.mockResolvedValueOnce([item]).mockRejectedValueOnce(new Error('刷新 KV 失败'))
    const errorMessage = vi.spyOn(message, 'error').mockImplementation(() => undefined as never)

    render(<KVPage />)
    expect(await screen.findByText('/app/key')).toBeTruthy()
    const refresh = screen.getByRole<HTMLButtonElement>('button', { name: /刷新/ })
    await waitFor(() => expect(refresh.disabled).toBe(false))
    fireEvent.click(refresh)

    await waitFor(() => expect(errorMessage).toHaveBeenCalledWith('刷新 KV 失败'))
    expect(screen.getByText('/app/key')).toBeTruthy()
    errorMessage.mockRestore()
  })
})

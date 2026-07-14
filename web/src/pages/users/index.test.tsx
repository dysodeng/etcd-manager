// @vitest-environment jsdom

import { afterEach, beforeAll, beforeEach, describe, expect, it, vi } from 'vitest'
import { cleanup, fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import type { Role, User, UserProfile } from '@/types'
import UsersPage from './index'

const boundary = vi.hoisted(() => ({
  userList: vi.fn(),
  create: vi.fn(),
  update: vi.fn(),
  delete: vi.fn(),
  transferSuper: vi.fn(),
  roleList: vi.fn(),
}))

const currentUser: UserProfile = { user_id: 'admin-1', username: 'admin', is_super: true, role: null }
const users: User[] = [
  { id: 'admin-1', username: 'admin', is_super: true, role_id: null, role_name: '', created_at: '', updated_at: '' },
  { id: 'user-2', username: 'operator', is_super: false, role_id: 'role-1', role_name: 'Operator', created_at: '', updated_at: '' },
]
const roles: Role[] = [{
  id: 'role-1',
  name: 'Operator',
  description: '',
  permissions: [],
  environment_ids: [],
  user_count: 1,
  created_at: '',
  updated_at: '',
}]

vi.mock('@/stores/auth', async () => {
  const access = await vi.importActual<typeof import('@/utils/access')>('@/utils/access')
  return { ...access, useAuthStore: () => ({ user: currentUser }) }
})

vi.mock('@/api/user', () => ({
  userApi: {
    list: boundary.userList,
    create: boundary.create,
    update: boundary.update,
    delete: boundary.delete,
    transferSuper: boundary.transferSuper,
  },
}))

vi.mock('@/api/role', () => ({ roleApi: { list: boundary.roleList } }))

describe('UsersPage mutation locking', () => {
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
    boundary.userList.mockReset().mockResolvedValue({ list: users, total: users.length, page: 1, page_size: 20 })
    boundary.roleList.mockReset().mockResolvedValue({ list: roles, total: roles.length, page: 1, page_size: 100 })
    boundary.transferSuper.mockReset()
  })

  afterEach(cleanup)

  it('issues one super-admin transfer when confirmation is clicked twice rapidly', async () => {
    boundary.transferSuper.mockReturnValue(new Promise(() => {}))
    render(<UsersPage />)
    await screen.findByText('operator')
    fireEvent.click(screen.getByRole('button', { name: '转移超管' }))

    const dialog = await screen.findByRole('dialog')
    const selects = within(dialog).getAllByRole('combobox')
    fireEvent.mouseDown(selects[0]!)
    fireEvent.click(await screen.findByText('operator', { selector: '.ant-select-item-option-content' }))
    await waitFor(() => expect(within(dialog).getByTitle('operator')).toBeTruthy())
    fireEvent.mouseDown(selects[1]!)
    fireEvent.click(await screen.findByText('Operator', { selector: '.ant-select-item-option-content' }))
    await waitFor(() => expect(within(dialog).getByTitle('Operator')).toBeTruthy())

    const confirm = within(dialog).getByRole('button', { name: '确认转移' })
    fireEvent.click(confirm)
    expect(within(dialog).getByRole('button', { name: /确认转移/ }).classList.contains('ant-btn-loading')).toBe(true)
    fireEvent.click(confirm)

    await waitFor(() => expect(boundary.transferSuper).toHaveBeenCalledTimes(1))
  })

  it('keeps the selected role badge while rendering dropdown options as plain text', async () => {
    const { container } = render(<UsersPage />)
    await screen.findByText('operator')

    const roleSelect = container.querySelector('.user-role-select')
    expect(roleSelect?.querySelector('.ant-select-selection-item .status-badge')).toBeTruthy()

    fireEvent.mouseDown(within(roleSelect as HTMLElement).getByRole('combobox'))
    const option = await screen.findByText('Operator', { selector: '.ant-select-item-option-content' })

    expect(option.querySelector('.status-badge')).toBeNull()
  })
})

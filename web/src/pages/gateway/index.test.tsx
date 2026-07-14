// @vitest-environment jsdom

import { afterEach, beforeAll, beforeEach, describe, expect, it, vi } from 'vitest'
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import type { Environment, ServiceGroup, UserProfile } from '@/types'
import GatewayPage from './index'

const boundary = vi.hoisted(() => ({
  list: vi.fn(),
  updateStatus: vi.fn(),
}))

const user: UserProfile = { user_id: 'admin-1', username: 'admin', is_super: true, role: null }
const environment: Environment = {
  id: 'env-1',
  name: 'production',
  key_prefix: '/production/',
  config_prefix: 'config/',
  gateway_prefix: 'gateway/',
  grpc_prefix: 'grpc/',
  description: '',
  sort_order: 1,
  created_at: '',
  updated_at: '',
}
const groups: ServiceGroup[] = [{
  service_name: 'orders',
  instance_count: 1,
  healthy_count: 1,
  unhealthy_count: 0,
  instances: [{
    key: '/production/gateway/orders/instance-1',
    id: 'instance-1',
    service_name: 'orders',
    host: '127.0.0.1',
    port: 8080,
    weight: 100,
    version: 'v1',
    status: 'up',
    registered_at: '2026-01-01T00:00:00Z',
    metadata: {},
  }],
}]

vi.mock('@/stores/auth', async () => {
  const access = await vi.importActual<typeof import('@/utils/access')>('@/utils/access')
  return {
    ...access,
    useAuthStore: (selector: (state: { user: UserProfile }) => unknown) => selector({ user }),
  }
})

vi.mock('@/stores/environment', () => ({
  useEnvironmentStore: (selector: (state: { current: Environment }) => unknown) => selector({ current: environment }),
}))

vi.mock('@/api/gateway', () => ({ gatewayApi: boundary }))

describe('GatewayPage async states', () => {
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
    boundary.list.mockReset()
    boundary.updateStatus.mockReset()
  })

  afterEach(cleanup)

  it('keeps loaded service groups visible while a refresh is pending', async () => {
    boundary.list.mockResolvedValueOnce(groups)
    let finishRefresh!: (value: ServiceGroup[]) => void
    boundary.list.mockImplementationOnce(() => new Promise<ServiceGroup[]>((resolve) => { finishRefresh = resolve }))

    render(<GatewayPage />)
    expect(await screen.findByText('orders')).toBeTruthy()

    const refreshButton = screen.getByRole<HTMLButtonElement>('button', { name: /刷新数据/ })
    await waitFor(() => expect(refreshButton.disabled).toBe(false))
    fireEvent.click(refreshButton)

    expect(screen.getByText('orders')).toBeTruthy()
    finishRefresh(groups)
    await waitFor(() => expect(boundary.list).toHaveBeenCalledTimes(2))
  })

  it('renders the shared loading state during the initial request', () => {
    boundary.list.mockReturnValue(new Promise(() => {}))

    const { container } = render(<GatewayPage />)

    expect(container.querySelector('.async-state')).toBeTruthy()
  })
})

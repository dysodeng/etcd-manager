// @vitest-environment jsdom

import { afterEach, beforeAll, beforeEach, describe, expect, it, vi } from 'vitest'
import { cleanup, render, waitFor } from '@testing-library/react'
import type { ClusterMetrics, ClusterStatus } from '@/types'
import ClusterPage from './index'

const boundary = vi.hoisted(() => ({
  status: vi.fn(),
  metrics: vi.fn(),
  memberStatuses: vi.fn(),
  alarms: vi.fn(),
}))

vi.mock('@/api/cluster', () => ({ clusterApi: boundary }))

const status: ClusterStatus = {
  cluster_id: 'cluster-01',
  leader: 'etcd-2',
  members: [{
    id: 'member-01',
    name: 'etcd-1',
    peer_urls: ['http://etcd-1:2380'],
    client_urls: ['http://etcd-1:2379'],
    is_learner: false,
  }],
}

const metrics: ClusterMetrics = {
  cluster_id: 'cluster-01',
  leader_name: 'etcd-2',
  db_size: 1024,
  db_size_in_use: 1024,
  member_count: 1,
  version: '3.5.5',
  health: {},
}

describe('ClusterPage member summary', () => {
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
    boundary.status.mockResolvedValue(status)
    boundary.metrics.mockResolvedValue(metrics)
    boundary.memberStatuses.mockResolvedValue([])
    boundary.alarms.mockResolvedValue([])
  })

  afterEach(cleanup)

  it('places cluster identity in a compact card-header summary', async () => {
    const { container } = render(<ClusterPage />)

    await waitFor(() => expect(container.querySelector('.cluster-summary')).toBeTruthy())
    const summary = container.querySelector('.cluster-summary')

    expect(summary?.textContent).toContain('集群 ID')
    expect(summary?.textContent).toContain('cluster-01')
    expect(summary?.textContent).toContain('Leader')
    expect(summary?.textContent).toContain('etcd-2')
    expect(summary?.querySelector('[title="etcd-2"]')).toBeTruthy()
    expect(container.querySelector('.cluster-descriptions')).toBeNull()
  })
})

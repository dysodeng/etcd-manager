// @vitest-environment jsdom

import { afterEach, beforeAll, describe, expect, it, vi } from 'vitest'
import { cleanup, render, screen } from '@testing-library/react'
import SyncRestorePanel from './SyncRestorePanel'

describe('SyncRestorePanel confirmation', () => {
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

  afterEach(cleanup)

  it('names selected environments and disables restore with no selection', () => {
    const props = {
      statuses: [
        { environment_id: 'env-1', environment_name: 'production', db_key_count: 3, etcd_key_count: 0, need_restore: true },
        { environment_id: 'env-2', environment_name: 'staging', db_key_count: 2, etcd_key_count: 0, need_restore: true },
      ],
      selectedIds: [] as string[],
      open: true,
      restoring: false,
      onOpen: vi.fn(),
      onClose: vi.fn(),
      onDismiss: vi.fn(),
      onSelectionChange: vi.fn(),
      onRestore: vi.fn(),
    }

    const { rerender } = render(<SyncRestorePanel {...props} />)
    expect(screen.getByRole<HTMLButtonElement>('button', { name: '恢复选中环境' }).disabled).toBe(true)

    rerender(<SyncRestorePanel {...props} selectedIds={['env-1']} />)
    expect(screen.getByText('即将恢复：production')).toBeTruthy()
  })
})

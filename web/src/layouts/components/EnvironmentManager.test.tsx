// @vitest-environment jsdom

import { afterEach, beforeAll, describe, expect, it, vi } from 'vitest'
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import EnvironmentManager from './EnvironmentManager'

describe('EnvironmentManager submission locking', () => {
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

  it('issues one save when confirmation is clicked twice rapidly', async () => {
    const onSave = vi.fn(() => new Promise<void>(() => {}))
    render(
      <EnvironmentManager
        open
        environments={[]}
        canManage
        onClose={vi.fn()}
        onDelete={vi.fn()}
        onSave={onSave}
      />,
    )

    fireEvent.click(screen.getByRole('button', { name: /新建环境/ }))
    fireEvent.change(screen.getByLabelText('环境名称'), { target: { value: 'production' } })
    fireEvent.change(screen.getByLabelText('Key 前缀'), { target: { value: '/production/' } })
    const save = screen.getByRole('button', { name: /保.*存/ })

    fireEvent.click(save)
    expect(screen.getByRole('button', { name: /保.*存/ }).classList.contains('ant-btn-loading')).toBe(true)
    fireEvent.click(save)

    await waitFor(() => expect(onSave).toHaveBeenCalledTimes(1))
  })

  it('releases the submission lock after validation fails', async () => {
    const onSave = vi.fn().mockResolvedValue(undefined)
    render(
      <EnvironmentManager
        open
        environments={[]}
        canManage
        onClose={vi.fn()}
        onDelete={vi.fn()}
        onSave={onSave}
      />,
    )

    fireEvent.click(screen.getByRole('button', { name: /新建环境/ }))
    fireEvent.click(screen.getByRole('button', { name: /保.*存/ }))
    expect(await screen.findByText('请输入环境名称')).toBeTruthy()

    fireEvent.change(screen.getByLabelText('环境名称'), { target: { value: 'production' } })
    fireEvent.change(screen.getByLabelText('Key 前缀'), { target: { value: '/production/' } })
    fireEvent.click(screen.getByRole('button', { name: /保.*存/ }))

    await waitFor(() => expect(onSave).toHaveBeenCalledTimes(1))
  })
})

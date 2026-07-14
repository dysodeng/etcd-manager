import { describe, expect, it } from 'vitest'
import { formatBytes, getFragmentation } from './presentation'

describe('cluster presentation', () => {
  it('formats storage units', () => {
    expect(formatBytes(1024)).toBe('1.0 KB')
    expect(formatBytes(1073741824)).toBe('1.00 GB')
  })

  it('maps fragmentation to semantic tones', () => {
    expect(getFragmentation(100, 71)).toEqual({ percent: 29, tone: 'success' })
    expect(getFragmentation(100, 70)).toEqual({ percent: 30, tone: 'warning' })
    expect(getFragmentation(100, 49)).toEqual({ percent: 51, tone: 'danger' })
  })
})

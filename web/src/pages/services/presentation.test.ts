import { describe, expect, it } from 'vitest'
import { buildServiceSummary } from './presentation'

describe('buildServiceSummary', () => {
  it('summarizes degraded service groups', () => {
    expect(buildServiceSummary([
      { instance_count: 3, healthy_count: 2 },
      { instance_count: 2, healthy_count: 2 },
    ])).toEqual({
      services: 2,
      instances: 5,
      healthy: 4,
      healthDisplay: '80.0%',
      tone: 'warning',
    })
  })

  it('distinguishes fully healthy and empty services', () => {
    expect(buildServiceSummary([{ instance_count: 5, healthy_count: 5 }]).tone).toBe('success')
    expect(buildServiceSummary([])).toEqual({
      services: 0,
      instances: 0,
      healthy: 0,
      healthDisplay: '0%',
      tone: 'default',
    })
  })
})

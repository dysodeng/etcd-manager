import { describe, expect, it } from 'vitest'
import { createAppTheme } from './index'

describe('createAppTheme', () => {
  it('uses the shared brand color in light mode', () => {
    const config = createAppTheme(false)
    expect(config.token?.colorPrimary).toBe('#316ff6')
    expect(config.token?.borderRadius).toBe(10)
  })

  it('uses dark surfaces without changing the brand color', () => {
    const config = createAppTheme(true)
    expect(config.token?.colorPrimary).toBe('#316ff6')
    expect(config.token?.colorBgBase).toBe('#0d1420')
  })
})

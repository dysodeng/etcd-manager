import { renderToStaticMarkup } from 'react-dom/server'
import { describe, expect, it } from 'vitest'
import { FragmentationProgress } from './FragmentationProgress'

describe('FragmentationProgress', () => {
  it.each([
    { dbSizeInUse: 71, percent: 29, tone: 'success' },
    { dbSizeInUse: 70, percent: 30, tone: 'warning' },
    { dbSizeInUse: 49, percent: 51, tone: 'danger' },
  ])('renders $percent% with the $tone semantic tone', ({ dbSizeInUse, percent, tone }) => {
    const html = renderToStaticMarkup(
      <FragmentationProgress dbSize={100} dbSizeInUse={dbSizeInUse} />,
    )

    expect(html).toContain(`fragmentation-progress--${tone}`)
    expect(html).toContain(`data-tone="${tone}"`)
    expect(html).toContain(`aria-label="碎片率 ${percent}%"`)
  })
})

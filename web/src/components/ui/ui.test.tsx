import { renderToStaticMarkup } from 'react-dom/server'
import { describe, expect, it } from 'vitest'
import { MetricCard, PageHeader, StatusBadge } from './index'

describe('console UI primitives', () => {
  it('renders page identity and the primary action', () => {
    const html = renderToStaticMarkup(
      <PageHeader eyebrow="Cluster Overview" title="集群概览" description="实时状态" extra={<button>刷新</button>} />,
    )
    expect(html).toContain('page-header')
    expect(html).toContain('Cluster Overview')
    expect(html).toContain('实时状态')
    expect(html).toContain('刷新')
  })

  it('renders metrics and semantic status classes', () => {
    expect(renderToStaticMarkup(<MetricCard label="成员" value={3} hint="全部在线" tone="success" />)).toContain('metric-card--success')
    expect(renderToStaticMarkup(<StatusBadge tone="danger">异常</StatusBadge>)).toContain('status-badge--danger')
  })
})

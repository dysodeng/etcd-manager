import type { ReactNode } from 'react'

export function StatusBadge({
  tone,
  children,
}: {
  tone: 'success' | 'warning' | 'danger' | 'info' | 'neutral'
  children: ReactNode
}) {
  return (
    <span className={`status-badge status-badge--${tone}`}>
      <i aria-hidden="true" />
      {children}
    </span>
  )
}

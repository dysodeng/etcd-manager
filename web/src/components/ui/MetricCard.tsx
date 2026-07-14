import type { ReactNode } from 'react'

type Tone = 'default' | 'primary' | 'success' | 'warning' | 'danger'

export function MetricCard({
  label,
  value,
  hint,
  icon,
  tone = 'default',
}: {
  label: ReactNode
  value: ReactNode
  hint?: ReactNode
  icon?: ReactNode
  tone?: Tone
}) {
  return (
    <article className={`metric-card metric-card--${tone}`}>
      <div className="metric-card__top">
        <span>{label}</span>
        {icon}
      </div>
      <strong>{value}</strong>
      {hint && <small>{hint}</small>}
    </article>
  )
}

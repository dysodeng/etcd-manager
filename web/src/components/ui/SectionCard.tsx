import type { ReactNode } from 'react'

export function SectionCard({
  title,
  description,
  extra,
  children,
  className = '',
}: {
  title?: ReactNode
  description?: ReactNode
  extra?: ReactNode
  children: ReactNode
  className?: string
}) {
  return (
    <section className={`section-card ${className}`.trim()}>
      {(title || description || extra) && (
        <div className="section-card__header">
          <div>
            <h2>{title}</h2>
            {description && <p>{description}</p>}
          </div>
          {extra}
        </div>
      )}
      <div className="section-card__body">{children}</div>
    </section>
  )
}

import type { ReactNode } from 'react'

interface PageHeaderProps {
  eyebrow?: string
  title: string
  description?: string
  extra?: ReactNode
}

export function PageHeader({ eyebrow, title, description, extra }: PageHeaderProps) {
  return (
    <header className="page-header">
      <div>
        {eyebrow && <div className="page-header__eyebrow">{eyebrow}</div>}
        <h1>{title}</h1>
        {description && <p>{description}</p>}
      </div>
      {extra && <div className="page-header__extra">{extra}</div>}
    </header>
  )
}

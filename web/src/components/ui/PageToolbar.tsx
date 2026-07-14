import type { ReactNode } from 'react'

export function PageToolbar({ children, trailing }: { children: ReactNode; trailing?: ReactNode }) {
  return (
    <div className="page-toolbar">
      <div className="page-toolbar__main">{children}</div>
      {trailing && <div className="page-toolbar__trailing">{trailing}</div>}
    </div>
  )
}

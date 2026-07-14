import type { ReactNode } from 'react'
import { Button, Empty, Skeleton } from 'antd'

export function LoadingState({ rows = 4 }: { rows?: number }) {
  return (
    <div className="async-state">
      <Skeleton active paragraph={{ rows }} />
    </div>
  )
}

export function EmptyState({ title, description, action }: { title: string; description?: string; action?: ReactNode }) {
  return (
    <div className="async-state">
      <Empty
        description={
          <>
            <strong>{title}</strong>
            {description && <p>{description}</p>}
          </>
        }
      >
        {action}
      </Empty>
    </div>
  )
}

export function ErrorState({
  title = '加载失败',
  description,
  onRetry,
}: {
  title?: string
  description: string
  onRetry?: () => void
}) {
  return (
    <div className="async-state async-state--error">
      <strong>{title}</strong>
      <p>{description}</p>
      {onRetry && <Button onClick={onRetry}>重新加载</Button>}
    </div>
  )
}

import { Progress } from 'antd'
import { getFragmentation } from './presentation'

export function FragmentationProgress({
  dbSize,
  dbSizeInUse,
}: {
  dbSize: number
  dbSizeInUse: number
}) {
  const { percent, tone } = getFragmentation(dbSize, dbSizeInUse)

  return (
    <Progress
      className={`fragmentation-progress fragmentation-progress--${tone}`}
      data-tone={tone}
      aria-label={`碎片率 ${percent}%`}
      percent={percent}
      size="small"
      status="normal"
      strokeColor="var(--fragmentation-tone)"
    />
  )
}

import type { ClusterMetrics } from '@/types'

export type MetricTone = 'success' | 'warning' | 'danger'

export function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  if (bytes < 1024 * 1024 * 1024) return `${(bytes / 1024 / 1024).toFixed(2)} MB`
  return `${(bytes / 1024 / 1024 / 1024).toFixed(2)} GB`
}

export function getFragmentation(
  dbSize: number,
  dbSizeInUse: number,
): { percent: number; tone: MetricTone } {
  const percent = dbSize > 0 ? Math.round((1 - dbSizeInUse / dbSize) * 100) : 0
  return {
    percent,
    tone: percent > 50 ? 'danger' : percent >= 30 ? 'warning' : 'success',
  }
}

export function buildClusterMetricView(metrics: ClusterMetrics) {
  const fragmentation = getFragmentation(metrics.db_size, metrics.db_size_in_use)
  const healthValues = Object.values(metrics.health)
  const healthy = healthValues.filter(Boolean).length

  return [
    {
      key: 'members',
      label: '集群成员',
      value: metrics.member_count,
      hint: `${healthy}/${healthValues.length} 节点健康`,
      tone: healthy === healthValues.length ? 'success' as const : 'danger' as const,
    },
    {
      key: 'db-size',
      label: '数据库大小',
      value: formatBytes(metrics.db_size),
      hint: `实际使用 ${formatBytes(metrics.db_size_in_use)}`,
      tone: 'default' as const,
    },
    {
      key: 'fragmentation',
      label: '碎片率',
      value: `${fragmentation.percent}%`,
      hint: '数据库空间碎片',
      tone: fragmentation.tone,
    },
    {
      key: 'version',
      label: 'etcd 版本',
      value: metrics.version,
      hint: `集群 ${metrics.cluster_id}`,
      tone: 'default' as const,
    },
  ]
}

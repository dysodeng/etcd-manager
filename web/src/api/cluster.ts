import client, { request } from './client'
import type { ClusterStatus, ClusterMetrics } from '@/types'

export const clusterApi = {
  status: () => request<ClusterStatus>(client.get('/cluster/status')),
  metrics: () => request<ClusterMetrics>(client.get('/cluster/metrics')),
}

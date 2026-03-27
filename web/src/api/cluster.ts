import client, { request } from './client'
import type { ClusterStatus, ClusterMetrics, MemberStatus, AlarmInfo } from '@/types'

export const clusterApi = {
  status: () => request<ClusterStatus>(client.get('/cluster/status')),
  metrics: () => request<ClusterMetrics>(client.get('/cluster/metrics')),
  memberStatuses: () => request<MemberStatus[]>(client.get('/cluster/member-statuses')),
  alarms: () => request<AlarmInfo[]>(client.get('/cluster/alarms')),
}

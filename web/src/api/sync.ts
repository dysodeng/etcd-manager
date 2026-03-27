import client, { request } from './client'

export interface EnvSyncStatus {
  environment_id: string
  environment_name: string
  etcd_key_count: number
  db_key_count: number
  need_restore: boolean
}

export interface RestoreResult {
  environment_id: string
  environment_name: string
  total: number
  success: number
  failed: string[]
}

export const syncApi = {
  check: () => request<EnvSyncStatus[]>(client.get('/sync/check')),
  restore: (environmentIds: string[]) =>
    request<RestoreResult[]>(client.post('/sync/restore', { environment_ids: environmentIds })),
}

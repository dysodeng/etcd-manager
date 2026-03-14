import client, { request } from './client'
import type {
  ConfigItem, ConfigCreateRequest, ConfigUpdateRequest,
  ConfigRollbackRequest, ConfigRevision, PaginatedData, ImportResult,
} from '@/types'

export const configApi = {
  list: (env: string, prefix?: string) =>
    request<ConfigItem[]>(client.get('/configs', { params: { env, prefix } })),
  create: (data: ConfigCreateRequest) =>
    request<null>(client.post('/configs', data)),
  update: (data: ConfigUpdateRequest) =>
    request<null>(client.put('/configs', data)),
  delete: (env: string, key: string) =>
    request<null>(client.delete('/configs', { params: { env, key } })),
  revisions: (env: string, key: string, page = 1, pageSize = 20) =>
    request<PaginatedData<ConfigRevision>>(
      client.get('/configs/revisions', { params: { env, key, page, page_size: pageSize } }),
    ),
  rollback: (data: ConfigRollbackRequest) =>
    request<null>(client.post('/configs/rollback', data)),
  export: (env: string, format: 'json' | 'yaml' = 'json') =>
    client.get('/configs/export', { params: { env, format }, responseType: 'blob' }),
  import: (env: string, file: File, dryRun = false) => {
    return request<ImportResult>(
      client.post(`/configs/import?env=${env}&dry_run=${dryRun}`, file, {
        headers: { 'Content-Type': 'application/octet-stream' },
      }),
    )
  },
}

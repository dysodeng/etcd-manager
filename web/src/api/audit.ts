import client, { request } from './client'
import type { AuditLog, AuditLogFilter, PaginatedData } from '@/types'

export const auditApi = {
  list: (filter: AuditLogFilter = {}) =>
    request<PaginatedData<AuditLog>>(client.get('/audit-logs', { params: filter })),
}

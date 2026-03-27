import client, { request } from './client'
import type { Role, RoleCreateRequest, RoleUpdateRequest, PaginatedData } from '@/types'

export const roleApi = {
  list: (page = 1, pageSize = 50) =>
    request<PaginatedData<Role>>(client.get('/roles', { params: { page, page_size: pageSize } })),
  getById: (id: string) => request<Role>(client.get(`/roles/${id}`)),
  create: (data: RoleCreateRequest) => request<Role>(client.post('/roles', data)),
  update: (id: string, data: RoleUpdateRequest) => request<null>(client.put(`/roles/${id}`, data)),
  delete: (id: string) => request<null>(client.delete(`/roles/${id}`)),
}

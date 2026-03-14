import client, { request } from './client'
import type { User, UserCreateRequest, UserUpdateRequest, PaginatedData } from '@/types'

export const userApi = {
  list: (page = 1, pageSize = 20) =>
    request<PaginatedData<User>>(client.get('/users', { params: { page, page_size: pageSize } })),
  create: (data: UserCreateRequest) => request<User>(client.post('/users', data)),
  update: (id: string, data: UserUpdateRequest) => request<null>(client.put(`/users/${id}`, data)),
  delete: (id: string) => request<null>(client.delete(`/users/${id}`)),
}

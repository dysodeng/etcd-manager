import client, { request } from './client'
import type { Environment, EnvironmentCreateRequest } from '@/types'

export const environmentApi = {
  list: () => request<Environment[]>(client.get('/environments')),
  create: (data: EnvironmentCreateRequest) =>
    request<Environment>(client.post('/environments', data)),
  update: (id: string, data: EnvironmentCreateRequest) =>
    request<null>(client.put(`/environments/${id}`, data)),
  delete: (id: string) => request<null>(client.delete(`/environments/${id}`)),
}

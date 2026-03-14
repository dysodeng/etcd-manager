import client, { request } from './client'
import type { ServiceGroup } from '@/types'

export const gatewayApi = {
  list: (prefix: string) =>
    request<ServiceGroup[]>(client.get('/gateway', { params: { prefix } })),
  updateStatus: (key: string, status: 'up' | 'down') =>
    request<null>(client.put('/gateway/status', { key, status })),
}

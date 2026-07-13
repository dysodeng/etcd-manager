import client, { request } from './client'
import type { ServiceGroup } from '@/types'

export const gatewayApi = {
  list: (env: string) =>
    request<ServiceGroup[]>(client.get('/gateway', { params: { env } })),
  updateStatus: (env: string, key: string, status: 'up' | 'down') =>
    request<null>(client.put('/gateway/status', { env, key, status })),
}

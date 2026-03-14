import client, { request } from './client'
import type { GrpcServiceGroup } from '@/types'

export const grpcApi = {
  list: (prefix: string) =>
    request<GrpcServiceGroup[]>(client.get('/grpc', { params: { prefix } })),
  updateStatus: (key: string, status: 'up' | 'down') =>
    request<null>(client.put('/grpc/status', { key, status })),
}

import client, { request } from './client'
import type { GrpcServiceGroup } from '@/types'

export const grpcApi = {
  list: (env: string) =>
    request<GrpcServiceGroup[]>(client.get('/grpc', { params: { env } })),
  updateStatus: (env: string, key: string, status: 'up' | 'down') =>
    request<null>(client.put('/grpc/status', { env, key, status })),
}

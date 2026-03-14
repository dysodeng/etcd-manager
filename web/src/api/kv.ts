import client, { request } from './client'
import type { KVItem } from '@/types'

export const kvApi = {
  list: (prefix = '/', limit = 50) =>
    request<KVItem[]>(client.get('/kv', { params: { prefix, limit } })),
  get: (key: string) =>
    request<KVItem>(client.get('/kv', { params: { key } })),
  create: (key: string, value: string) =>
    request<null>(client.post('/kv', { key, value })),
  update: (key: string, value: string) =>
    request<null>(client.put('/kv', { key, value })),
  delete: (key: string) =>
    request<null>(client.delete('/kv', { params: { key } })),
}

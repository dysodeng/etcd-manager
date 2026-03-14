import axios from 'axios'
import type { ApiResponse } from '@/types'

const client = axios.create({ baseURL: '/api/v1', timeout: 10000 })

client.interceptors.request.use((config) => {
  const token = localStorage.getItem('token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

client.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      localStorage.removeItem('token')
      window.location.href = '/login'
    }
    return Promise.reject(error)
  },
)

export async function request<T>(promise: Promise<{ data: ApiResponse<T> }>): Promise<T> {
  const { data: resp } = await promise
  if (resp.code !== 0) {
    throw new Error(resp.message)
  }
  return resp.data
}

export default client

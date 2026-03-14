import client, { request } from './client'
import type { LoginRequest, LoginResponse, User, ChangePasswordRequest } from '@/types'

export const authApi = {
  login: (data: LoginRequest) => request<LoginResponse>(client.post('/auth/login', data)),
  logout: () => request<null>(client.post('/auth/logout')),
  getProfile: () => request<User>(client.get('/auth/profile')),
  changePassword: (data: ChangePasswordRequest) => request<null>(client.put('/auth/password', data)),
}

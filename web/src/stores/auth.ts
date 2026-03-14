import { create } from 'zustand'
import type { User } from '@/types'
import { authApi } from '@/api/auth'

interface AuthState {
  token: string | null
  user: User | null
  loading: boolean
  login: (username: string, password: string) => Promise<void>
  logout: () => void
  fetchProfile: () => Promise<void>
}

export const useAuthStore = create<AuthState>((set) => ({
  token: localStorage.getItem('token'),
  user: null,
  loading: false,

  login: async (username, password) => {
    const res = await authApi.login({ username, password })
    localStorage.setItem('token', res.token)
    set({
      token: res.token,
      user: { id: res.user_id, username: res.username, role: res.role as 'admin' | 'viewer', created_at: '', updated_at: '' },
    })
  },

  logout: () => {
    localStorage.removeItem('token')
    set({ token: null, user: null })
  },

  fetchProfile: async () => {
    set({ loading: true })
    try {
      const user = await authApi.getProfile()
      set({ user, loading: false })
    } catch {
      localStorage.removeItem('token')
      set({ token: null, user: null, loading: false })
    }
  },
}))

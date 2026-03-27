import { create } from 'zustand'
import type { UserProfile } from '@/types'
import { authApi } from '@/api/auth'

interface AuthState {
  token: string | null
  user: UserProfile | null
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
      user: {
        user_id: res.user_id,
        username: res.username,
        is_super: res.is_super,
        role: res.role,
      },
    })
  },

  logout: () => {
    localStorage.removeItem('token')
    set({ token: null, user: null })
  },

  fetchProfile: async () => {
    set({ loading: true })
    try {
      const profile = await authApi.getProfile()
      set({ user: profile, loading: false })
    } catch {
      localStorage.removeItem('token')
      set({ token: null, user: null, loading: false })
    }
  },
}))

// 权限工具函数
export function canRead(user: UserProfile | null, module: string): boolean {
  if (!user) return false
  if (user.is_super) return true
  if (!user.role) return false
  return user.role.permissions.some(p => p.module === module && (p.can_read || p.can_write))
}

export function canWrite(user: UserProfile | null, module: string): boolean {
  if (!user) return false
  if (user.is_super) return true
  if (!user.role) return false
  return user.role.permissions.some(p => p.module === module && p.can_write)
}

export function isSuper(user: UserProfile | null): boolean {
  return user?.is_super === true
}

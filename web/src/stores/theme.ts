import { create } from 'zustand'
import { useSyncExternalStore } from 'react'

export type ThemeMode = 'light' | 'dark' | 'system'

interface ThemeState {
  mode: ThemeMode
  setMode: (mode: ThemeMode) => void
}

export const useThemeStore = create<ThemeState>((set) => ({
  mode: (localStorage.getItem('theme') as ThemeMode) || 'system',
  setMode: (mode) => {
    localStorage.setItem('theme', mode)
    set({ mode })
  },
}))

// 监听系统深色模式
function useSystemDark(): boolean {
  return useSyncExternalStore(
    (callback) => {
      const mq = window.matchMedia('(prefers-color-scheme: dark)')
      mq.addEventListener('change', callback)
      return () => mq.removeEventListener('change', callback)
    },
    () => window.matchMedia('(prefers-color-scheme: dark)').matches,
  )
}

// 根据 mode 和系统偏好，返回实际是否为 dark
export function useIsDark(): boolean {
  const mode = useThemeStore((s) => s.mode)
  const systemDark = useSystemDark()
  if (mode === 'system') return systemDark
  return mode === 'dark'
}

import { create } from 'zustand'
import type { Environment } from '@/types'
import { environmentApi } from '@/api/environment'

interface EnvironmentState {
  environments: Environment[]
  current: Environment | null
  loading: boolean
  fetch: () => Promise<void>
  setCurrent: (env: Environment) => void
}

export const useEnvironmentStore = create<EnvironmentState>((set, get) => ({
  environments: [],
  current: null,
  loading: false,

  fetch: async () => {
    set({ loading: true })
    const envs = await environmentApi.list()
    const current = get().current
    set({
      environments: envs,
      current: current ?? envs[0] ?? null,
      loading: false,
    })
  },

  setCurrent: (env) => set({ current: env }),
}))

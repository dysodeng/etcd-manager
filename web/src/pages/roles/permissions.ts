export type PermissionState = Record<string, { can_read: boolean; can_write: boolean }>

export function updatePermissionState(
  state: PermissionState,
  module: string,
  field: 'can_read' | 'can_write',
): PermissionState {
  const current = state[module] ?? { can_read: false, can_write: false }
  const updated = { ...current, [field]: !current[field] }

  if (field === 'can_write' && updated.can_write) updated.can_read = true
  if (field === 'can_read' && !updated.can_read) updated.can_write = false

  return { ...state, [module]: updated }
}

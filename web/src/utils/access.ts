import type { UserProfile } from '@/types'

export function canRead(user: UserProfile | null, module: string): boolean {
  if (!user) return false
  if (user.is_super) return true
  if (!user.role) return false
  return user.role.permissions.some(
    (permission) => permission.module === module && (permission.can_read || permission.can_write),
  )
}

export function canWrite(user: UserProfile | null, module: string): boolean {
  if (!user) return false
  if (user.is_super) return true
  if (!user.role) return false
  return user.role.permissions.some(
    (permission) => permission.module === module && permission.can_write,
  )
}

export function isSuper(user: UserProfile | null): boolean {
  return user?.is_super === true
}

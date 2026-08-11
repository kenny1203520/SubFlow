import type { AccessRole } from './api/types'

type Translate = (key: string, values?: Record<string, string | number>) => string

/** Keeps protected role keys stable while presenting them in the active language. */
export function roleLabel(role: Pick<AccessRole, 'key'|'name'> | undefined, tr: Translate) {
  if (!role) return ''
  if (role.key === 'owner' || role.key === 'member') return tr(role.key)
  return role.name || role.key
}

export function roleSearchText(role: Pick<AccessRole, 'key'|'name'|'category'>, tr: Translate) {
  return [roleLabel(role, tr), role.name, role.key, role.category].filter(Boolean).join(' ')
}

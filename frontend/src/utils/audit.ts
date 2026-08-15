import type { AuditLog } from '../api/types'
import { auditText } from '../locales/audit'
import { formatMoney } from '../api/money'

type Locale = 'zh-TW' | 'en'

// Fields whose value is a PocketBase user ID rather than plain data --
// resolved to "Name (id)" via a caller-supplied lookup instead of shown raw.
const userIDFields = new Set(['from_user_id', 'to_user_id', 'paid_by'])

const fallback = (value: string) => value.replace(/[._-]+/g, ' ').replace(/\b\w/g, char => char.toUpperCase())
export function auditPresentation(log: AuditLog, locale: Locale) {
  const text = auditText[locale]
  return { action: text.action[log.action] || fallback(log.action), resource: text.resource[log.resource] || fallback(log.resource), outcome: log.outcome === 'success' ? text.success : log.outcome === 'failure' ? text.failure : fallback(log.outcome), actor: log.actorName || text.system }
}

export interface AuditChange { field: string; before?: unknown; after?: unknown }
export interface AuditSummary { details?: Record<string, unknown>; changes?: AuditChange[] }

// The backend has always emitted a plain string summary; only recent rows use
// the structured JSON shape (see backend/internal/application/auditsummary.go).
// Older rows, and any row saved before this change shipped, stay plain text.
export function parseAuditSummary(raw?: string): AuditSummary | null {
  if (!raw) return null
  try {
    const parsed = JSON.parse(raw)
    if (parsed && typeof parsed === 'object' && !Array.isArray(parsed)) return parsed as AuditSummary
  } catch { /* legacy plain-text summary */ }
  return null
}

export function auditFieldLabel(field: string, locale: Locale) {
  return auditText[locale].field[field as keyof typeof auditText[Locale]['field']] || fallback(field)
}

export function auditValueLabel(value: unknown, locale: Locale): string {
  const text = auditText[locale]
  if (value === null || value === undefined || value === '') return text.empty
  if (typeof value === 'boolean') return value ? text.yes : text.no
  if (Array.isArray(value)) return value.length ? value.map(item => auditValueLabel(item, locale)).join(text.listSeparator) : text.empty
  if (typeof value === 'object') return Object.entries(value as Record<string, unknown>).map(([key, item]) => `${auditFieldLabel(key, locale)}: ${auditValueLabel(item, locale)}`).join('; ')
  return String(value)
}

export type ResolveUser = (id: string) => string

// Renders one field's value, given the rest of the same details/changes
// object for context: amount_minor is paired with the object's own currency
// (the backend always stores it alongside, per auditsummary.go) instead of
// showing a bare, currency-less integer; user-ID fields go through the
// caller's member lookup instead of showing a raw PocketBase ID.
export function auditFieldValue(field: string, value: unknown, siblingFields: Record<string, unknown> | undefined, locale: Locale, resolveUser: ResolveUser): string {
  if (value === null || value === undefined || value === '') return auditText[locale].empty
  if (field === 'amount_minor' && typeof value === 'number') return formatMoney(value, String(siblingFields?.currency ?? 'TWD'), locale)
  if (userIDFields.has(field) && typeof value === 'string') return resolveUser(value)
  return auditValueLabel(value, locale)
}

// A details/changes object's own "currency" entry is only worth listing on
// its own when there's no amount_minor to pair it with -- otherwise it's
// already folded into that field's formatted value and would just repeat.
export function visibleAuditFields(fields: Record<string, unknown> | string[]): string[] {
  const keys = Array.isArray(fields) ? fields : Object.keys(fields)
  return keys.includes('amount_minor') ? keys.filter(key => key !== 'currency') : keys
}

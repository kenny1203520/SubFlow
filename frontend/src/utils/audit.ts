import type { AuditLog } from '../api/types'
import { auditText } from '../locales/audit'

type Locale = 'zh-TW' | 'en'

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

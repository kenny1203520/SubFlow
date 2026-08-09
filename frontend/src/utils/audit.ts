import type { AuditLog } from '../api/types'
import { auditText } from '../locales/audit'

const fallback = (value: string) => value.replace(/[._-]+/g, ' ').replace(/\b\w/g, char => char.toUpperCase())
export function auditPresentation(log: AuditLog, locale: 'zh-TW' | 'en') {
  const text = auditText[locale]
  return { action: text.action[log.action] || fallback(log.action), resource: text.resource[log.resource] || fallback(log.resource), outcome: log.outcome === 'success' ? text.success : log.outcome === 'failure' ? text.failure : fallback(log.outcome), actor: log.actorName || text.system }
}

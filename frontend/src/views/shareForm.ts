import type { ShareAccessMode, ShareRangeMode } from '../api/types'

export interface ShareFormState {
  name: string
  accessMode: ShareAccessMode
  password: string
  viewerEmails: string
  enabled: boolean
  expiresAt: string
  rangeMode: ShareRangeMode
  rollingDays: number
  startsOn: string
  endsOn: string
  showSummary: boolean
  showExpenses: boolean
  showSubscriptions: boolean
  showSettlements: boolean
  showIdentities: boolean
  showNotes: boolean
}

function dateValue(value: string, endOfDay = false): string | undefined {
  if (!value) return undefined
  return new Date(`${value}T${endOfDay ? '23:59:59' : '00:00:00'}`).toISOString()
}

export function serializeShareForm(form: ShareFormState) {
  const payload: Record<string, unknown> = {
    ...form,
    viewerEmails: form.viewerEmails.split(/[\n,;]/).map(value => value.trim()).filter(Boolean),
  }
  delete payload.expiresAt
  delete payload.startsOn
  delete payload.endsOn
  const expiresAt = dateValue(form.expiresAt, true)
  const startsOn = dateValue(form.startsOn)
  const endsOn = dateValue(form.endsOn)
  if (expiresAt) payload.expiresAt = expiresAt
  if (startsOn) payload.startsOn = startsOn
  if (endsOn) payload.endsOn = endsOn
  return payload
}

import type { ShareAccessMode, ShareRangeMode } from '../api/types'
import { fromDateInput, toDateInput } from '../dateInput'
import { browserTimezone } from '../timezone'

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
  showIncomes: boolean
  showRecurringIncome: boolean
  showPendingIncome: boolean
  showExpenses: boolean
  showSubscriptions: boolean
  showSettlements: boolean
  showIdentities: boolean
  showNotes: boolean
  showAttachments: boolean
}

function dateValue(value: string, endOfDay = false): string | undefined {
  if (!value) return undefined
  if (!endOfDay) return fromDateInput(value, browserTimezone())
  const date = new Date(`${value}T00:00:00Z`)
  date.setUTCDate(date.getUTCDate() + 1)
  const nextDay = date.toISOString().slice(0, 10)
  return new Date(new Date(fromDateInput(nextDay, browserTimezone())).getTime() - 1000).toISOString()
}

export function shareDateInput(value?: string): string { return value ? toDateInput(value, browserTimezone()) : '' }

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

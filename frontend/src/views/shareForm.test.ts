import { describe, expect, it } from 'vitest'
import { serializeShareForm, type ShareFormState } from './shareForm'

function form(showAttachments: boolean): ShareFormState {
  return {
    name: 'Shared ledger', accessMode: 'link', password: '', viewerEmails: '', enabled: true,
    expiresAt: '', rangeMode: 'all', rollingDays: 30, startsOn: '', endsOn: '',
    showSummary: true, showIncomes: false, showRecurringIncome: false, showPendingIncome: false,
    showExpenses: true, showSubscriptions: true, showSettlements: false, showIdentities: false,
    showNotes: false, showAttachments,
  }
}

describe('share attachment setting', () => {
  it('serializes the disabled default for legacy and new shares', () => {
    expect(serializeShareForm(form(false))).toMatchObject({ showAttachments: false })
  })

  it('preserves an enabled setting in the API payload', () => {
    expect(serializeShareForm(form(true))).toMatchObject({ showAttachments: true })
  })
})

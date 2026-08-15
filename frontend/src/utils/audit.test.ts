import { describe, expect, it } from 'vitest'
import { auditFieldValue, auditPresentation, visibleAuditFields } from './audit'
import { formatMoney } from '../api/money'

const resolveUser = (id: string) => (id === 'u1' ? `kenny (${id})` : id)

describe('auditFieldValue', () => {
  it('formats amount_minor with the sibling currency instead of a bare integer', () => {
    expect(auditFieldValue('amount_minor', 29800, { currency: 'TWD' }, 'zh-TW', resolveUser)).toBe(formatMoney(29800, 'TWD', 'zh-TW'))
    expect(auditFieldValue('amount_minor', 29800, { currency: 'TWD' }, 'zh-TW', resolveUser)).not.toBe('29800')
  })

  it('falls back to TWD when no sibling currency is present', () => {
    expect(auditFieldValue('amount_minor', 500, undefined, 'zh-TW', resolveUser)).toBe(formatMoney(500, 'TWD', 'zh-TW'))
  })

  it('resolves user-ID fields through the caller-supplied lookup', () => {
    expect(auditFieldValue('from_user_id', 'u1', {}, 'zh-TW', resolveUser)).toBe('kenny (u1)')
    expect(auditFieldValue('to_user_id', 'unknown-id', {}, 'zh-TW', resolveUser)).toBe('unknown-id')
    expect(auditFieldValue('paid_by', 'u1', {}, 'zh-TW', resolveUser)).toBe('kenny (u1)')
  })

  it('leaves unrelated fields to the normal value formatter', () => {
    expect(auditFieldValue('scope', 'one_off', {}, 'zh-TW', resolveUser)).toBe('one_off')
    expect(auditFieldValue('name', '', {}, 'zh-TW', resolveUser)).toBe('—')
  })
})

describe('visibleAuditFields', () => {
  it('hides a standalone currency entry once amount_minor is present, since it is folded into that value', () => {
    expect(visibleAuditFields({ amount_minor: 100, currency: 'TWD', settled_on: '2026-08-01' })).toEqual(['amount_minor', 'settled_on'])
  })

  it('keeps currency visible on its own when there is no amount to pair it with', () => {
    expect(visibleAuditFields({ currency: 'TWD', name: 'x' })).toEqual(['currency', 'name'])
  })

  it('accepts a plain field-name array (for the changes list) the same way', () => {
    expect(visibleAuditFields(['amount_minor', 'currency', 'paid_by'])).toEqual(['amount_minor', 'paid_by'])
  })
})

describe('auditPresentation', () => {
  it('translates every action introduced alongside the currency/detail fixes', () => {
    const regenerated = auditPresentation({ id: '1', actorId: '', groupId: '', scope: 'group', action: 'subscription.occurrence_regenerated', resource: 'expense', resourceId: '', outcome: 'success', createdAt: '' }, 'zh-TW')
    const tempMember = auditPresentation({ id: '2', actorId: '', groupId: '', scope: 'group', action: 'temp_member.created', resource: 'membership', resourceId: '', outcome: 'success', createdAt: '' }, 'zh-TW')
    expect(regenerated.action).toBe('重新產生訂閱入帳紀錄')
    expect(tempMember.action).toBe('新增暫時成員')
  })
})

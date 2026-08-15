// @vitest-environment jsdom
import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import AuditLogList from './AuditLogList.vue'
import { useWorkspaceStore } from '../stores/workspace'
import type { AuditLog } from '../api/types'

vi.mock('../offline/db', () => ({ get: vi.fn(async () => undefined), set: vi.fn(async () => undefined), del: vi.fn(async () => undefined), getAll: vi.fn(async () => []) }))
vi.mock('../offline/snapshot', () => ({ saveSnapshot: vi.fn(async () => {}), loadSnapshot: vi.fn(async () => undefined), saveGroups: vi.fn(async () => {}), loadGroups: vi.fn(async () => undefined), saveCurrencies: vi.fn(async () => {}), loadCurrencies: vi.fn(async () => undefined) }))
vi.mock('../offline/outbox', () => ({ localId: () => 'local-1', isLocalId: () => false, enqueue: vi.fn(async () => ({})), listForUser: vi.fn(async () => []), remove: vi.fn(async () => {}), markFailed: vi.fn(async () => {}), updateTargetId: vi.fn(async () => {}) }))

function settlementLog(): AuditLog {
  return {
    id: 'a1', actorId: 'u1', actorName: 'kenny', groupId: 'g1', scope: 'group',
    action: 'settlement.created', resource: 'settlement', resourceId: 's1', outcome: 'success',
    summary: JSON.stringify({ details: { from_user_id: 'u1', to_user_id: 'u2', amount_minor: 5500, currency: 'TWD', settled_on: '2026-08-16' } }),
    createdAt: '2026-08-16T00:00:00Z',
  }
}

describe('AuditLogList', () => {
  it('resolves user-ID fields to names and folds currency into the formatted amount', async () => {
    setActivePinia(createPinia())
    const workspace = useWorkspaceStore()
    workspace.members.push(
      { id: 'm1', groupId: 'g1', userId: 'u1', role: 'member', createdAt: '', user: { id: 'u1', email: 'kenny@example.com', name: 'kenny', timezone: 'UTC' } },
      { id: 'm2', groupId: 'g1', userId: 'u2', role: 'member', createdAt: '', user: { id: 'u2', email: 'alice@example.com', name: 'alice', timezone: 'UTC' } },
    )

    const wrapper = mount(AuditLogList, { props: { logs: [settlementLog()], pageSize: 25 } })
    await wrapper.find('.audit-row-toggle').trigger('click')
    const text = wrapper.text()

    expect(text).toContain('kenny (u1)')
    expect(text).toContain('alice (u2)')
    // amount_minor must be a formatted amount, not the bare minor-unit integer.
    expect(text).not.toContain('5500')
    // currency is folded into the amount display, not repeated as its own line.
    expect(text.match(/TWD/g)?.length ?? 0).toBeLessThanOrEqual(1)
  })

  it('falls back to the raw ID when the user is not in the current member list', async () => {
    setActivePinia(createPinia())
    const wrapper = mount(AuditLogList, { props: { logs: [settlementLog()], pageSize: 25 } })
    await wrapper.find('.audit-row-toggle').trigger('click')
    expect(wrapper.text()).toContain('u1')
  })
})

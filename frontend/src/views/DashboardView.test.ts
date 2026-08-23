// @vitest-environment jsdom
import { mount } from '@vue/test-utils'
import { nextTick, reactive } from 'vue'
import { describe, expect, it, vi } from 'vitest'

const route = reactive({ params: { groupId: 'g1' }, query: { month: '2026-08' } })
const workspace = reactive({
  summary: null, personalSummary: null, groups: [{ id: 'g1', name: 'Home', currency: 'TWD', timezone: 'UTC' }], currentGroup: { id: 'g1', currency: 'TWD', timezone: 'UTC' }, currentMembership: { userId: 'u1' },
  groupPermissions: [] as string[], members: [{ userId: 'u1', user: { name: 'Owner' } }, { userId: 'u2', user: { name: 'Member' } }],
  settlements: [{ id: 'st1', groupId: 'g1', fromUserId: 'u2', toUserId: 'u1', createdBy: 'u2', amountMinor: 100, currency: 'TWD', baseCurrency: 'TWD', baseAmountMinor: 100, exchangeRate: '1', exchangeRateDate: '', settledOn: '2026-08-01T00:00:00Z', notes: 'Dinner reimbursement', createdAt: '', updatedAt: '' }],
  settlementsMeta: { page: 1, perPage: 25, totalItems: 1, totalPages: 1 }, loading: false, localizedError: '',
  refreshDashboard: vi.fn(async () => {}), loadSettlementsPage: vi.fn(async () => {}), updateSettlement: vi.fn(async () => true), addSettlement: vi.fn(async () => true), deleteSettlement: vi.fn(async () => {}),
})

vi.mock('vue-router', () => ({ useRoute: () => route, useRouter: () => ({ replace: vi.fn(async () => {}) }) }))
vi.mock('../stores/workspace', () => ({ useWorkspaceStore: () => workspace }))
vi.mock('../stores/auth', () => ({ useAuthStore: () => ({ record: { id: 'u1', timezone: 'UTC' } }) }))
vi.mock('../i18n', () => ({ useI18n: () => ({ tr: (key: string) => key, formatDate: (value: string) => value, formatMonth: (value: string) => value }) }))

import DashboardView from './DashboardView.vue'

describe('DashboardView settlement permissions', () => {
  it('labels every settlement party and detail instead of collapsing them into one line', async () => {
    workspace.groupPermissions = ['ledger.settlements.read']
    const wrapper = mount(DashboardView, { global: { stubs: { RouterLink: true, MoneyValue: true, EmptyState: true, SyncBadge: true, AppDrawer: true, ConfirmDialog: true, BaseCombobox: true, MonthNav: true, Pagination: true, PageSizeSelect: true, SettlementFilterBar: true } } })
    await nextTick()
    const row = wrapper.find('.settlement-row')
    expect(row.text()).toContain('fromMember')
    expect(row.text()).toContain('Member')
    expect(row.text()).toContain('toMember')
    expect(row.text()).toContain('Owner')
    expect(row.text()).toContain('amount')
    expect(row.text()).toContain('settlementDate')
    expect(row.text()).toContain('recordedBy')
    expect(row.text()).toContain('notes')
    expect(row.text()).toContain('Dinner reimbursement')
  })

  it('places the record repayment action in the repayment history header when creation is allowed', async () => {
    workspace.groupPermissions = ['ledger.settlements.read', 'ledger.settlements.create']
    const wrapper = mount(DashboardView, { global: { stubs: { RouterLink: true, MoneyValue: true, EmptyState: true, SyncBadge: true, AppDrawer: true, ConfirmDialog: true, BaseCombobox: true, MonthNav: true, Pagination: true, PageSizeSelect: true, SettlementFilterBar: true } } })
    await nextTick()
    expect(wrapper.find('.settlement-history-controls .primary').text()).toBe('recordSettlement')
    expect(wrapper.findAll('.dashboard-grid button').map(button => button.text())).not.toContain('recordSettlement')
  })

  it('hides each action unless its explicit permission and ownership rule allow it', async () => {
    workspace.groupPermissions = ['ledger.settlements.read']
    const wrapper = mount(DashboardView, { global: { stubs: { RouterLink: true, MoneyValue: true, EmptyState: true, SyncBadge: true, AppDrawer: true, ConfirmDialog: true, BaseCombobox: true, MonthNav: true, Pagination: true, PageSizeSelect: true, SettlementFilterBar: true } } })
    await nextTick()
    expect(wrapper.findAll('button[aria-label="edit"]')).toHaveLength(0)
    expect(wrapper.findAll('button[aria-label="delete"]')).toHaveLength(0)

    workspace.groupPermissions = ['ledger.settlements.read', 'ledger.settlements.update', 'ledger.settlements.delete']
    await nextTick()
    expect(wrapper.findAll('button[aria-label="edit"]')).toHaveLength(0)
    expect(wrapper.findAll('button[aria-label="delete"]')).toHaveLength(0)

    workspace.groupPermissions = ['ledger.settlements.read', 'ledger.settlements.update', 'ledger.settlements.delete', 'ledger.settlements.manage']
    await nextTick()
    expect(wrapper.findAll('button[aria-label="edit"]')).toHaveLength(1)
    expect(wrapper.findAll('button[aria-label="delete"]')).toHaveLength(1)
  })
})

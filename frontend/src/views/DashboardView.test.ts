// @vitest-environment jsdom
import { mount } from '@vue/test-utils'
import { nextTick, reactive } from 'vue'
import { describe, expect, it, vi } from 'vitest'

const route = reactive({ params: { groupId: 'g1' }, query: { month: '2026-08' } })
const workspace = reactive({
  summary: null, personalSummary: null, groups: [{ id: 'g1', name: 'Home', currency: 'TWD', timezone: 'UTC' }], currentGroup: { id: 'g1', currency: 'TWD', timezone: 'UTC' }, currentMembership: { userId: 'u1' },
  groupPermissions: [] as string[], members: [{ userId: 'u1', user: { name: 'Owner' } }, { userId: 'u2', user: { name: 'Member' } }],
  settlements: [{ id: 'st1', groupId: 'g1', fromUserId: 'u2', toUserId: 'u1', createdBy: 'u2', amountMinor: 100, currency: 'TWD', baseCurrency: 'TWD', baseAmountMinor: 100, exchangeRate: '1', exchangeRateDate: '', settledOn: '2026-08-01T00:00:00Z', notes: '', createdAt: '', updatedAt: '' }],
  settlementsMeta: { page: 1, perPage: 25, totalItems: 1, totalPages: 1 }, loading: false, localizedError: '',
  refreshDashboard: vi.fn(async () => {}), loadSettlementsPage: vi.fn(async () => {}), updateSettlement: vi.fn(async () => true), addSettlement: vi.fn(async () => true), deleteSettlement: vi.fn(async () => {}),
})

vi.mock('vue-router', () => ({ useRoute: () => route, useRouter: () => ({ replace: vi.fn(async () => {}) }) }))
vi.mock('../stores/workspace', () => ({ useWorkspaceStore: () => workspace }))
vi.mock('../stores/auth', () => ({ useAuthStore: () => ({ record: { id: 'u1', timezone: 'UTC' } }) }))
vi.mock('../i18n', () => ({ useI18n: () => ({ tr: (key: string) => key, formatDate: (value: string) => value, formatMonth: (value: string) => value }) }))

import DashboardView from './DashboardView.vue'

describe('DashboardView settlement permissions', () => {
  it('hides edit and delete actions for another member settlement without write permission', async () => {
    workspace.groupPermissions = []
    const wrapper = mount(DashboardView, { global: { stubs: { RouterLink: true, MoneyValue: true, EmptyState: true, SyncBadge: true, AppDrawer: true, ConfirmDialog: true, BaseCombobox: true, MonthNav: true, Pagination: true, PageSizeSelect: true, SettlementFilterBar: true } } })
    await nextTick()
    expect(wrapper.findAll('button[aria-label="edit"]')).toHaveLength(0)
    expect(wrapper.findAll('button[aria-label="delete"]')).toHaveLength(0)

    workspace.groupPermissions = ['ledger.settlements.write']
    await nextTick()
    expect(wrapper.findAll('button[aria-label="edit"]')).toHaveLength(1)
    expect(wrapper.findAll('button[aria-label="delete"]')).toHaveLength(1)
  })
})

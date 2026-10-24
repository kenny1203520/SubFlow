// @vitest-environment jsdom
import { defineComponent } from 'vue'
import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'

const home = {
  asOf: '2026-10-25T01:30:00+08:00',
  month: '2026-10',
  receivables: [],
  payables: [],
  groups: [],
  actions: [],
  reviews: [],
  upcoming: [],
  monthly: [
    { month: '2026-10', timezone: 'Asia/Taipei', currency: 'TWD', incomeReceivedMinor: 0, expensePaidMinor: 0, settlementReceivedMinor: 0, settlementPaidMinor: 0, beneficialIncomeMinor: 0, personalExpenseShareMinor: 0 },
    { groupId: 'group-1', groupName: 'Family', month: '2026-09', timezone: 'America/Los_Angeles', currency: 'USD', incomeReceivedMinor: 0, expensePaidMinor: 0, settlementReceivedMinor: 0, settlementPaidMinor: 0, beneficialIncomeMinor: 0, personalExpenseShareMinor: 0 },
  ],
}

vi.mock('../api/client', () => ({
  ApiClient: class {
    async get() { return { data: home } }
    async post() { return { data: {} } }
  },
}))
vi.mock('../stores/auth', () => ({ useAuthStore: () => ({ token: 'test', name: 'Taylor', record: { id: 'user-1', name: 'Taylor', timezone: 'Asia/Taipei' }, logout: vi.fn() }) }))
vi.mock('../i18n', () => ({ useI18n: () => ({ tr: (key: string, params?: { date?: string }) => key === 'asOf' ? `Data through ${params?.date}` : key, formatDate: () => 'Oct 25, 2026, 1:30 AM' }) }))

import HomeView from './HomeView.vue'

const RouterLinkStub = defineComponent({
  props: { to: { type: [String, Object], required: false } },
  template: '<a class="router-link-stub" :data-to="JSON.stringify(to)"><slot /></a>',
})

describe('HomeView monthly analysis links', () => {
  it('opens analysis in the matching scope and group-local month', async () => {
    const wrapper = mount(HomeView, { global: { stubs: { RouterLink: RouterLinkStub, MoneyValue: true, EmptyState: true } } })
    await vi.waitFor(() => expect(wrapper.findAll('.home-monthly-analysis')).toHaveLength(2))

    const targets = wrapper.findAll('.home-monthly-analysis').map(link => JSON.parse(link.attributes('data-to') || '{}'))
    expect(targets[0]).toEqual({ name: 'dashboard', query: { scope: 'personal', month: '2026-10' } })
    expect(targets[1]).toEqual({ name: 'group-overview', params: { groupId: 'group-1' }, query: { month: '2026-09' } })
    expect(wrapper.text()).toContain('Oct 25, 2026, 1:30 AM')
  })
})

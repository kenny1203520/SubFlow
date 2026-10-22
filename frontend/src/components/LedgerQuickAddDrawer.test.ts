// @vitest-environment jsdom
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { ref } from 'vue'
import { pb } from '../pocketbase'
import { useWorkspaceStore } from '../stores/workspace'
import type { Expense } from '../api/types'
import LedgerQuickAddDrawer from './LedgerQuickAddDrawer.vue'

vi.mock('../i18n', () => ({ useI18n: () => ({ tr: (key: string) => key, locale: ref('en'), formatDate: (value: string) => value }) }))

const expense: Expense = { id: 'expense-1', title: 'Original', category: '', amountMinor: 12300, currency: 'TWD', baseCurrency: 'TWD', baseAmountMinor: 12300, exchangeRate: '1', exchangeRateDate: '2026-10-22T00:00:00Z', rateMode: 'automatic', paidBy: 'user-1', incurredOn: '2026-10-22T00:00:00Z', notes: '', splitMode: 'equal', splits: [], createdAt: '', updatedAt: '' }

beforeEach(() => {
  setActivePinia(createPinia())
  pb.authStore.save('', { id: 'user-1', collectionId: 'users', collectionName: 'users', name: 'Test user', timezone: 'UTC', defaultCurrency: 'TWD' })
})
afterEach(() => { pb.authStore.clear(); vi.restoreAllMocks() })

async function editor() {
  const workspace = useWorkspaceStore()
  vi.spyOn(workspace, 'loadCategories').mockResolvedValue([])
  const wrapper = mount(LedgerQuickAddDrawer, {
    props: { open: false, personal: true, date: '2026-10-22', editRecord: expense, editKind: 'expense' },
    global: { stubs: { AppDrawer: { template: '<div><slot /></div>' }, CategorySelect: true, CurrencySelect: true, BaseCombobox: true, ConversionPreview: true } },
  })
  await wrapper.setProps({ open: true })
  await flushPromises()
  await wrapper.find('input[type="text"]').setValue('Updated')
  return { workspace, wrapper, save: wrapper.get('button.primary') }
}

describe('ledger saves during background reads', () => {
  it('keeps a valid edit available while a background refresh is pending', async () => {
    const { workspace, wrapper, save } = await editor()
    const update = vi.spyOn(workspace, 'updateExpense').mockResolvedValue(true)
    workspace.busy.personal = 1
    await flushPromises()
    expect(save.attributes('disabled')).toBeUndefined()
    await wrapper.get('form').trigger('submit')
    await flushPromises()
    expect(update).toHaveBeenCalledWith('expense-1', expect.objectContaining({ title: 'Updated', amountMinor: 12300 }))
    expect(wrapper.emitted('saved')).toHaveLength(1)
    wrapper.unmount()
  })

  it('blocks duplicate submits until its own write completes', async () => {
    const { workspace, wrapper, save } = await editor()
    let complete: ((value: boolean) => void) | undefined
    const update = vi.spyOn(workspace, 'updateExpense').mockImplementation(() => new Promise<boolean>(resolve => { complete = resolve }))
    await wrapper.get('form').trigger('submit')
    expect(save.attributes('disabled')).toBeDefined()
    await wrapper.get('form').trigger('submit')
    expect(update).toHaveBeenCalledTimes(1)
    complete?.(true)
    await flushPromises()
    expect(wrapper.emitted('saved')).toHaveLength(1)
    wrapper.unmount()
  })

  it('retains the edit and allows retry after a rejected write', async () => {
    const { workspace, wrapper, save } = await editor()
    vi.spyOn(workspace, 'updateExpense').mockResolvedValue(false)
    await wrapper.get('form').trigger('submit')
    await flushPromises()
    expect(wrapper.emitted('saved')).toBeUndefined()
    expect(wrapper.get<HTMLInputElement>('input[type="text"]').element.value).toBe('Updated')
    expect(wrapper.get('.notice.danger').text()).not.toBe('')
    expect(save.attributes('disabled')).toBeUndefined()
    wrapper.unmount()
  })
})

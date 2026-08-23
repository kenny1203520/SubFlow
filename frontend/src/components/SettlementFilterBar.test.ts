// @vitest-environment jsdom
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import SettlementFilterBar from './SettlementFilterBar.vue'

const emptyFilters = { memberId:'', from:'', to:'', sort:'-settled_on' }
const members = [{ userId:'u1', label:'Alice' }, { userId:'u2', label:'Bob' }]

describe('SettlementFilterBar', () => {
  it('emits updated filters and applies the requested query', async () => {
    const wrapper = mount(SettlementFilterBar, { props: { modelValue: emptyFilters, members } })
    const from = wrapper.find('input[type="date"]')
    await from.setValue('2026-08-01')
    expect(wrapper.emitted('update:modelValue')?.[0]).toEqual([{ ...emptyFilters, from:'2026-08-01' }])
    await wrapper.find('form').trigger('submit')
    expect(wrapper.emitted('apply')).toHaveLength(1)
  })

  it('clears every filter with the standard reset action', async () => {
    const wrapper = mount(SettlementFilterBar, { props: { modelValue: { ...emptyFilters, memberId:'u1', from:'2026-08-01' }, members } })
    const buttons = wrapper.findAll('button')
    await buttons.at(-1)!.trigger('click')
    expect(wrapper.emitted('update:modelValue')?.at(-1)).toEqual([emptyFilters])
    expect(wrapper.emitted('reset')).toHaveLength(1)
  })
})

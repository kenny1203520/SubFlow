// @vitest-environment jsdom
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import AuditFilterBar from './AuditFilterBar.vue'

const emptyFilters = { q:'', action:'', resource:'', outcome:'', from:'', to:'' }

describe('AuditFilterBar', () => {
  it('emits updated filters and applies the requested query', async () => {
    const wrapper = mount(AuditFilterBar, { props: { modelValue: emptyFilters } })
    const search = wrapper.find('input[type="text"]')
    await search.setValue('settings')
    expect(wrapper.emitted('update:modelValue')?.[0]).toEqual([{ ...emptyFilters, q:'settings' }])
    await wrapper.find('form').trigger('submit')
    expect(wrapper.emitted('apply')).toHaveLength(1)
  })

  it('clears every filter with the standard reset action', async () => {
    const wrapper = mount(AuditFilterBar, { props: { modelValue: { ...emptyFilters, action:'system.settings.updated', from:'2026-08-01' } } })
    const buttons = wrapper.findAll('button')
    await buttons.at(-1)!.trigger('click')
    expect(wrapper.emitted('update:modelValue')?.at(-1)).toEqual([emptyFilters])
    expect(wrapper.emitted('reset')).toHaveLength(1)
  })
})

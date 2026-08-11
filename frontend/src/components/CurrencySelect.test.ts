// @vitest-environment jsdom
import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import CurrencySelect from './CurrencySelect.vue'

describe('CurrencySelect', () => {
  it('searches localized currency labels and emits the selected ISO code', async () => {
    const wrapper = mount(CurrencySelect, {
      props: {
        modelValue: 'TWD',
        currencies: [{ code: 'TWD', digits: 2 }, { code: 'USD', digits: 2 }, { code: 'JPY', digits: 0 }],
      },
    })
    await wrapper.get('.currency-trigger').trigger('click')
    await wrapper.get('.currency-search input').setValue('usd')
    expect(wrapper.findAll('.currency-option')).toHaveLength(1)
    expect(wrapper.text()).toContain('USD')
    await wrapper.get('.currency-option').trigger('click')
    expect(wrapper.emitted('update:modelValue')).toEqual([['USD']])
  })
})

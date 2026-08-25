// @vitest-environment jsdom
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import SharePagination from './SharePagination.vue'

describe('share pagination', () => {
  it('keeps previous, next and page-size controls on the same state contract', async () => {
    const wrapper = mount(SharePagination, { props: { page: 2, totalPages: 4, pageSize: 5 } })
    const buttons = wrapper.findAll('button')
    await buttons[0].trigger('click')
    await buttons[buttons.length - 1].trigger('click')
    expect(wrapper.emitted('page')).toEqual([[1], [3]])
    await wrapper.find('select').setValue('10')
    expect(wrapper.emitted('update:pageSize')).toEqual([[10]])
  })

  it('disables navigation at the boundaries', () => {
    const wrapper = mount(SharePagination, { props: { page: 1, totalPages: 1, pageSize: 25 } })
    const buttons = wrapper.findAll('button')
    expect(buttons[0].attributes('disabled')).toBeDefined()
    expect(buttons[buttons.length - 1].attributes('disabled')).toBeDefined()
  })
})

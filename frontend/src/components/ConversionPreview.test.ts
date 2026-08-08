// @vitest-environment jsdom
import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'

const quoteRate = vi.fn(async () => ({ rate: '31.25', provider: 'test', effectiveDate: '2026-08-08T00:00:00Z', stale: false }))
vi.mock('../stores/workspace', () => ({ useWorkspaceStore: () => ({ quoteRate }) }))

import ConversionPreview from './ConversionPreview.vue'

describe('ConversionPreview', () => {
  it('shows the same-currency conversion immediately', async () => {
    const wrapper = mount(ConversionPreview, { props: { from:'TWD', to:'TWD', amount:'120', date:'2026-08-08', mode:'automatic', manualRate:'' } })
    expect(wrapper.text()).toContain('1')
    expect(wrapper.text()).toContain('無需換匯')
    expect(wrapper.emitted('validity')?.at(-1)).toEqual([true])
  })

  it('shows a manual conversion and rejects an invalid manual rate', async () => {
    const wrapper = mount(ConversionPreview, { props: { from:'USD', to:'TWD', amount:'10', date:'2026-08-08', mode:'manual', manualRate:'31.5' } })
    expect(wrapper.text()).toContain('USD')
    expect(wrapper.text()).toContain('31.5')
    expect(wrapper.text()).toContain('TWD')
    expect(wrapper.text()).toContain('315')
    await wrapper.setProps({ manualRate:'0' })
    expect(wrapper.text()).toContain('請輸入大於 0 的手動匯率')
    expect(wrapper.emitted('validity')?.at(-1)).toEqual([false])
  })

  it('recalculates the manual conversion as the rate changes', async () => {
    const wrapper = mount(ConversionPreview, { props: { from:'USD', to:'TWD', amount:'10', date:'2026-08-08', mode:'manual', manualRate:'31.5' } })
    expect(wrapper.text()).toContain('315')
    await wrapper.setProps({ manualRate:'32' })
    expect(wrapper.text()).toContain('320')
    expect(wrapper.emitted('validity')?.at(-1)).toEqual([true])
  })
})

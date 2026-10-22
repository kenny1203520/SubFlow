// @vitest-environment jsdom
import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import LedgerDateNav from './LedgerDateNav.vue'

vi.mock('../i18n', () => ({ useI18n: () => ({ tr: (key: string) => key, locale: ref('en') }) }))

describe('LedgerDateNav', () => {
  it('keeps the date controls balanced and emits calendar-safe date changes', async () => {
    const wrapper = mount(LedgerDateNav, { props: { date: '2026-01-01', today: '2026-01-02', timezone: 'Asia/Taipei' } })
    const buttons = wrapper.findAll('button')
    await buttons[0].trigger('click')
    expect(wrapper.emitted('change')?.[0]).toEqual(['2025-12-31'])
    expect((wrapper.find('input[type="date"]').element as HTMLInputElement).value).toBe('2026-01-01')
    expect(wrapper.text()).toContain('Asia/Taipei · 2026-01-01')
  })

  it('emits the selected day and reset day independently from API state', async () => {
    const wrapper = mount(LedgerDateNav, { props: { date: '2026-04-30', today: '2026-05-01', timezone: 'UTC' } })
    await wrapper.find('input[type="date"]').setValue('2026-05-03')
    await wrapper.find('.today-button').trigger('click')
    expect(wrapper.emitted('change')).toEqual([['2026-05-03'], ['2026-05-01']])
  })
})

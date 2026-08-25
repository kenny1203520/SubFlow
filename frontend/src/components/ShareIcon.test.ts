// @vitest-environment jsdom
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import ShareIcon from './ShareIcon.vue'

describe('share icon', () => {
  it('renders the supported share glyphs as SVG paths', () => {
    const names = ['arrow-up-right', 'lock', 'expense', 'subscription', 'settlement', 'empty', 'alert'] as const

    for (const name of names) {
      const wrapper = mount(ShareIcon, { props: { name } })
      expect(wrapper.find('svg').exists()).toBe(true)
      expect(wrapper.find('svg').findAll('path, rect, circle').length).toBeGreaterThan(0)
    }
  })
})

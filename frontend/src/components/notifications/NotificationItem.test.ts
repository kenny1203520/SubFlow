// @vitest-environment jsdom
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import NotificationItem from './NotificationItem.vue'

describe('NotificationItem', () => {
  it('renders notification state as text and leaves actions to its parent', () => {
    const wrapper = mount(NotificationItem, {
      props: {
        title: 'Group invitation',
        category: 'Tasks',
        status: 'Needs action',
        readState: 'Unread',
        createdAt: '2026-10-22T02:00:00Z',
        formattedDate: 'Today, 10:00',
        unread: true,
      },
      slots: { actions: '<button type="button">Accept</button>' },
    })

    expect(wrapper.text()).toContain('Group invitation')
    expect(wrapper.text()).toContain('Unread')
    expect(wrapper.text()).toContain('Needs action')
    expect(wrapper.get('time').attributes('datetime')).toBe('2026-10-22T02:00:00Z')
    expect(wrapper.get('button').text()).toBe('Accept')
    expect(wrapper.classes()).toContain('is-unread')
  })
})

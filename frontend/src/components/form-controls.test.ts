// @vitest-environment jsdom
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import BaseInput from './BaseInput.vue'
import PasswordField from './PasswordField.vue'

describe('form controls', () => {
  it('links input help and errors to its label', async () => {
    const wrapper = mount(BaseInput, { props: { modelValue: '', label: 'Email', type: 'email', required: true, help: 'Use your work email', error: 'Email is required' } })
    const input = wrapper.find('input')
    expect(input.attributes('type')).toBe('email')
    expect(input.attributes('required')).toBeDefined()
    expect(input.attributes('aria-invalid')).toBe('true')
    expect(input.attributes('aria-describedby')).toContain('-help')
    await input.setValue('user@example.com')
    expect(wrapper.emitted('update:modelValue')?.[0]).toEqual(['user@example.com'])
  })

  it('toggles passwords without changing the value', async () => {
    const wrapper = mount(PasswordField, { props: { modelValue: 'secret-value', label: 'Password', autocomplete: 'current-password' } })
    expect(wrapper.find('input').attributes('type')).toBe('password')
    await wrapper.find('button').trigger('click')
    expect(wrapper.find('input').attributes('type')).toBe('text')
    expect((wrapper.find('input').element as HTMLInputElement).value).toBe('secret-value')
  })
})

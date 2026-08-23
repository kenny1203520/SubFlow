// @vitest-environment jsdom
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import CaptchaChallenge from './CaptchaChallenge.vue'
import { useSetupStore } from '../stores/setup'

const flows = { register: { enabled: true, trigger: 'load' as const, mode: 'interactive' as const }, passwordReset: { enabled: true, trigger: 'load' as const, mode: 'interactive' as const }, otpRequest: { enabled: true, trigger: 'load' as const, mode: 'interactive' as const }, login: { enabled: true, trigger: 'load' as const, mode: 'interactive' as const } }

describe('CaptchaChallenge Turnstile actions', () => {
  const render = vi.fn()

  beforeEach(() => {
    setActivePinia(createPinia())
    const setup = useSetupStore()
    setup.status = { initialized: true, captchaProvider: 'turnstile', captchaSiteKey: 'site-key', captchaFlows: flows }
    render.mockReset()
    render.mockReturnValue('widget-1')
    ;(window as any).turnstile = { render, remove: vi.fn() }
    const script = document.createElement('script')
    script.src = 'https://challenges.cloudflare.com/turnstile/v0/api.js?render=explicit'
    document.head.appendChild(script)
  })

  afterEach(() => {
    document.head.querySelectorAll('script[src*="challenges.cloudflare.com/turnstile"]').forEach(node => node.remove())
    delete (window as any).turnstile
  })

  it.each([
    ['register', 'register'],
    ['passwordReset', 'password_reset'],
    ['otpRequest', 'otp_request'],
    ['login', 'login'],
  ])('renders %s with the %s action', async (flow, action) => {
    const wrapper = mount(CaptchaChallenge, { props: { flow } })
    await flushPromises()
    expect(render).toHaveBeenCalledWith(expect.any(HTMLElement), expect.objectContaining({ sitekey: 'site-key', action }))
    wrapper.unmount()
  })

  it('clears the token and pending state when Turnstile expires', async () => {
    const wrapper = mount(CaptchaChallenge, { props: { flow: 'login' } })
    await flushPromises()
    const options = render.mock.calls[0][1]
    options.callback('fresh-token')
    options['expired-callback']()
    expect(wrapper.emitted('update:modelValue')).toEqual([[''], ['fresh-token'], ['']])
    wrapper.unmount()
  })
})

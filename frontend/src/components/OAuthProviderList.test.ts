// @vitest-environment jsdom
import { mount } from '@vue/test-utils'
import { afterEach, describe, expect, it } from 'vitest'
import { useI18n } from '../i18n'
import OAuthProviderList from './OAuthProviderList.vue'
import OAuthProviderLogo from './OAuthProviderLogo.vue'
import type { OAuthProviderInfo } from '../stores/auth'

const provider = (name: string, displayName = name): OAuthProviderInfo => ({
  name,
  displayName,
})

const { setLocale } = useI18n()

afterEach(() => setLocale('zh-TW'))

describe('OAuth provider list', () => {
  it('renders no provider buttons for an empty provider set', () => {
    const wrapper = mount(OAuthProviderList, { props: { providers: [] } })
    expect(wrapper.findAll('button')).toHaveLength(0)
  })

  it('keeps a single provider named and selectable', async () => {
    const wrapper = mount(OAuthProviderList, { props: { providers: [provider('authentik')] } })
    await wrapper.get('button').trigger('click')
    expect(wrapper.text()).toContain('authentik')
    expect(wrapper.emitted('select')).toEqual([['authentik']])
    expect(wrapper.get('[role="group"]').attributes('aria-label')).toBe('或使用第三方帳號登入')
  })

  it('shows five providers with their labels and keeps the selected provider ID', async () => {
    setLocale('en')
    const providers = [
      provider('authentik', 'Company identity with a deliberately long provider name'),
      provider('spotify', 'Spotify'),
      provider('google', 'Google'),
      provider('github', 'GitHub'),
      provider('discord', 'Discord'),
    ]
    const wrapper = mount(OAuthProviderList, { props: { providers } })
    const buttons = wrapper.findAll('button')

    expect(buttons).toHaveLength(5)
    expect(wrapper.text()).toContain('Company identity with a deliberately long provider name')
    expect(wrapper.text()).toContain('Continue with Google')
    await buttons[1].trigger('click')
    expect(wrapper.emitted('select')).toEqual([['spotify']])
  })

  it('disables all provider actions while authentication is busy', () => {
    const wrapper = mount(OAuthProviderList, {
      props: { providers: [provider('google'), provider('github')], busy: true },
    })
    expect(wrapper.findAll('button').every(button => button.attributes('disabled') !== undefined)).toBe(true)
  })
})

describe('OAuth provider logo', () => {
  it('uses PocketBase SVG logos as encoded image sources', () => {
    const wrapper = mount(OAuthProviderLogo, {
      props: { provider: { ...provider('google'), logo: '<svg viewBox="0 0 1 1"><path /></svg>' } },
    })
    expect(wrapper.find('img').attributes('src')).toContain('data:image/svg+xml,%3Csvg')
    expect(wrapper.find('img').attributes('src')).toContain('%3Cpath%20%2F%3E')
  })

  it('uses a generic provider mark when no usable logo is supplied', () => {
    const wrapper = mount(OAuthProviderLogo, { props: { provider: provider('custom-oidc') } })
    expect(wrapper.find('img').exists()).toBe(false)
    expect(wrapper.find('svg').exists()).toBe(true)
  })
})

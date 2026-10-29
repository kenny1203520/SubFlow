// @vitest-environment jsdom
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { createMemoryHistory, createRouter } from 'vue-router'
import AdminView from './AdminView.vue'
import { locale } from '../i18n'

vi.mock('../stores/auth', () => ({ useAuthStore: () => ({ token: 'test-token', logout: vi.fn(), permissions: ['system.settings.manage', 'system.roles.manage', 'system.users.assign', 'system.audit.read'] }) }))
vi.mock('../stores/workspace', () => ({ useWorkspaceStore: () => ({ currencies: [] }) }))

function response(data: unknown, status = 200) {
  return new Response(JSON.stringify(status === 200 ? { data } : { error: { code: 'not_found', message: 'Not found' } }), { status, headers: { 'Content-Type': 'application/json' } })
}
async function render(path='/admin/notification-emails') {
  const router = createRouter({ history: createMemoryHistory(), routes: [{ path: '/admin/:section', name: 'admin-section', component: AdminView }, { path: '/admin', name: 'admin', component: AdminView }] })
  await router.push(path)
  await router.isReady()
  const wrapper = mount(AdminView, { global: { plugins: [router] } })
  await flushPromises()
  return wrapper
}

beforeEach(() => { locale.value = 'en' })
afterEach(() => vi.unstubAllGlobals())

describe('admin notification email states', () => {
  it('loads only the selected section and accepts an empty response', async () => {
    const fetch = vi.fn<(url: string, options?: RequestInit) => Promise<Response>>().mockResolvedValue(response(null))
    vi.stubGlobal('fetch', fetch)
    const wrapper = await render()
    expect(fetch).toHaveBeenCalledTimes(1)
    expect(fetch.mock.calls[0]?.[0]).toBe('/api/subflow/v1/system/notification-emails/failed?page=1&perPage=25&sort=-created')
    expect(wrapper.text()).toContain('No failed notification emails')
    expect(wrapper.find('[role="alert"]').exists()).toBe(false)
    wrapper.unmount()
  })

  it('shows a recoverable error instead of a successful empty state on a 404', async () => {
    const fetch = vi.fn().mockResolvedValueOnce(response(null, 404)).mockResolvedValueOnce(response([]))
    vi.stubGlobal('fetch', fetch)
    const wrapper = await render()
    expect(wrapper.find('[role="alert"]').text()).toContain('Notification email management is unavailable')
    expect(wrapper.text()).not.toContain('No failed notification emails')
    await wrapper.find('[role="alert"] button').trigger('click')
    await flushPromises()
    expect(wrapper.find('[role="alert"]').exists()).toBe(false)
    expect(wrapper.text()).toContain('No failed notification emails')
    wrapper.unmount()
  })

  it('queues only once while a retry is pending and reports success after the write succeeds', async () => {
    let complete: (value: Response) => void = () => {}
    const pending = new Promise<Response>(resolve => { complete = resolve })
    const fetch = vi.fn().mockResolvedValueOnce(response([{ id: 'job-1', attempts: 5, lastError: 'mail_delivery_failed' }])).mockReturnValueOnce(pending).mockResolvedValueOnce(response([]))
    vi.stubGlobal('fetch', fetch)
    const wrapper = await render()
    const button = wrapper.find('.failed-email-row button')
    await button.trigger('click')
    expect(button.attributes('disabled')).toBeDefined()
    await button.trigger('click')
    expect(fetch).toHaveBeenCalledTimes(2)
    expect(wrapper.text()).not.toContain('The notification email has been queued again.')
    complete(response({ queued: true }))
    await flushPromises()
    expect(wrapper.find('.failed-email-row').exists()).toBe(false)
    expect(wrapper.find('[role="status"]').text()).toContain('The notification email has been queued again.')
    wrapper.unmount()
  })
})

describe('admin CAPTCHA provider compatibility',()=>{
  it('allows selecting a provider when older settings responses omit provider profiles',async()=>{
    const legacySettings={initialized:true,siteName:'SubFlow',defaultTimezone:'UTC',defaultCurrency:'TWD',attachmentDefaultMaxSize:20971520,attachmentDefaultMaxCount:10,attachmentLimitMaxSize:20971520,attachmentLimitMaxCount:10,allowPasswordRegistration:true,allowOidcRegistration:true,captchaProvider:'',captchaFlows:{register:{enabled:false,trigger:'load',mode:'interactive'},passwordReset:{enabled:false,trigger:'load',mode:'interactive'},otpRequest:{enabled:false,trigger:'load',mode:'interactive'},login:{enabled:false,trigger:'load',mode:'interactive'}}}
    const fetch=vi.fn().mockResolvedValue(response(legacySettings))
    vi.stubGlobal('fetch',fetch)
    const wrapper=await render('/admin/settings')
    await wrapper.find('.combobox-trigger').trigger('click')
    const option=wrapper.findAll('.combobox-option').find(button=>button.text().includes('Cloudflare Turnstile'))
    expect(option).toBeTruthy()
    await option!.trigger('click')
    expect(wrapper.text()).not.toContain('Page failed to load')
    expect(wrapper.find('[role="alert"]').exists()).toBe(false)
    wrapper.unmount()
  })
})

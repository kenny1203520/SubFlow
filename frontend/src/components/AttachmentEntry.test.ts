// @vitest-environment jsdom
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import AttachmentEntry from './AttachmentEntry.vue'
import { locale } from '../i18n'

vi.mock('../stores/auth', () => ({ useAuthStore: () => ({ token: 'token', logout: vi.fn() }) }))
let stored: object[] = []
let status = 201
let responseCode = 'forbidden'
let failRefresh = false
class UploadXHR {
 status = 0
 responseText = ''
 upload = { onprogress: null }
 onload?: () => void
 onerror?: () => void
 onabort?: () => void
 open = vi.fn()
 setRequestHeader = vi.fn()
 send() {
  this.status = status
  this.responseText = JSON.stringify({ error: { code: responseCode } })
  if (status === 201) stored = [{ id: 'attachment1', fileName: 'receipt.pdf', size: 100, mediaType: 'application/pdf' }]
  this.onload?.()
 }
 abort() { this.onabort?.() }
}

const wrappers: ReturnType<typeof mount>[] = []
async function openEntry() {
 const wrapper = mount(AttachmentEntry, { props: { targetType: 'expense', targetId: 'expense1', online: true }, attachTo: document.body })
 wrappers.push(wrapper)
 await wrapper.get('button').trigger('click')
 await flushPromises()
 return wrapper
}
async function select() {
 const input = document.querySelector<HTMLInputElement>('input[type=file]')!
 Object.defineProperty(input, 'files', { configurable: true, value: [new File(['%PDF-1.7'], 'receipt.pdf')] })
 input.dispatchEvent(new Event('change', { bubbles: true }))
 await flushPromises()
}
describe('existing expense attachments', () => {
 beforeEach(() => {
  locale.value = 'en'; stored = []; status = 201; failRefresh = false
  vi.stubGlobal('XMLHttpRequest', UploadXHR)
  HTMLDialogElement.prototype.showModal=function(){this.setAttribute('open','')}
  HTMLDialogElement.prototype.close=function(){this.removeAttribute('open')}
  vi.stubGlobal('fetch', vi.fn(async () => {
   if (failRefresh && stored.length) throw Error('offline')
   return { ok: true, status: 200, json: async () => ({ data: { items: stored, limits: { maxSize: 1024, maxCount: 10 }, access: { canRead: true, canUpload: true, canDelete: true } } }) }
  }))
 })
 afterEach(() => { wrappers.splice(0).forEach(w => w.unmount()); document.body.innerHTML = ''; vi.unstubAllGlobals() })
 it('shows success and content and reloads after closing and reopening', async () => {
  const wrapper = await openEntry(); await select()
  expect(document.body.textContent).toContain('receipt.pdf')
  expect(document.querySelector('[role=status]')?.textContent).toContain('Uploaded 1')
  ;(document.querySelector('button[aria-label=Close]') as HTMLButtonElement).click(); await flushPromises()
  await wrapper.get('button').trigger('click'); await flushPromises()
  expect(document.body.textContent).toContain('receipt.pdf')
 })
 it('distinguishes stored upload from a failed refresh', async () => {
  await openEntry(); failRefresh = true; await select()
  expect(document.querySelector('[role=status]')?.textContent).toContain('Uploaded 1')
  expect(document.body.textContent).toContain('refresh')
 })
 it('keeps files for retry and clears a permission failure after success', async () => {
  await openEntry(); status = 403; await select()
  expect(document.querySelector('[role=alert]')?.textContent).toContain('permission')
  status = 201
  const retry = [...document.querySelectorAll('button')].find(b => b.textContent === 'Retry')!
  retry.click(); await flushPromises()
  expect(document.querySelector('[role=alert]')).toBeNull()
  expect(document.body.textContent).toContain('receipt.pdf')
 })
})

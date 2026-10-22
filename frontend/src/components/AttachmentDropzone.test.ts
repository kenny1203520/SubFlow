// @vitest-environment jsdom
import { afterEach, beforeEach, describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import AttachmentDropzone from './AttachmentDropzone.vue'
import { locale } from '../i18n'

function file(name: string) {
  return new File(['content'], name, { type: 'application/pdf' })
}
function drag(type: string, files: File[] = [], types = ['Files']) {
  const event = new Event(type, { bubbles: true, cancelable: true })
  Object.defineProperty(event, 'dataTransfer', { value: { types, files, dropEffect: 'none' } })
  return event
}

describe('AttachmentDropzone', () => {
  beforeEach(() => { locale.value = 'en' })
  afterEach(() => { document.body.innerHTML = '' })

  it('emits every dropped file and restores the idle state', async () => {
    const wrapper = mount(AttachmentDropzone)
    const target = wrapper.get('.attachment-dropzone-wrap')
    await target.element.dispatchEvent(drag('dragenter'))
    expect(target.classes()).toContain('is-dragging')
    const event = drag('drop', [file('a.pdf'), file('b.pdf')])
    await target.element.dispatchEvent(event)
    expect(event.defaultPrevented).toBe(true)
    expect(wrapper.emitted('filesSelected')?.[0]?.[0]).toEqual([expect.any(File), expect.any(File)])
    expect(target.classes()).not.toContain('is-dragging')
  })

  it('keeps highlight while the pointer moves between descendants', async () => {
    const wrapper = mount(AttachmentDropzone)
    const target = wrapper.get('.attachment-dropzone-wrap')
    const label = wrapper.get('label')
    await target.element.dispatchEvent(drag('dragenter'))
    await label.element.dispatchEvent(drag('dragenter'))
    await target.element.dispatchEvent(drag('dragleave'))
    expect(target.classes()).toContain('is-dragging')
    await label.element.dispatchEvent(drag('dragleave'))
    expect(target.classes()).not.toContain('is-dragging')
  })

  it('ignores non-file drags and blocks file navigation when disabled', async () => {
    const wrapper = mount(AttachmentDropzone, { props: { disabled: true, disabledMessage: 'Offline' } })
    const target = wrapper.get('.attachment-dropzone-wrap')
    const linkDrag = drag('dragover', [], ['text/uri-list'])
    await target.element.dispatchEvent(linkDrag)
    expect(linkDrag.defaultPrevented).toBe(false)
    const fileDrop = drag('drop', [file('a.pdf')])
    await target.element.dispatchEvent(fileDrop)
    expect(fileDrop.defaultPrevented).toBe(true)
    expect(wrapper.emitted('filesSelected')).toBeUndefined()
    expect(wrapper.text()).toContain('Offline')
    expect((wrapper.get('input[type=file]').element as HTMLInputElement).disabled).toBe(true)
  })

  it('supports multiple selection and can select the same filename again', async () => {
    const wrapper = mount(AttachmentDropzone)
    const input = wrapper.get('input[type=file]')
    Object.defineProperty(input.element, 'files', { configurable: true, value: [file('a.pdf'), file('b.pdf')] })
    await input.trigger('change')
    expect(wrapper.emitted('filesSelected')?.[0]?.[0]).toHaveLength(2)
    expect((input.element as HTMLInputElement).value).toBe('')
    Object.defineProperty(input.element, 'files', { configurable: true, value: [file('a.pdf')] })
    await input.trigger('change')
    expect(wrapper.emitted('filesSelected')).toHaveLength(2)
  })
})

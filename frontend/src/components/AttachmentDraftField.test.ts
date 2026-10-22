// @vitest-environment jsdom
import { afterEach, beforeEach, describe, expect, it } from 'vitest'
import { defineComponent, ref } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import AttachmentDraftField from './AttachmentDraftField.vue'
import { locale } from '../i18n'

const Harness = defineComponent({
  components: { AttachmentDraftField },
  setup() { return { files: ref<File[]>([]) } },
  template: '<AttachmentDraftField v-model="files" />',
})
function file(name: string) { return new File(['content'], name, { type: 'application/pdf' }) }

describe('AttachmentDraftField', () => {
  beforeEach(() => { locale.value = 'en' })
  afterEach(() => { document.body.innerHTML = '' })

  it('appends repeated selections, lists names and sizes, and removes only the selected file', async () => {
    const wrapper = mount(Harness)
    const input = wrapper.get('input[type=file]')
    Object.defineProperty(input.element, 'files', { configurable: true, value: [file('same.pdf'), file('other.pdf')] })
    await input.trigger('change')
    await flushPromises()
    Object.defineProperty(input.element, 'files', { configurable: true, value: [file('same.pdf')] })
    await input.trigger('change')
    await flushPromises()
    expect(wrapper.findAll('.attachment-draft-list li')).toHaveLength(3)
    expect(wrapper.text()).toContain('3 files selected')
    expect(wrapper.text()).toContain('same.pdf')
    expect(wrapper.text()).toContain('0.00 MB')
    await wrapper.get('button[aria-label="Remove other.pdf from pending uploads"]').trigger('click')
    await flushPromises()
    expect(wrapper.findAll('.attachment-draft-list li')).toHaveLength(2)
    expect(wrapper.text()).toContain('2 files selected')
  })

  it('uses the same draft list for dropped files', async () => {
    const wrapper = mount(Harness)
    const event = new Event('drop', { bubbles: true, cancelable: true })
    Object.defineProperty(event, 'dataTransfer', { value: { types: ['Files'], files: [file('dropped.pdf')], dropEffect: 'none' } })
    await wrapper.get('.attachment-dropzone-wrap').element.dispatchEvent(event)
    await flushPromises()
    expect(wrapper.findAll('.attachment-draft-list li')).toHaveLength(1)
    expect(wrapper.text()).toContain('dropped.pdf')
  })
})

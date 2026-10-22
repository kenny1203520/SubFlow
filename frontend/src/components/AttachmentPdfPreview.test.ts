// @vitest-environment jsdom
import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const pdf = vi.hoisted(() => ({ getDocument: vi.fn(), getPage: vi.fn(), GlobalWorkerOptions: { workerSrc: '' } }))
vi.mock('pdfjs-dist', () => pdf)

import AttachmentPdfPreview from './AttachmentPdfPreview.vue'

describe('AttachmentPdfPreview', () => {
  let render: ReturnType<typeof vi.fn>

  beforeEach(() => {
    render = vi.fn(() => ({ promise: Promise.resolve(), cancel: vi.fn() }))
    pdf.getPage.mockImplementation(async () => ({
      getViewport: vi.fn(({ scale }: { scale: number }) => ({ width: 100 * scale, height: 140 * scale })),
      render,
    }))
    const document = {
      numPages: 2,
      getPage: pdf.getPage,
    }
    pdf.getDocument.mockReturnValue({ promise: Promise.resolve(document), destroy: vi.fn(async () => {}) })
    vi.spyOn(HTMLCanvasElement.prototype, 'getContext').mockReturnValue({} as CanvasRenderingContext2D)
  })

  it('loads protected blob content, renders a page and lets the viewer navigate', async () => {
    vi.stubGlobal('fetch', vi.fn(async () => ({ ok: true, arrayBuffer: async () => new ArrayBuffer(8) })))
    const wrapper = mount(AttachmentPdfPreview, { props: { src: 'blob:private-file', fileName: 'receipt.pdf' } })
    await flushPromises()
    expect(pdf.getPage).toHaveBeenCalled()
    await vi.waitFor(() => expect(render).toHaveBeenCalledTimes(1))
    expect(pdf.getDocument).toHaveBeenCalledWith({ data: expect.any(ArrayBuffer) })
    expect(wrapper.find('canvas').exists()).toBe(true)
    expect(wrapper.find('[role="alert"]').exists(), wrapper.text()).toBe(false)
    expect(render).toHaveBeenCalledTimes(1)
    await wrapper.get('button:last-of-type').trigger('click')
    await flushPromises()
    expect(wrapper.text()).toContain('2 / 2')
    expect(render).toHaveBeenCalledTimes(2)
    wrapper.unmount()
  })

  it('shows an error state when the protected content cannot be loaded', async () => {
    vi.stubGlobal('fetch', vi.fn(async () => ({ ok: false, arrayBuffer: async () => new ArrayBuffer(0) })))
    const wrapper = mount(AttachmentPdfPreview, { props: { src: 'blob:missing', fileName: 'missing.pdf' } })
    await flushPromises()
    expect(wrapper.find('[role="alert"]').exists()).toBe(true)
    wrapper.unmount()
  })
})

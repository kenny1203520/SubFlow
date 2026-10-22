import { createApp, h } from 'vue'
import AttachmentPdfPreview from '../src/components/AttachmentPdfPreview.vue'

export function mountPdfPreview(src: string) {
  const host = document.createElement('div')
  host.style.cssText = 'position:fixed;inset:20px;background:white;z-index:99999;width:calc(100vw - 40px);height:calc(100vh - 40px)'
  document.body.append(host)
  const app = createApp({ render: () => h(AttachmentPdfPreview, { src, fileName: 'sample.pdf' }) })
  app.mount(host)
  return () => { app.unmount(); host.remove() }
}

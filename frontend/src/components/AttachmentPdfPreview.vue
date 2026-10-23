<script setup lang="ts">
import type { PDFDocumentLoadingTask, PDFDocumentProxy, RenderTask } from 'pdfjs-dist'
import workerUrl from 'pdfjs-dist/build/pdf.worker.min.mjs?url'
import { nextTick, onBeforeUnmount, ref, watch } from 'vue'
import { useI18n } from '../i18n'

const props = defineProps<{ src: string; fileName: string }>()
const { tr } = useI18n()
const canvas = ref<HTMLCanvasElement>()
const pageNumber = ref(1)
const pageCount = ref(0)
const loading = ref(false)
const failed = ref(false)
let generation = 0
let documentProxy: PDFDocumentProxy | undefined
let loadingTask: PDFDocumentLoadingTask | undefined
let renderTask: RenderTask | undefined

async function renderPage() {
  if (!documentProxy || !canvas.value) return
  renderTask?.cancel()
  const current = generation
  try {
    const page = await documentProxy.getPage(pageNumber.value)
    if (current !== generation || !canvas.value) return
    const base = page.getViewport({ scale: 1 })
    const maxWidth = Math.max(240, (canvas.value.parentElement?.clientWidth ?? 800) - 24)
    const viewport = page.getViewport({ scale: Math.min(1.6, maxWidth / base.width) })
    const pixelRatio = Math.min(window.devicePixelRatio || 1, 2)
    const target = canvas.value
    target.width = Math.floor(viewport.width * pixelRatio)
    target.height = Math.floor(viewport.height * pixelRatio)
    target.style.width = `${Math.floor(viewport.width)}px`
    target.style.height = `${Math.floor(viewport.height)}px`
    const context = target.getContext('2d')
    if (!context) throw new Error('canvas_unavailable')
    renderTask = page.render({ canvas: target, canvasContext: context, viewport, transform: pixelRatio === 1 ? undefined : [pixelRatio, 0, 0, pixelRatio, 0, 0] })
    await renderTask.promise
  } catch (error) {
    if (current === generation && !(error instanceof Error && error.name === 'RenderingCancelledException')) failed.value = true
  }
}

watch(() => props.src, async (src) => {
  const current = ++generation
  renderTask?.cancel()
  await loadingTask?.destroy().catch(() => {})
  documentProxy = undefined
  loadingTask = undefined
  pageNumber.value = 1
  pageCount.value = 0
  failed.value = false
  loading.value = true
  try {
    const response = await fetch(src)
    if (!response.ok) throw new Error('pdf_fetch_failed')
    const pdfjs = await import('pdfjs-dist')
    pdfjs.GlobalWorkerOptions.workerSrc = workerUrl
    const task = pdfjs.getDocument({ data: await response.arrayBuffer() })
    loadingTask = task
    const document = await task.promise
    if (current !== generation) { await task.destroy(); return }
    if (document.numPages < 1) throw new Error('pdf_empty')
    documentProxy = document
    pageCount.value = document.numPages
    loading.value = false
    await nextTick()
    await renderPage()
  } catch {
    if (current === generation) failed.value = true
  } finally {
    if (current === generation) loading.value = false
  }
}, { immediate: true })

watch(pageNumber, () => { void renderPage() })
onBeforeUnmount(() => {
  generation++
  renderTask?.cancel()
  void loadingTask?.destroy()
})
</script>

<template>
  <section class="pdf-preview" :aria-label="fileName">
    <p v-if="loading" role="status" class="preview-status">{{ tr('attachmentPreviewLoading') }}</p>
    <p v-else-if="failed" role="alert" class="notice danger">{{ tr('attachmentPreviewFailed') }}</p>
    <template v-else-if="pageCount">
      <nav class="pdf-controls" :aria-label="tr('attachmentPreview')">
        <button type="button" class="ghost" :disabled="pageNumber <= 1" @click="pageNumber--">{{ tr('previousPage') }}</button>
        <span aria-live="polite">{{ pageNumber }} / {{ pageCount }}</span>
        <button type="button" class="ghost" :disabled="pageNumber >= pageCount" @click="pageNumber++">{{ tr('nextPage') }}</button>
      </nav>
      <div class="pdf-canvas-wrap"><canvas ref="canvas" :aria-label="`${tr('attachmentPreview')} ${pageNumber} / ${pageCount}`"></canvas></div>
    </template>
  </section>
</template>

<style scoped>
.pdf-preview{display:flex;flex:1;min-height:0;flex-direction:column;gap:.5rem}.pdf-controls{display:flex;justify-content:center;align-items:center;gap:.75rem;flex-wrap:wrap}.pdf-controls button{min-height:44px}.pdf-canvas-wrap{flex:1;min-height:0;overflow:auto;display:grid;place-items:start center;padding:12px;background:var(--surface-soft);border-radius:8px}.pdf-canvas-wrap canvas{max-width:100%;height:auto!important;box-shadow:0 1px 8px #0003;background:white}.preview-status{padding:1rem;text-align:center;color:var(--muted)}
</style>

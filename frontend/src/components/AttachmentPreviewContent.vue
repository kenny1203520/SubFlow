<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { canPreviewAttachment, parseOfficePreview, type OfficePreview } from '../utils/attachmentPreview'
import AttachmentPdfPreview from './AttachmentPdfPreview.vue'
import { useI18n } from '../i18n'

const props = defineProps<{ src: string; fileName: string; mediaType: string }>()
const { tr } = useI18n()
const office = ref<OfficePreview>()
const loading = ref(false)
const failed = ref(false)
const image = computed(() => props.mediaType.startsWith('image/') && canPreviewAttachment(props.fileName, props.mediaType))
const pdf = computed(() => props.mediaType === 'application/pdf')
let generation = 0

watch(() => [props.src, props.fileName, props.mediaType] as const, async ([src, fileName, mediaType]) => {
  const current = ++generation
  office.value = undefined
  failed.value = false
  if (!canPreviewAttachment(fileName, mediaType) || image.value || pdf.value) return
  loading.value = true
  try {
    const response = await fetch(src)
    if (!response.ok) throw new Error('preview_fetch_failed')
    office.value = await parseOfficePreview(await response.blob(), fileName)
  } catch {
    failed.value = true
  } finally {
    if (generation === current) loading.value = false
  }
}, { immediate: true })

onBeforeUnmount(() => { generation++ })
</script>

<template>
  <img v-if="image" :src="src" :alt="fileName">
  <AttachmentPdfPreview v-else-if="pdf" :src="src" :file-name="fileName" />
  <p v-else-if="loading" role="status" class="preview-status">{{ tr('attachmentPreviewLoading') }}</p>
  <p v-else-if="failed" role="alert" class="notice danger">{{ tr('attachmentPreviewFailed') }}</p>
  <article v-else-if="office?.kind === 'document'" class="office-document">
    <p v-for="(paragraph, index) in office.paragraphs" :key="index">{{ paragraph }}</p>
    <p v-if="office.paragraphs.length === 0" class="preview-status">{{ tr('attachmentPreviewEmpty') }}</p>
  </article>
  <section v-else-if="office?.kind === 'spreadsheet'" class="office-spreadsheet" :aria-label="tr('attachmentSpreadsheetPreview')">
    <section v-for="(sheet, index) in office.sheets" :key="index" class="office-sheet">
      <h3>{{ sheet.name }}</h3>
      <div class="office-table-scroll" tabindex="0" :aria-label="sheet.name">
        <table><tbody><tr v-for="(row, rowIndex) in sheet.rows" :key="rowIndex"><td v-for="(cell, cellIndex) in row" :key="cellIndex">{{ cell }}</td></tr></tbody></table>
      </div>
    </section>
    <p v-if="office.truncated" class="preview-status">{{ tr('attachmentSpreadsheetTruncated') }}</p>
  </section>
  <section v-else-if="office?.kind === 'slides'" class="office-slides">
    <section v-for="(slide, index) in office.slides" :key="index" class="office-slide">
      <h3>{{ slide.title || tr('attachmentSlide',{number:index+1}) }}</h3>
      <p v-for="(paragraph, paragraphIndex) in slide.paragraphs.slice(1)" :key="paragraphIndex">{{ paragraph }}</p>
    </section>
    <p v-if="office.slides.length === 0" class="preview-status">{{ tr('attachmentPreviewEmpty') }}</p>
  </section>
</template>

<style scoped>
img{display:block;width:100%;height:calc(100dvh - 100px);border:0;object-fit:contain}.office-document,.office-slides{max-width:850px;margin:1rem auto;padding:1.25rem;line-height:1.65;overflow:auto;max-height:calc(100dvh - 110px);background:var(--surface);color:var(--ink)}.office-document p{white-space:pre-wrap;margin:.4rem 0 1rem}.office-spreadsheet{display:grid;gap:1rem;max-height:calc(100dvh - 110px);overflow:auto;padding:.75rem}.office-sheet h3,.office-slide h3{margin:0 0 .65rem}.office-table-scroll{overflow:auto;max-height:65dvh;border:1px solid var(--line);border-radius:8px}.office-table-scroll:focus-visible{outline:2px solid var(--brand);outline-offset:2px}table{border-collapse:collapse;min-width:100%;font-size:.875rem}td{min-width:8rem;max-width:24rem;padding:.45rem .6rem;border:1px solid var(--line);overflow-wrap:anywhere;white-space:pre-wrap}.office-slide{padding:1rem;margin-bottom:1rem;border:1px solid var(--line);border-radius:10px;background:var(--surface-soft)}.office-slide p{white-space:pre-wrap}.preview-status{padding:1rem;text-align:center;color:var(--muted)}
</style>

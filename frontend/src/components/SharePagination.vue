<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from '../i18n'

const props = withDefaults(defineProps<{
  page: number
  totalPages: number
  pageSize: number
  pageSizes?: number[]
  loading?: boolean
}>(), {
  pageSizes: () => [5, 10, 15, 25],
  loading: false,
})
const emit = defineEmits<{
  page: [value: number]
  'update:pageSize': [value: number]
}>()
const { tr } = useI18n()
const pages = computed(() => {
  const total = Math.max(1, props.totalPages)
  const current = Math.min(Math.max(1, props.page), total)
  const start = Math.max(1, Math.min(current - 2, total - 4))
  return Array.from({ length: Math.min(5, total) }, (_, index) => start + index)
})
function selectPage(value: number) {
  if (!props.loading && value >= 1 && value <= props.totalPages && value !== props.page) emit('page', value)
}
</script>

<template>
  <div class="share-pagination" :class="{ 'is-loading': loading }">
    <label class="share-page-size">
      <span>{{ tr('pageSize') }}</span>
      <select :value="pageSize" :disabled="loading" @change="emit('update:pageSize', Number(($event.target as HTMLSelectElement).value))">
        <option v-for="size in pageSizes" :key="size" :value="size">{{ tr('pageSizeCount', { count: size }) }}</option>
      </select>
    </label>
    <nav class="share-page-navigation" :aria-label="tr('pagination')">
      <button type="button" class="share-page-button" :disabled="loading || page <= 1" @click="selectPage(page - 1)">{{ tr('previousPage') }}</button>
      <button v-for="number in pages" :key="number" type="button" class="share-page-button page-number" :class="{ active: page === number }" :disabled="loading" :aria-current="page === number ? 'page' : undefined" @click="selectPage(number)">{{ number }}</button>
      <button type="button" class="share-page-button" :disabled="loading || page >= totalPages" @click="selectPage(page + 1)">{{ tr('nextPage') }}</button>
    </nav>
    <span class="share-page-status">{{ page }} / {{ Math.max(1, totalPages) }}</span>
  </div>
</template>

<style scoped>
.share-pagination{display:flex;align-items:center;justify-content:space-between;gap:14px;padding:14px 20px;border-top:1px solid var(--line);background:var(--surface-soft)}.share-page-size{display:flex;align-items:center;gap:8px;color:var(--muted);font-size:14px}.share-page-size select{padding:8px 28px 8px 10px;border:1px solid var(--line);border-radius:9px;background:var(--surface);color:var(--ink);font-size:14px}.share-page-navigation{display:flex;align-items:center;gap:6px;flex-wrap:wrap;justify-content:flex-end}.share-page-button{min-width:34px;padding:8px 10px;border:1px solid var(--line);border-radius:9px;background:var(--surface);color:var(--muted);font-size:14px;cursor:pointer}.share-page-button:hover:not(:disabled),.share-page-button.active{border-color:var(--brand);background:var(--brand-soft);color:var(--ink)}.share-page-button:disabled{cursor:not-allowed;opacity:.5}.share-page-status{color:var(--muted);font-size:14px;white-space:nowrap}.is-loading{opacity:.7}@media(max-width:680px){.share-pagination{align-items:stretch;flex-direction:column;padding:14px 18px}.share-page-size{justify-content:space-between}.share-page-navigation{justify-content:space-between}.share-page-button{flex:1}.share-page-button.page-number{flex:0 0 34px}.share-page-status{text-align:center}}
</style>

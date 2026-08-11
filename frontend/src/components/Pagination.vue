<script setup lang="ts">
import { computed } from 'vue'
import type { Meta } from '../api/types'
import { useI18n } from '../i18n'

const props = defineProps<{ meta?: Meta }>()
const emit = defineEmits<{ page: [page: number] }>()
const { tr } = useI18n()
const pages = computed(() => {
  const total = props.meta?.totalPages || 0
  const current = props.meta?.page || 1
  const start = Math.max(1, current - 2)
  return Array.from({ length: Math.min(5, Math.max(0, total - start + 1)) }, (_, index) => start + index)
})
</script>

<template>
  <nav v-if="meta && meta.totalPages > 1" class="audit-pagination" :aria-label="tr('auditPagination')">
    <button class="ghost" type="button" :disabled="meta.page <= 1" @click="emit('page', meta.page - 1)">{{ tr('previousPage') }}</button>
    <button v-for="value in pages" :key="value" class="page-number" :class="{ active: value === meta.page }" type="button" :aria-current="value === meta.page ? 'page' : undefined" @click="emit('page', value)">{{ value }}</button>
    <button class="ghost" type="button" :disabled="meta.page >= meta.totalPages" @click="emit('page', meta.page + 1)">{{ tr('nextPage') }}</button>
  </nav>
</template>

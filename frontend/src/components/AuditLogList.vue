<script setup lang="ts">
import { computed } from 'vue'
import type { AuditLog, Meta } from '../api/types'
import EmptyState from './EmptyState.vue'
import { useI18n } from '../i18n'
import { auditPresentation } from '../utils/audit'

const props = withDefaults(defineProps<{ logs:AuditLog[]; meta?:Meta; loading?:boolean; error?:string }>(), { logs:()=>[], loading:false, error:'' })
const emit = defineEmits<{ retry:[]; page:[page:number] }>()
const { tr, locale, formatDate } = useI18n()
const pages = computed(() => {
  const total = props.meta?.totalPages || 0
  const current = props.meta?.page || 1
  const start = Math.max(1, current - 2)
  return Array.from({ length:Math.min(5, Math.max(0, total - start + 1)) }, (_, index) => start + index)
})
function info(log:AuditLog) { return auditPresentation(log, locale.value) }
</script>

<template>
  <section class="card audit-list">
    <div class="audit-list-heading">
      <h2>{{ tr('auditLogs') }}</h2>
      <span v-if="meta" class="pill">{{ tr('auditResults', { count: meta.totalItems }) }}</span>
    </div>
    <div v-if="loading" class="empty-inline">{{ tr('processing') }}</div>
    <div v-else-if="error" class="resource-error"><p>{{ error }}</p><button class="ghost" @click="emit('retry')">{{ tr('retry') }}</button></div>
    <div v-else-if="logs.length" class="data-list">
      <article v-for="log in logs" :key="log.id" class="data-row audit-row">
        <div class="grow"><strong>{{ info(log).action }}</strong><small>{{ info(log).actor }} · {{ info(log).resource }} · {{ formatDate(log.createdAt) }}</small></div>
        <span class="pill">{{ info(log).outcome }}</span>
      </article>
    </div>
    <EmptyState v-else :title="tr('noSummary')" :description="tr('noSummary')" />
    <nav v-if="meta && meta.totalPages > 1" class="audit-pagination" :aria-label="tr('auditPagination')">
      <button class="ghost" type="button" :disabled="meta.page <= 1" @click="emit('page', meta.page - 1)">{{ tr('previousPage') }}</button>
      <button v-for="value in pages" :key="value" class="page-number" :class="{ active:value === meta.page }" type="button" :aria-current="value === meta.page ? 'page' : undefined" @click="emit('page', value)">{{ value }}</button>
      <button class="ghost" type="button" :disabled="meta.page >= meta.totalPages" @click="emit('page', meta.page + 1)">{{ tr('nextPage') }}</button>
    </nav>
  </section>
</template>

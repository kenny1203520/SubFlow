<script setup lang="ts">
import type { AuditLog, Meta } from '../api/types'
import EmptyState from './EmptyState.vue'
import Pagination from './Pagination.vue'
import PageSizeSelect from './PageSizeSelect.vue'
import { useI18n } from '../i18n'
import { auditPresentation } from '../utils/audit'

withDefaults(defineProps<{ logs:AuditLog[]; meta?:Meta; pageSize:number; loading?:boolean; error?:string }>(), { logs:()=>[], loading:false, error:'' })
const emit = defineEmits<{ retry:[]; page:[page:number]; 'update:pageSize':[value:number] }>()
const { tr, locale, formatDate } = useI18n()
function info(log:AuditLog) { return auditPresentation(log, locale.value) }
</script>

<template>
  <section class="card audit-list">
    <div class="audit-list-heading">
      <h2>{{ tr('auditLogs') }}</h2>
      <span v-if="meta" class="pill">{{ tr('auditResults', { count: meta.totalItems }) }}</span>
      <PageSizeSelect :model-value="pageSize" @update:model-value="emit('update:pageSize', $event)" />
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
    <Pagination :meta="meta" @page="emit('page', $event)" />
  </section>
</template>

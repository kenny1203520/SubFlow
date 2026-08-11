<script setup lang="ts">
import { ref } from 'vue'
import type { AuditLog, Meta } from '../api/types'
import EmptyState from './EmptyState.vue'
import Pagination from './Pagination.vue'
import PageSizeSelect from './PageSizeSelect.vue'
import { useI18n } from '../i18n'
import { auditFieldLabel, auditPresentation, auditValueLabel, parseAuditSummary } from '../utils/audit'
import { auditText } from '../locales/audit'

withDefaults(defineProps<{ logs:AuditLog[]; meta?:Meta; pageSize:number; loading?:boolean; error?:string }>(), { logs:()=>[], loading:false, error:'' })
const emit = defineEmits<{ retry:[]; page:[page:number]; 'update:pageSize':[value:number] }>()
const { tr, locale, formatDate } = useI18n()
const expanded = ref<Record<string, boolean>>({})
function info(log:AuditLog) { return auditPresentation(log, locale.value) }
function summary(log:AuditLog) { return parseAuditSummary(log.summary) }
function toggle(id:string) { expanded.value[id] = !expanded.value[id] }
function fieldLabel(field:string) { return auditFieldLabel(field, locale.value) }
function valueLabel(value:unknown) { return auditValueLabel(value, locale.value) }
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
      <article v-for="log in logs" :key="log.id" class="audit-row-wrap">
        <button type="button" class="data-row audit-row audit-row-toggle" :aria-expanded="!!expanded[log.id]" @click="toggle(log.id)">
          <div class="grow"><strong>{{ info(log).action }}</strong><small>{{ info(log).actor }} · {{ info(log).resource }} · {{ formatDate(log.createdAt) }}</small></div>
          <span class="pill" :class="{danger: log.outcome==='failure'}">{{ info(log).outcome }}</span>
          <span class="audit-row-caret" aria-hidden="true">{{ expanded[log.id] ? '▾' : '▸' }}</span>
        </button>
        <div v-if="expanded[log.id]" class="audit-row-detail">
          <template v-if="summary(log)">
            <div v-if="summary(log)?.details" class="audit-detail-block">
              <small>{{ auditText[locale].details }}</small>
              <ul><li v-for="(value, key) in summary(log)?.details" :key="key">{{ fieldLabel(String(key)) }}: {{ valueLabel(value) }}</li></ul>
            </div>
            <div v-if="summary(log)?.changes?.length" class="audit-detail-block">
              <small>{{ auditText[locale].changes }}</small>
              <ul><li v-for="change in summary(log)?.changes" :key="change.field">{{ fieldLabel(change.field) }}: {{ valueLabel(change.before) }} {{ auditText[locale].changeArrow }} {{ valueLabel(change.after) }}</li></ul>
            </div>
          </template>
          <p v-else-if="log.summary" class="audit-detail-raw">{{ log.summary }}</p>
          <p v-else class="audit-detail-raw empty-inline">{{ tr('noSummary') }}</p>
          <div class="audit-detail-meta">
            <small>{{ auditText[locale].ip }}: {{ log.ip || auditText[locale].empty }}</small>
            <small>{{ auditText[locale].userAgent }}: {{ log.userAgent || auditText[locale].empty }}</small>
          </div>
        </div>
      </article>
    </div>
    <EmptyState v-else :title="tr('noSummary')" :description="tr('noSummary')" />
    <Pagination :meta="meta" @page="emit('page', $event)" />
  </section>
</template>

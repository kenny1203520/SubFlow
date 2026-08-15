<script setup lang="ts">
import { ref } from 'vue'
import type { AuditLog, Meta } from '../api/types'
import EmptyState from './EmptyState.vue'
import Pagination from './Pagination.vue'
import PageSizeSelect from './PageSizeSelect.vue'
import { useI18n } from '../i18n'
import { useWorkspaceStore } from '../stores/workspace'
import { auditFieldLabel, auditFieldValue, auditPresentation, parseAuditSummary, visibleAuditFields } from '../utils/audit'
import { auditText } from '../locales/audit'

withDefaults(defineProps<{ logs:AuditLog[]; meta?:Meta; pageSize:number; loading?:boolean; error?:string }>(), { logs:()=>[], loading:false, error:'' })
const emit = defineEmits<{ retry:[]; page:[page:number]; 'update:pageSize':[value:number] }>()
const { tr, locale, formatDate } = useI18n()
const workspace = useWorkspaceStore()
const expanded = ref<Record<string, boolean>>({})
function info(log:AuditLog) { return auditPresentation(log, locale.value) }
function summary(log:AuditLog) { return parseAuditSummary(log.summary) }
function toggle(id:string) { expanded.value[id] = !expanded.value[id] }
function fieldLabel(field:string) { return auditFieldLabel(field, locale.value) }
// Audit rows reference members by PocketBase ID; resolve against the current
// group's already-loaded roster (same pattern as ExpensesView/SubscriptionsView)
// so the log reads "kenny (fd8p1plkjr2xg4l)" instead of a bare opaque ID.
function resolveUser(id:string) {
  const member = workspace.members.find(value => value.userId === id)
  const name = member?.user?.name || member?.user?.email
  return name ? `${name} (${id})` : id
}
function detailItems(log:AuditLog) {
  const details = summary(log)?.details
  if (!details) return []
  return visibleAuditFields(details).map(key => ({ key, text: auditFieldValue(key, details[key], details, locale.value, resolveUser) }))
}
function changeItems(log:AuditLog) {
  const parsed = summary(log)
  const changes = parsed?.changes
  if (!changes?.length) return []
  const visible = new Set(visibleAuditFields(changes.map(change => change.field)))
  return changes.filter(change => visible.has(change.field)).map(change => ({
    field: change.field,
    before: auditFieldValue(change.field, change.before, parsed?.details, locale.value, resolveUser),
    after: auditFieldValue(change.field, change.after, parsed?.details, locale.value, resolveUser),
  }))
}
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
            <div v-if="detailItems(log).length" class="audit-detail-block">
              <small>{{ auditText[locale].details }}</small>
              <ul><li v-for="item in detailItems(log)" :key="item.key">{{ fieldLabel(item.key) }}: {{ item.text }}</li></ul>
            </div>
            <div v-if="changeItems(log).length" class="audit-detail-block">
              <small>{{ auditText[locale].changes }}</small>
              <ul><li v-for="change in changeItems(log)" :key="change.field">{{ fieldLabel(change.field) }}: {{ change.before }} {{ auditText[locale].changeArrow }} {{ change.after }}</li></ul>
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

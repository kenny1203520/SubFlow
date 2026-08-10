<script setup lang="ts">
import { reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import AuditFilterBar, { type AuditFilters } from '../components/AuditFilterBar.vue'
import AuditLogList from '../components/AuditLogList.vue'
import { useWorkspaceStore } from '../stores/workspace'
import { useI18n } from '../i18n'

const route = useRoute(), router = useRouter(), workspace = useWorkspaceStore(), { tr } = useI18n()
const loading = ref(false), error = ref('')
const filters = reactive<AuditFilters>({ q:'', action:'', resource:'', outcome:'', from:'', to:'' })

function readFilters(): AuditFilters {
  return { q:String(route.query.q || ''), action:String(route.query.action || ''), resource:String(route.query.resource || ''), outcome:String(route.query.outcome || ''), from:String(route.query.from || ''), to:String(route.query.to || '') }
}
function queryFor(page = Number(route.query.page || 1)) {
  const query:Record<string, string> = {}
  for (const [key, value] of Object.entries(filters)) if (value) query[key] = value
  if (page > 1) query.page = String(page)
  return query
}
async function load() {
  if (!route.params.groupId) return
  loading.value = true; error.value = ''
  try { await workspace.loadGroupAuditLogs(new URLSearchParams(queryFor()).toString()) } catch { error.value = tr('requestFailed') } finally { loading.value = false }
}
function apply() { void router.replace({ query:queryFor(1) }) }
function reset() { void router.replace({ query:{} }) }
function setPage(page:number) { void router.replace({ query:queryFor(page) }) }
watch(() => route.fullPath, () => { Object.assign(filters, readFilters()); void load() }, { immediate:true })
</script>

<template>
  <section class="page group-audit-page">
    <div class="page-heading"><div><p class="eyebrow">{{ tr('auditLogs') }}</p><h1>{{ tr('auditLogs') }}</h1><p>{{ tr('groupAuditDesc') }}</p></div></div>
    <AuditFilterBar :model-value="filters" @update:model-value="Object.assign(filters, $event)" @apply="apply" @reset="reset" />
    <AuditLogList :logs="workspace.groupAuditLogs" :meta="workspace.groupAuditMeta" :loading="loading" :error="error" @retry="load" @page="setPage" />
  </section>
</template>

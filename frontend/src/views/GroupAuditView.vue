<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import AuditFilterBar, { type AuditFilters } from '../components/AuditFilterBar.vue'
import AuditLogList from '../components/AuditLogList.vue'
import { useWorkspaceStore } from '../stores/workspace'
import { useI18n } from '../i18n'
import { defaultPageSize } from '../pageSize'

const route = useRoute(), router = useRouter(), workspace = useWorkspaceStore(), { tr } = useI18n()
const loading = ref(false), error = ref('')
const filters = reactive<AuditFilters>({ q:'', action:'', resource:'', outcome:'', from:'', to:'' })
// Page size lives in the URL alongside the filters so that changing it always
// alters the route and re-triggers the loader below, and so the view can be
// shared or reloaded with the same size.
const pageSize = computed(() => Number(route.query.perPage) || defaultPageSize.value)

function readFilters(): AuditFilters {
  return { q:String(route.query.q || ''), action:String(route.query.action || ''), resource:String(route.query.resource || ''), outcome:String(route.query.outcome || ''), from:String(route.query.from || ''), to:String(route.query.to || '') }
}
function queryFor(page = Number(route.query.page || 1), perPage = pageSize.value) {
  const query:Record<string, string> = {}
  for (const [key, value] of Object.entries(filters)) if (value) query[key] = value
  if (page > 1) query.page = String(page)
  if (perPage !== defaultPageSize.value) query.perPage = String(perPage)
  return query
}
async function load() {
  if (!route.params.groupId) return
  loading.value = true; error.value = ''
  try { await workspace.loadGroupAuditLogs(new URLSearchParams({ ...queryFor(), perPage:String(pageSize.value) }).toString()) } catch { error.value = tr('requestFailed') } finally { loading.value = false }
}
function apply() { void router.replace({ query:queryFor(1) }) }
function reset() { void router.replace({ query:{} }) }
function setPage(page:number) { void router.replace({ query:queryFor(page) }) }
function setPageSize(value:number) { void router.replace({ query:queryFor(1, value) }) }
watch(() => route.fullPath, () => { Object.assign(filters, readFilters()); void load() }, { immediate:true })
</script>

<template>
  <section class="page group-audit-page">
    <div class="page-heading"><div><p class="eyebrow">{{ tr('auditLogs') }}</p><h1>{{ tr('auditLogs') }}</h1><p>{{ tr('groupAuditDesc') }}</p></div></div>
    <AuditFilterBar :model-value="filters" @update:model-value="Object.assign(filters, $event)" @apply="apply" @reset="reset" />
    <AuditLogList :logs="workspace.groupAuditLogs" :meta="workspace.groupAuditMeta" :page-size="pageSize" :loading="loading" :error="error" @retry="load" @page="setPage" @update:page-size="setPageSize" />
  </section>
</template>

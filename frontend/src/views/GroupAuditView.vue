<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import EmptyState from '../components/EmptyState.vue'
import { useWorkspaceStore } from '../stores/workspace'
import { useI18n } from '../i18n'
import { auditPresentation } from '../utils/audit'

const route = useRoute()
const workspace = useWorkspaceStore()
const { tr, formatDate, locale } = useI18n()
const loading = ref(false)
const failed = ref(false)
const groupId = computed(() => String(route.params.groupId || ''))
const auditInfo = (entry: import('../api/types').AuditLog) => auditPresentation(entry, locale.value)

async function load() {
  if (!groupId.value) return
  loading.value = true
  failed.value = false
  try { await workspace.loadGroupAuditLogs() } catch { failed.value = true } finally { loading.value = false }
}
onMounted(() => void load())
watch(groupId, () => void load())
</script>

<template>
  <section class="page group-audit-page">
    <div class="page-heading">
      <div><p class="eyebrow">{{ tr('auditLogs') }}</p><h1>{{ tr('auditLogs') }}</h1><p>{{ tr('groupAuditDesc') }}</p></div>
    </div>
    <section class="card audit-list">
      <div v-if="loading" class="empty-inline">{{ tr('processing') }}</div>
      <div v-else-if="failed" class="resource-error"><p>{{ tr('requestFailed') }}</p><button class="ghost" @click="load">{{ tr('retry') }}</button></div>
      <div v-else-if="workspace.groupAuditLogs.length" class="rows">
        <div v-for="entry in workspace.groupAuditLogs" :key="entry.id" class="row">
          <div class="grow"><strong>{{ auditInfo(entry).action }}</strong><small>{{ auditInfo(entry).actor }} · {{ auditInfo(entry).resource }} · {{ formatDate(entry.createdAt) }}</small></div>
          <span class="pill">{{ auditInfo(entry).outcome }}</span>
        </div>
      </div>
      <EmptyState v-else :title="tr('noSummary')" :description="tr('groupAuditEmptyDesc')" />
    </section>
  </section>
</template>

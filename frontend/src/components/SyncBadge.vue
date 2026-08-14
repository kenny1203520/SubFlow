<script setup lang="ts">
import { useI18n } from '../i18n'

const props = defineProps<{ pendingSync?: boolean; syncError?: string }>()
const { tr } = useI18n()
</script>

<template>
  <span v-if="props.syncError" class="sync-badge sync-badge-error" :title="tr('syncErrorBadgeHelp', { reason: props.syncError })">⚠ {{ tr('syncErrorBadge') }}</span>
  <span v-else-if="props.pendingSync" class="sync-badge sync-badge-pending" :title="tr('pendingSyncBadgeHelp')">☁︎/ {{ tr('pendingSyncBadge') }}</span>
</template>

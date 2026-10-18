<script setup lang="ts">
import { ref } from 'vue'
import AppDrawer from './AppDrawer.vue'
import AttachmentPanel from './AttachmentPanel.vue'
import { useI18n } from '../i18n'

const props = defineProps<{ targetType:string; targetId:string; online:boolean; disabled?:boolean; billingAt?:string }>()
const open = ref(false)
const { tr } = useI18n()
</script>

<template>
  <template v-if="online">
    <button type="button" class="icon-button" :aria-label="tr('attachments')" :disabled="disabled" @click="open=true"><svg aria-hidden="true" viewBox="0 0 24 24" width="18" height="18" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="m8 12.5 6.7-6.7a3.2 3.2 0 0 1 4.5 4.5l-8.5 8.5a5 5 0 0 1-7.1-7.1l8.1-8.1"/><path d="m7.3 13.2 6.4-6.4"/></svg></button>
    <AppDrawer :open="open" :title="tr('attachments')" @close="open=false">
      <AttachmentPanel :target-type="targetType" :target-id="targetId" :billing-at="billingAt" :online="online" :disabled="disabled"/>
    </AppDrawer>
  </template>
</template>

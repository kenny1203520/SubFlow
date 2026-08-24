<script setup lang="ts">
import { computed } from 'vue'
import BaseInput from './BaseInput.vue'
import BaseCombobox from './BaseCombobox.vue'
import { useI18n } from '../i18n'

export type SettlementFilters = { memberId:string; from:string; to:string; sort:string }

const props = defineProps<{ modelValue: SettlementFilters; members: { userId:string; label:string }[] }>()
const emit = defineEmits<{ 'update:modelValue':[value:SettlementFilters]; apply:[]; reset:[] }>()
const { tr } = useI18n()
const memberOptions = computed(() => props.members.map(member => ({ value:member.userId, label:member.label })))
const sortOptions = computed(() => [
  { value:'-settled_on', label:tr('settlementSortNewest') },
  { value:'settled_on', label:tr('settlementSortOldest') },
  { value:'-created', label:tr('settlementSortCreatedNewest') },
  { value:'created', label:tr('settlementSortCreatedOldest') },
])

function update<K extends keyof SettlementFilters>(key:K, value:string) {
  emit('update:modelValue', { ...props.modelValue, [key]:value })
}
function reset() {
  emit('update:modelValue', { memberId:'', from:'', to:'', sort:'-settled_on' })
  emit('reset')
}
</script>

<template>
  <form class="settlement-filter-bar" @submit.prevent="emit('apply')">
    <BaseCombobox :model-value="modelValue.memberId" :options="memberOptions" :label="tr('settlementMemberFilter')" :placeholder="tr('allAuditValues')" :allow-create="false" @update:model-value="update('memberId', $event)" />
    <BaseInput :model-value="modelValue.from" type="date" :label="tr('auditFrom')" @update:model-value="update('from', $event)" />
    <BaseInput :model-value="modelValue.to" type="date" :label="tr('auditTo')" @update:model-value="update('to', $event)" />
    <BaseCombobox :model-value="modelValue.sort" :options="sortOptions" :label="tr('settlementSort')" :allow-create="false" @update:model-value="update('sort', $event)" />
    <div class="audit-filter-actions">
      <button class="primary" type="submit">{{ tr('applyFilters') }}</button>
      <button class="ghost" type="button" @click="reset">{{ tr('clearFilters') }}</button>
    </div>
  </form>
</template>
<style scoped>
.settlement-filter-bar{display:grid;grid-template-columns:repeat(auto-fit,minmax(180px,1fr));align-items:end;gap:14px;padding:20px;border:1px solid var(--line);border-radius:14px;background:var(--surface);margin-bottom:16px}
</style>

<script setup lang="ts">
import { computed } from 'vue'
import BaseInput from './BaseInput.vue'
import BaseCombobox from './BaseCombobox.vue'
import { useI18n } from '../i18n'
import { auditText } from '../locales/audit'

export type AuditFilters = { q:string; action:string; resource:string; outcome:string; from:string; to:string }

const props = defineProps<{ modelValue: AuditFilters }>()
const emit = defineEmits<{ 'update:modelValue':[value:AuditFilters]; apply:[]; reset:[] }>()
const { tr, locale } = useI18n()
const translations = computed(() => auditText[locale.value])
const actions = computed(() => Object.entries(translations.value.action).map(([value, label]) => ({ value, label })))
const resources = computed(() => Object.entries(translations.value.resource).map(([value, label]) => ({ value, label })))
const outcomes = computed(() => [{ value:'success', label:translations.value.success }, { value:'failure', label:translations.value.failure }])

function update<K extends keyof AuditFilters>(key:K, value:string) {
  emit('update:modelValue', { ...props.modelValue, [key]:value })
}
function reset() {
  emit('update:modelValue', { q:'', action:'', resource:'', outcome:'', from:'', to:'' })
  emit('reset')
}
</script>

<template>
  <form class="audit-filter-bar" @submit.prevent="emit('apply')">
    <BaseInput :model-value="modelValue.q" :label="tr('auditSearch')" :placeholder="tr('auditSearchPlaceholder')" @update:model-value="update('q', $event)" />
    <BaseCombobox :model-value="modelValue.action" :options="actions" :label="tr('auditAction')" :placeholder="tr('allAuditValues')" :allow-create="false" @update:model-value="update('action', $event)" />
    <BaseCombobox :model-value="modelValue.resource" :options="resources" :label="tr('auditResource')" :placeholder="tr('allAuditValues')" :allow-create="false" @update:model-value="update('resource', $event)" />
    <BaseCombobox :model-value="modelValue.outcome" :options="outcomes" :label="tr('auditOutcome')" :placeholder="tr('allAuditValues')" :allow-create="false" @update:model-value="update('outcome', $event)" />
    <BaseInput :model-value="modelValue.from" type="date" :label="tr('auditFrom')" @update:model-value="update('from', $event)" />
    <BaseInput :model-value="modelValue.to" type="date" :label="tr('auditTo')" @update:model-value="update('to', $event)" />
    <div class="audit-filter-actions">
      <button class="primary" type="submit">{{ tr('applyFilters') }}</button>
      <button class="ghost" type="button" @click="reset">{{ tr('clearFilters') }}</button>
    </div>
  </form>
</template>

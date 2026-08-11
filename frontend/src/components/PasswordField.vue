<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from '../i18n'
import BaseInput from './BaseInput.vue'

const props = defineProps<{ modelValue: string; label: string; autocomplete?: string; required?: boolean; minlength?: number; disabled?: boolean; help?: string; error?: string }>()
const emit = defineEmits<{ 'update:modelValue': [value: string] }>()
const visible = ref(false)
const { tr } = useI18n()
const toggleLabel = computed(() => visible.value ? tr('hidePassword') : tr('showPassword'))
</script>

<template>
  <BaseInput v-bind="props" :type="visible ? 'text' : 'password'" @update:model-value="emit('update:modelValue', $event)">
    <template #trailing>
      <button class="password-toggle" type="button" :disabled="disabled" :aria-label="toggleLabel" :aria-pressed="visible" @click="visible = !visible">
        <svg v-if="visible" viewBox="0 0 24 24" aria-hidden="true"><path d="m3 3 18 18M10.6 10.7a3 3 0 0 0 4.2 4.2M9.9 5.1A10.7 10.7 0 0 1 12 5c5.2 0 8.7 4.1 9.7 6.2a1.9 1.9 0 0 1 0 1.6 17 17 0 0 1-2.3 3.3M6.2 6.2C3.9 7.7 2.5 10.1 2.1 11.2a1.9 1.9 0 0 0 0 1.6C3.1 14.9 6.6 19 12 19c1 0 1.9-.1 2.8-.4"/></svg>
        <svg v-else viewBox="0 0 24 24" aria-hidden="true"><path d="M2.1 12c1-2.1 4.5-7 9.9-7s8.9 4.9 9.9 7c-1 2.1-4.5 7-9.9 7S3.1 14.1 2.1 12Z"/><circle cx="12" cy="12" r="3"/></svg>
      </button>
    </template>
  </BaseInput>
</template>

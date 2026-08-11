<script setup lang="ts">
import { useI18n } from '../i18n'

withDefaults(defineProps<{ modelValue: number; options?: number[] }>(), { options: () => [10, 25, 50, 100] })
const emit = defineEmits<{ 'update:modelValue': [value: number] }>()
const { tr } = useI18n()
</script>

<template>
  <label class="page-size-select">
    <span>{{ tr('pageSize') }}</span>
    <select :value="modelValue" @change="emit('update:modelValue', Number(($event.target as HTMLSelectElement).value))">
      <option v-for="option in options" :key="option" :value="option">{{ tr('pageSizeCount', { count: option }) }}</option>
    </select>
  </label>
</template>

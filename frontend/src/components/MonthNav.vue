<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from '../i18n'

const props = defineProps<{ modelValue: string }>()
const emit = defineEmits<{ 'update:modelValue': [string] }>()
const { tr, formatMonth } = useI18n()

const isCurrentMonth = computed(() => props.modelValue === new Date().toISOString().slice(0, 7))

function moveMonth(delta: number) {
  const [year, month] = props.modelValue.split('-').map(Number)
  const next = new Date(Date.UTC(year, month - 1 + delta, 1))
  emit('update:modelValue', next.toISOString().slice(0, 7))
}
function goCurrentMonth() {
  if (isCurrentMonth.value) return
  emit('update:modelValue', new Date().toISOString().slice(0, 7))
}
function onPicker(event: Event) {
  emit('update:modelValue', (event.target as HTMLInputElement).value)
}
</script>

<template>
  <div class="month-nav">
    <button class="icon-button" :aria-label="tr('previousMonth')" @click="moveMonth(-1)">‹</button>
    <strong>{{ formatMonth(modelValue) }}</strong>
    <button class="icon-button" :aria-label="tr('nextMonth')" @click="moveMonth(1)">›</button>
    <button class="ghost month-jump" :class="{ 'is-current': isCurrentMonth }" :disabled="isCurrentMonth" @click="goCurrentMonth">{{ tr('currentMonth') }}</button>
    <input class="month-picker" type="month" :value="modelValue" :aria-label="tr('currentMonth')" @change="onPicker">
  </div>
</template>

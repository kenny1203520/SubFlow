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
    <div class="month-navigation">
      <button class="icon-button" :aria-label="tr('previousMonth')" @click="moveMonth(-1)">‹</button>
      <strong>{{ formatMonth(modelValue) }}</strong>
      <button class="icon-button" :aria-label="tr('nextMonth')" @click="moveMonth(1)">›</button>
    </div>
    <div class="month-actions">
      <button class="ghost month-jump" :class="{ 'is-current': isCurrentMonth }" :disabled="isCurrentMonth" @click="goCurrentMonth">{{ tr('currentMonth') }}</button>
      <input class="month-picker" type="month" :value="modelValue" :aria-label="tr('currentMonth')" @change="onPicker">
    </div>
  </div>
</template>

<style scoped>
.month-nav{display:grid;grid-template-columns:minmax(0,1fr);gap:8px;width:100%;min-width:0;white-space:normal}
.month-navigation{display:grid;grid-template-columns:44px minmax(0,1fr) 44px;align-items:center;gap:8px}
.month-navigation strong{min-width:0;text-align:center}
.month-navigation .icon-button{width:44px;min-width:44px;height:44px}
.month-actions{display:flex;flex-wrap:wrap;align-items:center;gap:8px}
.month-actions .month-jump{flex:0 0 auto;min-height:44px}
.month-actions .month-picker{flex:1 0 150px;width:150px;max-width:100%;min-height:44px}
</style>

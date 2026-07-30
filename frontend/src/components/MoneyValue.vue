<script setup lang="ts">
import { computed } from 'vue'
import { minorToMajor } from '../api/money'

const props = defineProps<{ amount: number; currency?: string }>()
const value = computed(() => new Intl.NumberFormat('zh-TW', {
  style: 'currency',
  currency: props.currency ?? 'TWD',
}).format(minorToMajor(props.amount, props.currency)))
</script>

<template><span>{{ value }}</span></template>

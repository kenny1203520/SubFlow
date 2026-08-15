<script setup lang="ts">
import { computed } from 'vue'
import { formatMoney } from '../api/money'
import { useI18n } from '../i18n'
import { currencyLabel } from '../currency'

const props = defineProps<{ amount: number; currency?: string }>()
const { locale } = useI18n()
const value = computed(() => formatMoney(props.amount, props.currency ?? 'TWD', locale.value))
const accessible = computed(() => `${value.value} · ${currencyLabel(props.currency ?? 'TWD', locale.value)}`)
</script>

<template><span :aria-label="accessible" :title="currencyLabel(currency ?? 'TWD',locale)">{{ value }}</span></template>

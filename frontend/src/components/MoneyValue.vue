<script setup lang="ts">
import { computed } from 'vue'
import { minorToMajor } from '../api/money'
import { useI18n } from '../i18n'
import { currencyLabel } from '../currency'

const props = defineProps<{ amount: number; currency?: string }>()
const { locale } = useI18n()
const value = computed(() => { try { return new Intl.NumberFormat(locale.value, {style:'currency',currency:props.currency??'TWD'}).format(minorToMajor(props.amount,props.currency)) } catch { return `${minorToMajor(props.amount,props.currency)} ${props.currency??'TWD'}` } })
const accessible = computed(() => `${value.value} · ${currencyLabel(props.currency ?? 'TWD', locale.value)}`)
</script>

<template><span :aria-label="accessible" :title="currencyLabel(currency ?? 'TWD',locale)">{{ value }}</span></template>

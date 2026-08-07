<script setup lang="ts">
import { computed } from 'vue'
import type { Currency, CurrencyInfo } from '../api/types'
import { currencyLabel } from '../currency'
import { useI18n } from '../i18n'
const props=withDefaults(defineProps<{modelValue:Currency;currencies?:CurrencyInfo[]}>(),{currencies:()=>[{code:'TWD',digits:2},{code:'USD',digits:2},{code:'JPY',digits:0},{code:'EUR',digits:2}]})
const emit=defineEmits<{(e:'update:modelValue',value:Currency):void}>()
const {locale}=useI18n()
const options=computed(()=>props.currencies.map(item=>({...item,label:currencyLabel(item.code,locale.value)})))
</script>
<template><select :value="modelValue" @change="emit('update:modelValue',($event.target as HTMLSelectElement).value)"><option v-for="item in options" :key="item.code" :value="item.code">{{item.label}}</option></select></template>

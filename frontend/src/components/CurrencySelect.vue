<script setup lang="ts">
import { computed, nextTick, ref } from 'vue'
import type { Currency, CurrencyInfo } from '../api/types'
import { currencyLabel } from '../currency'
import { useI18n } from '../i18n'
import BaseDropdown from './BaseDropdown.vue'
const props=withDefaults(defineProps<{modelValue:Currency;currencies?:CurrencyInfo[]}>(),{currencies:()=>[{code:'TWD',digits:2},{code:'USD',digits:2},{code:'JPY',digits:0},{code:'EUR',digits:2}]})
const fallbackCurrencies:CurrencyInfo[]=[{code:'TWD',digits:2},{code:'USD',digits:2},{code:'JPY',digits:0},{code:'EUR',digits:2}]
const emit=defineEmits<{(e:'update:modelValue',value:Currency):void}>()
const {locale,tr}=useI18n()
const open=ref(false),query=ref(''),searchInput=ref<HTMLInputElement|null>(null)
const options=computed(()=>{const values=props.currencies?.length?props.currencies:fallbackCurrencies;return values.map(item=>({...item,label:currencyLabel(item.code,locale.value)}))})
const selected=computed(()=>options.value.find(item=>item.code===props.modelValue))
const matches=computed(()=>{const term=query.value.trim().toLocaleLowerCase(locale.value);return options.value.filter(item=>!term||`${item.label} ${item.code}`.toLocaleLowerCase(locale.value).includes(term))})
function select(value:Currency,close:()=>void){emit('update:modelValue',value);query.value='';close()}
function focusSearch(){nextTick(()=>searchInput.value?.focus())}
</script>
<template>
  <BaseDropdown v-model="open" class="currency-select" :panel-label="tr('currency')" mobile-sheet @opened="focusSearch">
    <template #trigger="{ open: isOpen, toggle }"><button class="currency-trigger" type="button" :aria-expanded="isOpen" @click="toggle"><span>{{selected?.label||modelValue}}</span><svg viewBox="0 0 24 24" aria-hidden="true"><path d="m7 10 5 5 5-5"/></svg></button></template>
    <template #default="{ close }"><div class="currency-dropdown"><div class="currency-search"><svg viewBox="0 0 24 24" aria-hidden="true"><circle cx="11" cy="11" r="6"/><path d="m16 16 4 4"/></svg><input ref="searchInput" v-model="query" :placeholder="tr('search')"/></div><p v-if="!matches.length" class="currency-empty">{{tr('notFound')}}</p><button v-for="item in matches" :key="item.code" type="button" class="currency-option" :class="{selected:item.code===modelValue}" role="option" :aria-selected="item.code===modelValue" @click="select(item.code,close)"><span><strong>{{item.label}}</strong><em>{{item.code}}</em></span><svg v-if="item.code===modelValue" viewBox="0 0 24 24" aria-hidden="true"><path d="m5 12 4.2 4.2L19 6.8"/></svg></button></div></template>
  </BaseDropdown>
</template>

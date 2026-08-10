<script setup lang="ts">
import { computed, nextTick, ref } from 'vue'
import BaseDropdown from './BaseDropdown.vue'
import CategoryIcon from './CategoryIcon.vue'
import { useI18n } from '../i18n'

const props=withDefaults(defineProps<{modelValue:string;label?:string}>(),{label:''})
const emit=defineEmits<{ 'update:modelValue':[value:string] }>()
const {tr}=useI18n(),open=ref(false),query=ref(''),input=ref<HTMLInputElement|null>(null)
const icons=['tag','food_dining','transport','housing','utilities','shopping','entertainment','health','education','travel','insurance','software_digital','memberships','taxes_fees','gifts_donations','other']
const iconLabel=(icon:string)=>icon==='tag'?tr('category'):tr(`category_${icon}` as never)
const filtered=computed(()=>{const value=query.value.trim().toLocaleLowerCase();return !value?icons:icons.filter(icon=>`${iconLabel(icon)} ${icon}`.toLocaleLowerCase().includes(value))})
function choose(icon:string,close:()=>void){emit('update:modelValue',icon);query.value='';close()}
</script>
<template><BaseDropdown v-model="open" class="category-icon-picker" :panel-label="props.label||tr('category')" @opened="nextTick(()=>input?.focus())"><template #trigger="{toggle}"><button type="button" class="icon-trigger" :aria-label="props.label||tr('category')" :aria-expanded="open" @click="toggle"><CategoryIcon :icon="modelValue"/><span aria-hidden="true">⌄</span></button></template><template #default="{close}"><div class="icon-menu"><input ref="input" v-model="query" :placeholder="tr('search')"><button v-for="icon in filtered" :key="icon" type="button" class="icon-option" :class="{selected:icon===modelValue}" @click="choose(icon,close)"><CategoryIcon :icon="icon"/><span>{{iconLabel(icon)}}</span></button></div></template></BaseDropdown></template>
<style scoped>.category-icon-picker{position:relative}.icon-trigger{display:flex;align-items:center;justify-content:space-between;gap:.4rem;min-width:4rem;min-height:44px;padding:.45rem .65rem;border:1px solid var(--line-strong);border-radius:10px;background:var(--surface-soft);color:var(--ink)}.icon-menu{display:grid;gap:.3rem;min-width:min(16rem,calc(100vw - 2rem));padding:.55rem}.icon-option{display:flex;align-items:center;gap:.55rem;padding:.5rem;border-radius:.55rem;background:transparent;color:var(--ink);text-align:left}.icon-option:hover,.icon-option.selected{background:var(--brand-soft);color:var(--brand)}</style>

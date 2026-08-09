<script setup lang="ts">
import { computed, nextTick, ref } from 'vue'
import type { AccessRole } from '../api/types'
import { useI18n } from '../i18n'
import BaseDropdown from './BaseDropdown.vue'

const props=withDefaults(defineProps<{modelValue:string;roles:AccessRole[];label?:string}>(),{label:''})
const emit=defineEmits<{ 'update:modelValue':[value:string] }>()
const {tr}=useI18n()
const open=ref(false),search=ref(''),input=ref<HTMLInputElement|null>(null)
const selected=computed(()=>props.roles.find(role=>role.id===props.modelValue))
const filtered=computed(()=>{const term=search.value.trim().toLocaleLowerCase();return props.roles.filter(role=>!term||`${role.name} ${role.key}`.toLocaleLowerCase().includes(term))})
function choose(value:string,close:()=>void){emit('update:modelValue',value);search.value='';close()}
</script>
<template><BaseDropdown v-model="open" class="role-select" :panel-label="label||tr('assignRole')" @opened="nextTick(()=>input?.focus())"><template #trigger="{toggle}"><button type="button" class="role-trigger" :aria-expanded="open" @click="toggle"><span>{{selected?.name||tr('assignRole')}}</span><span aria-hidden="true">⌄</span></button></template><template #default="{close}"><div class="role-menu"><input ref="input" v-model="search" :placeholder="tr('search')"><button v-for="role in filtered" :key="role.id" type="button" class="role-option" :class="{selected:role.id===modelValue}" @click="choose(role.id,close)"><strong>{{role.name}}</strong><small>{{role.key}}</small></button><p v-if="!filtered.length" class="empty-inline">{{tr('noSummary')}}</p></div></template></BaseDropdown></template>
<style scoped>.role-select{position:relative;width:min(260px,100%)}.role-trigger{display:flex;align-items:center;justify-content:space-between;gap:.7rem;width:100%;min-height:44px;padding:.5rem .7rem;border:1px solid var(--line-strong);border-radius:10px;background:var(--surface-soft);color:var(--ink);text-align:left}.role-trigger:hover{border-color:var(--brand)}.role-menu{display:grid;gap:.35rem;min-width:min(17rem,calc(100vw - 2rem));padding:.55rem}.role-option{display:grid;gap:.15rem;width:100%;padding:.55rem .6rem;border-radius:.55rem;background:transparent;color:var(--ink);text-align:left}.role-option:hover,.role-option.selected{background:var(--brand-soft);color:var(--brand)}.role-option small{color:var(--muted);font-size:11px}</style>

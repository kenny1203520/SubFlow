<script setup lang="ts">
import { computed, getCurrentInstance, nextTick, ref } from 'vue'
import BaseDropdown from './BaseDropdown.vue'
import { useI18n } from '../i18n'

export type ComboboxOption = string | { value:string; label:string; searchText?:string }
const props = withDefaults(defineProps<{ modelValue:string; options:ComboboxOption[]; label:string; placeholder?:string; help?:string; error?:string; required?:boolean; allowCreate?:boolean; hideLabel?:boolean }>(), { placeholder:'', allowCreate:true })
const emit = defineEmits<{ 'update:modelValue':[value:string] }>()
const { tr } = useI18n()
const open=ref(false),search=ref(''),input=ref<HTMLInputElement|null>(null)
const uid=getCurrentInstance()?.uid??Math.floor(Math.random()*100000), inputId=`base-combobox-${uid}`,helpId=`${inputId}-help`,errorId=`${inputId}-error`
const describedBy=computed(()=>[props.help?helpId:'',props.error?errorId:''].filter(Boolean).join(' ')||undefined)
const normalized=computed(()=>props.options.map(option=>typeof option==='string'?{value:option,label:option,searchText:option}:{...option,searchText:option.searchText||`${option.label} ${option.value}`}))
const selected=computed(()=>normalized.value.find(option=>option.value===props.modelValue))
const filtered=computed(()=>{const query=search.value.trim().toLocaleLowerCase();return !query?normalized.value:normalized.value.filter(option=>option.searchText.toLocaleLowerCase().includes(query))})
function choose(value:string,close:()=>void){emit('update:modelValue',value);search.value='';close()}
function commit(close:()=>void){const value=search.value.trim();if(props.allowCreate&&value)choose(value,close)}
</script>
<template>
  <div class="base-combobox">
    <label class="combobox-label" :class="{ 'visually-hidden': hideLabel }" :for="inputId">{{label}}<span v-if="required" aria-hidden="true"> *</span></label>
    <BaseDropdown v-model="open" :panel-label="label" mobile-sheet @opened="nextTick(()=>input?.focus())">
      <template #trigger="{toggle}"><button :id="inputId" type="button" class="combobox-trigger" :aria-expanded="open" :aria-invalid="!!error" :aria-describedby="describedBy" @click="toggle"><span>{{selected?.label||placeholder||label}}</span><span aria-hidden="true">⌄</span></button></template>
      <template #default="{close}"><div class="combobox-menu"><input ref="input" v-model="search" :placeholder="tr('search')" @keydown.enter.prevent="commit(close)"><button v-for="option in filtered" :key="option.value" type="button" class="combobox-option" :class="{selected:option.value===modelValue}" @click="choose(option.value,close)">{{option.label}}</button><button v-if="allowCreate&&search.trim()&&!normalized.some(option=>option.value===search.trim())" type="button" class="combobox-option create" @click="commit(close)">+ {{search.trim()}}</button><p v-if="!filtered.length&&!allowCreate" class="empty-inline">{{tr('notFound')}}</p></div></template>
    </BaseDropdown>
    <small v-if="help" :id="helpId" class="combobox-help">{{help}}</small><small v-if="error" :id="errorId" class="combobox-error">{{error}}</small>
  </div>
</template>
<style scoped>.base-combobox{display:grid;gap:6px;position:relative;width:100%}.combobox-label{color:var(--ink);font-size:13px;font-weight:750}.combobox-label.visually-hidden{position:absolute;width:1px;height:1px;padding:0;margin:-1px;overflow:hidden;clip:rect(0,0,0,0);white-space:nowrap;border:0}.combobox-trigger{display:flex;align-items:center;justify-content:space-between;gap:12px;width:100%;min-height:44px;padding:10px 12px;border:1px solid var(--line-strong);border-radius:10px;background:var(--surface-soft);color:var(--ink);text-align:left}.combobox-trigger:focus-visible{outline:2px solid var(--brand);outline-offset:2px}.combobox-menu{display:grid;gap:4px;max-height:var(--dropdown-available-height,min(420px,55vh));padding:8px;overflow-y:auto;overscroll-behavior:contain;min-width:min(18rem,calc(100vw - 2rem))}.combobox-menu>input{position:sticky;z-index:1;top:-8px;background:var(--surface)}.combobox-option{width:100%;padding:9px 10px;border-radius:9px;background:transparent;color:var(--ink);text-align:left}.combobox-option:hover,.combobox-option.selected{background:var(--brand-soft);color:var(--brand)}.combobox-option.create{font-weight:750}.combobox-help{color:var(--muted);font-size:12px;line-height:1.45}.combobox-error{color:var(--danger);font-size:12px;line-height:1.45}</style>

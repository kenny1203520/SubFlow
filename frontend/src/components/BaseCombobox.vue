<script setup lang="ts">
import { computed, nextTick, ref } from 'vue'
import BaseDropdown from './BaseDropdown.vue'
import { useI18n } from '../i18n'

const props = withDefaults(defineProps<{ modelValue: string; options: string[]; label: string; placeholder?: string }>(), { placeholder: '' })
const emit = defineEmits<{ 'update:modelValue': [value: string] }>()
const { tr } = useI18n()
const open = ref(false), search = ref(''), input = ref<HTMLInputElement | null>(null)
const filtered = computed(() => props.options.filter(option => option.toLocaleLowerCase().includes(search.value.trim().toLocaleLowerCase())))
function choose(value: string, close: () => void) { emit('update:modelValue', value); search.value = ''; close() }
function commit(close: () => void) { const value = search.value.trim(); if (value) choose(value, close) }
</script>
<template>
  <BaseDropdown v-model="open" class="base-combobox" :panel-label="label" @opened="nextTick(() => input?.focus())">
    <template #trigger="{ toggle }"><button type="button" class="combobox-trigger" :aria-expanded="open" @click="toggle"><span>{{ modelValue || placeholder || label }}</span><span aria-hidden="true">⌄</span></button></template>
    <template #default="{ close }"><div class="combobox-menu"><input ref="input" v-model="search" :placeholder="tr('search')" @keydown.enter.prevent="commit(close)"><button v-for="option in filtered" :key="option" type="button" class="combobox-option" :class="{ selected: option === modelValue }" @click="choose(option, close)">{{ option }}</button><button v-if="search.trim() && !options.includes(search.trim())" type="button" class="combobox-option create" @click="commit(close)">+ {{ search.trim() }}</button></div></template>
  </BaseDropdown>
</template>
<style scoped>.base-combobox{position:relative;width:100%}.combobox-trigger{display:flex;align-items:center;justify-content:space-between;gap:12px;width:100%;min-height:44px;padding:10px 12px;border:1px solid var(--line-strong);border-radius:10px;background:var(--surface-soft);color:var(--ink);text-align:left}.combobox-menu{display:grid;gap:4px;padding:8px}.combobox-option{width:100%;padding:9px 10px;border-radius:9px;background:transparent;color:var(--ink);text-align:left}.combobox-option:hover,.combobox-option.selected{background:var(--brand-soft);color:var(--brand)}.combobox-option.create{font-weight:750}</style>

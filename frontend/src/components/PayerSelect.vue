<script setup lang="ts">
import { computed, nextTick, ref } from 'vue'
import type { Membership } from '../api/types'
import { useI18n } from '../i18n'
import BaseDropdown from './BaseDropdown.vue'

const props = defineProps<{ modelValue: string; members: Membership[]; selfId?: string }>()
const emit = defineEmits<{ 'update:modelValue': [value: string] }>()
const { tr } = useI18n()
const open = ref(false), query = ref(''), input = ref<HTMLInputElement | null>(null)
const options = computed(() => props.members.map(member => ({ id: member.userId, label: member.user?.name || member.user?.email || tr('unnamedMember'), self: member.userId === props.selfId })))
const selected = computed(() => options.value.find(option => option.id === props.modelValue))
const matches = computed(() => { const term = query.value.trim().toLocaleLowerCase(); return options.value.filter(option => !term || option.label.toLocaleLowerCase().includes(term)) })
function select(value: string, close: () => void) { emit('update:modelValue', value); query.value = ''; close() }
function focus() { nextTick(() => input.value?.focus()) }
</script>
<template><BaseDropdown v-model="open" class="payer-select" :panel-label="tr('payer')" @opened="focus"><template #trigger="{open:isOpen,toggle}"><button type="button" class="payer-trigger" :aria-expanded="isOpen" @click="toggle"><span>{{selected?.self?tr('myself'):selected?.label||tr('myself')}}</span><span class="chevron">⌄</span></button></template><template #default="{close}"><div class="payer-menu"><input ref="input" v-model="query" :placeholder="tr('search')"><p v-if="!matches.length" class="empty-inline">{{tr('notFound')}}</p><button v-for="option in matches" :key="option.id" type="button" class="payer-option" :class="{selected:option.id===modelValue}" @click="select(option.id,close)"><span>{{option.self?tr('myself'):option.label}}</span><small v-if="option.self">{{option.label}}</small></button></div></template></BaseDropdown></template>
<style scoped>.payer-select{position:relative}.payer-trigger{display:flex;align-items:center;gap:.55rem;width:100%;min-height:44px;padding:.45rem .65rem;border:1px solid var(--line-strong);border-radius:10px;background:var(--surface-soft);color:var(--ink);text-align:left}.payer-trigger:hover{border-color:var(--brand)}.chevron{margin-left:auto}.payer-menu{display:grid;gap:.35rem;min-width:min(18rem,calc(100vw - 2rem));padding:.55rem}.payer-option{display:grid;gap:.1rem;width:100%;padding:.55rem;border-radius:.55rem;background:transparent;color:var(--ink);text-align:left}.payer-option:hover,.payer-option.selected{background:var(--brand-soft);color:var(--brand)}.payer-option small{color:var(--muted);font-size:.75rem}</style>

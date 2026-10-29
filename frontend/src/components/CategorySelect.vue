<script setup lang="ts">
import { computed, nextTick, ref, useId } from 'vue'
import type { Category } from '../api/types'
import { useI18n } from '../i18n'
import { categoryIcon, categoryLabel } from '../category'
import BaseDropdown from './BaseDropdown.vue'
import CategoryIcon from './CategoryIcon.vue'
import CategoryIconPicker from './CategoryIconPicker.vue'

const props = withDefaults(defineProps<{ modelValue: string; categories?: Category[]; id?: string; label?: string }>(), { categories: () => [], label: '' })
const emit = defineEmits<{ 'update:modelValue': [value: string]; create: [name: string, icon: string] }>()
const { tr } = useI18n()
const open = ref(false), search = ref(''), creating = ref(false), newName = ref(''), icon = ref('tag'), searchInput = ref<HTMLInputElement | null>(null), searchId = useId(), nameId = useId()
const filtered = computed(() => { const term=search.value.trim().toLocaleLowerCase(); return props.categories.filter(item => !term || `${categoryText(item)} ${item.customName || ''} ${item.systemKey || ''}`.toLocaleLowerCase().includes(term)) })
const selected = computed(() => props.categories.find(item => item.id === props.modelValue))
function categoryText(item?: Category) { return item ? categoryLabel(item, '', tr) : tr('uncategorized') }
function choose(value: string) { emit('update:modelValue', value); open.value = false; search.value = '' }
function startCreate() { creating.value = !creating.value; if (creating.value) nextTick(() => searchInput.value?.focus()) }
function create() { const value = newName.value.trim(); if (!value) return; emit('create', value, icon.value); newName.value = ''; creating.value = false }
</script>
<template><BaseDropdown v-model="open" class="category-dropdown" :panel-label="props.label||tr('category')" mobile-sheet @opened="nextTick(() => searchInput?.focus())"><template #trigger="{ toggle }"><button :id="props.id" type="button" class="category-trigger" :aria-label="props.label||tr('category')" :aria-expanded="open" @click="toggle"><CategoryIcon :icon="categoryIcon(selected)"/><span>{{categoryText(selected)}}</span><span class="chevron">⌄</span></button></template><div class="category-menu"><label class="sr-only" :for="searchId">{{tr('search')}}</label><input :id="searchId" ref="searchInput" v-model="search" :placeholder="tr('search')"><button type="button" class="category-option" :class="{selected:!modelValue}" @click="choose('')"><CategoryIcon icon="tag"/><span>{{tr('uncategorized')}}</span></button><button v-for="item in filtered" :key="item.id" type="button" class="category-option" :class="{selected:item.id===modelValue}" @click="choose(item.id)"><CategoryIcon :icon="categoryIcon(item)"/><span>{{categoryText(item)}}</span></button><p v-if="!filtered.length" class="empty-inline">{{tr('noSummary')}}</p><button type="button" class="ghost category-create-toggle" @click="startCreate">{{tr('addCategory')}}</button><div v-if="creating" class="category-create"><label class="sr-only" :for="nameId">{{tr('name')}}</label><input :id="nameId" v-model="newName" maxlength="80" :placeholder="tr('name')" @keyup.enter.prevent="create"><CategoryIconPicker v-model="icon" :label="tr('category')"/><button type="button" class="primary" @click="create">{{tr('add')}}</button></div></div></BaseDropdown></template>
<style scoped>.category-dropdown{position:relative}.category-trigger{display:flex;align-items:center;gap:.55rem;width:100%;min-height:44px;padding:.45rem .65rem;border:1px solid var(--line-strong);border-radius:10px;background:var(--surface-soft);color:var(--ink);text-align:left}.category-trigger:hover{border-color:var(--brand)}.chevron{margin-left:auto}.category-menu{display:grid;gap:.35rem;min-width:min(19rem,calc(100vw - 2rem));padding:.55rem}.category-option{display:flex;align-items:center;gap:.55rem;width:100%;padding:.5rem;border-radius:.55rem;background:transparent;color:var(--ink);text-align:left}.category-option:hover,.category-option.selected{background:var(--brand-soft);color:var(--brand)}.category-create-toggle{justify-self:start}.category-create{display:grid;grid-template-columns:minmax(0,1fr) minmax(8rem,.5fr) auto;gap:.4rem;padding-top:.4rem;border-top:1px solid var(--line)}@media(max-width:560px){.category-create{grid-template-columns:1fr}.category-create .primary{width:100%}}</style>

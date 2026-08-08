<script setup lang="ts">
import { computed, ref } from 'vue'
import type { Category } from '../api/types'
import { useI18n } from '../i18n'
import { categoryLabel } from '../category'
const props=withDefaults(defineProps<{modelValue:string;categories?:Category[]}>(),{categories:()=>[]});const emit=defineEmits<{(e:'update:modelValue',value:string):void;(e:'create',name:string,icon:string):void}>();const {tr}=useI18n();const search=ref('');const creating=ref(false);const icon=ref('tag');const filtered=computed(()=>props.categories.filter(v=>(v.customName||v.systemKey||'').toLowerCase().includes(search.value.toLowerCase())));function label(v:Category){return categoryLabel(v,'',tr)}
</script>
<template><div class="category-select"><input v-model="search" :placeholder="tr('search')"><select :value="modelValue" @change="emit('update:modelValue',($event.target as HTMLSelectElement).value)"><option value="">{{tr('uncategorized')}}</option><option v-for="item in filtered" :key="item.id" :value="item.id">{{label(item)}}</option></select><button type="button" class="ghost" @click="creating=!creating">{{tr('addCategory')}}</button><div v-if="creating" class="inline-create"><input v-model="search" maxlength="80"><select v-model="icon" aria-label="Category icon"><option v-for="value in ['tag','food_dining','transport','shopping','travel','health','other']" :key="value" :value="value">{{value}}</option></select><button type="button" @click="emit('create',search,icon);creating=false">{{tr('add')}}</button></div></div></template>

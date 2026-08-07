<script setup lang="ts">
import { computed, ref } from 'vue'
import type { Category } from '../api/types'
import { useI18n } from '../i18n'
const props=defineProps<{modelValue:string;categories:Category[]}>();const emit=defineEmits<{(e:'update:modelValue',value:string):void;(e:'create',name:string):void}>();const {tr}=useI18n();const search=ref('');const creating=ref(false);const filtered=computed(()=>props.categories.filter(v=>(v.customName||v.systemKey||'').toLowerCase().includes(search.value.toLowerCase())));function label(v:Category){return v.systemKey?tr((`category_${v.systemKey}`) as Parameters<typeof tr>[0]):v.customName||''}
</script>
<template><div class="category-select"><input v-model="search" :placeholder="tr('search')"><select :value="modelValue" @change="emit('update:modelValue',($event.target as HTMLSelectElement).value)"><option value="">{{tr('uncategorized')}}</option><option v-for="item in filtered" :key="item.id" :value="item.id">{{label(item)}}</option></select><button type="button" class="ghost" @click="creating=!creating">{{tr('addCategory')}}</button><div v-if="creating" class="inline-create"><input v-model="search" maxlength="80"><button type="button" @click="emit('create',search);creating=false">{{tr('add')}}</button></div></div></template>

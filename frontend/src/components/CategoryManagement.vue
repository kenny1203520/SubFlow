<script setup lang="ts">
import { computed, onMounted, reactive, ref, useId, watch } from 'vue'
import type { Category } from '../api/types'
import { useWorkspaceStore } from '../stores/workspace'
import { useI18n } from '../i18n'
import CategoryIcon from './CategoryIcon.vue'
import CategoryIconPicker from './CategoryIconPicker.vue'
import BaseInput from './BaseInput.vue'
import { categoryIcon, categoryLabel } from '../category'

// canManage is typed boolean, so Vue auto-infers `default:false` for callers
// that omit it (e.g. ProfileView.vue) unless withDefaults says otherwise —
// without this, the add/edit/archive editor silently never renders there.
const props=withDefaults(defineProps<{scope:'personal'|'group';groupId?:string;canManage?:boolean}>(),{canManage:true})
const workspace=useWorkspaceStore(),{tr}=useI18n(),editing=ref<Category|undefined>()
const iconPickerId = useId()
const form=reactive({name:'',icon:'tag'})
const kind=ref<'income'|'expense'>('expense'), error=ref(''), saving=ref(false)
const items=computed(()=>workspace.categories?.filter(item=>item.scope===props.scope&&!item.systemKey&&(item.kind||'expense')===kind.value)||[])
function reset(item?:Category){editing.value=item;form.name=item?.customName||'';form.icon=item?.iconKey||'tag'}
async function save(){if(!form.name.trim()||saving.value)return;saving.value=true;error.value='';try{if(editing.value)await workspace.updateCategory(editing.value.id,{customName:form.name,iconKey:form.icon});else await workspace.createCategory(props.scope,form.name,props.groupId||'',form.icon,kind.value);reset()}catch{error.value=tr('requestFailed')}finally{saving.value=false}}
async function archive(item:Category){try{await workspace.archiveCategory(item.id);if(editing.value?.id===item.id)reset()}catch{error.value=tr('requestFailed')}}
async function load(){error.value='';try{await workspace.loadCategories?.(props.scope,props.groupId||'',kind.value)}catch{error.value=tr('requestFailed')}}
onMounted(()=>void load());watch([()=>props.groupId,kind],()=>{reset();void load()})
</script>
<template><section class="category-management"><label>{{tr('categoryKind')}}<select v-model="kind" :disabled="saving"><option value="expense">{{tr('expense')}}</option><option value="income">{{tr('income')}}</option></select></label><p v-if="error" role="alert" class="form-error">{{error}}</p><div class="card-title"><div><h2>{{tr('category')}}</h2><p class="field-help">{{tr('categoryManagementDesc')}}</p></div></div><div v-if="canManage!==false" class="category-editor"><BaseInput v-model="form.name" :label="editing?tr('editCategory'):tr('addCategory')" :placeholder="tr('name')" required/><div><span class="icon-label">{{tr('categoryIcon')}}</span><CategoryIconPicker :id="iconPickerId" v-model="form.icon" :label="tr('categoryIcon')"/></div><div class="category-actions"><button class="primary" type="button" :disabled="!workspace.online || saving" :title="workspace.online?'':tr('offlineActionDisabled')" @click="save">{{editing?tr('saveChanges'):tr('add')}}</button><button v-if="editing" type="button" class="ghost" @click="reset()">{{tr('cancel')}}</button></div></div><div class="data-list"><article v-for="item in items" :key="item.id" class="data-row"><CategoryIcon :icon="categoryIcon(item)"/><div class="grow"><strong>{{categoryLabel(item,'',tr)}}</strong><small>{{item.iconKey||'tag'}}</small></div><template v-if="canManage!==false"><button class="ghost" @click="reset(item)" type="button">{{tr('edit')}}</button><button class="ghost danger-text" :disabled="!workspace.online || saving" :title="workspace.online?'':tr('offlineActionDisabled')" @click="archive(item)" type="button">{{tr('remove')}}</button></template></article><p v-if="!items.length" class="empty-inline">{{tr('noCategories')}}</p></div></section></template>
<style scoped>.category-management{display:grid;gap:16px}.category-editor{display:grid;grid-template-columns:minmax(0,1fr) auto auto;align-items:end;gap:12px;padding:14px;border:1px solid var(--line);border-radius:12px;background:var(--surface-soft)}.icon-label{display:block;margin-bottom:6px;color:var(--ink);font-size:13px;font-weight:750}.category-actions{display:flex;gap:8px;flex-wrap:wrap}.data-row small{display:block;color:var(--muted)}@media(max-width:620px){.category-editor{grid-template-columns:1fr}.category-actions .primary{flex:1}}</style>

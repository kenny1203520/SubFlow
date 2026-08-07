<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useWorkspaceStore } from '../stores/workspace'
import AppDrawer from '../components/AppDrawer.vue'
import ConfirmDialog from '../components/ConfirmDialog.vue'
import EmptyState from '../components/EmptyState.vue'
import type { Currency, Group } from '../api/types'
import { useI18n } from '../i18n'

const workspace=useWorkspaceStore(),router=useRouter(),route=useRoute(),{tr}=useI18n()
const settings=computed(()=>route.name==='group-settings'),drawer=ref(false),deleting=ref(false)
const form=reactive({name:'',description:'',currency:'TWD' as Currency,color:'#7357ff'})
function reset(){Object.assign(form,{name:'',description:'',currency:'TWD',color:'#7357ff'})}
function create(){reset();drawer.value=true}
function fill(group:Group){Object.assign(form,{name:group.name,description:group.description,currency:group.currency,color:group.color})}
async function open(group:Group){await router.push({name:'group-overview',params:{groupId:group.id}})}
async function submit(){if(settings.value){await workspace.updateGroup({...form})}else{await workspace.createGroup({...form});const id=workspace.currentGroupId;drawer.value=false;if(id)await router.push({name:'group-overview',params:{groupId:id}})}}
async function remove(){await workspace.deleteGroup();deleting.value=false;await router.push({name:'groups'})}
watch(()=>workspace.currentGroup,group=>{if(settings.value&&group)fill(group)},{immediate:true})
</script>

<template><section class="page"><template v-if="!settings"><div class="page-heading"><div><p class="eyebrow">{{tr('groups')}}</p><h1>{{tr('groupSpace')}}</h1><p>{{tr('groupSpaceDesc')}}</p></div><button class="primary" @click="create">{{tr('createGroup')}}</button></div><div v-if="workspace.groups.length" class="group-directory"><article v-for="group in workspace.groups" :key="group.id" class="group-card"><span class="group-color" :style="{background:group.color}"></span><div><h2>{{group.name}}</h2><p>{{group.description||tr('noDescription')}}</p></div><footer><span>{{group.currency}}</span><button class="ghost" @click="open(group)">{{tr('openGroup')}} →</button></footer></article></div><div v-else class="card"><EmptyState :title="tr('noGroups')" :description="tr('noGroupsDesc')"/></div></template>
  <template v-else><div class="page-heading"><div><p class="eyebrow">{{tr('settings')}}</p><h1>{{tr('settings')}}</h1><p>{{tr('groupWorkspaceDesc')}}</p></div></div><form class="card form-card settings-form" @submit.prevent="submit"><label>{{tr('groupName')}}<input v-model="form.name" required maxlength="120"></label><label>{{tr('groupDescription')}}<textarea v-model="form.description" rows="4"></textarea></label><div class="form-row"><label>{{tr('currency')}}<select v-model="form.currency"><option>TWD</option><option>USD</option><option>JPY</option><option>EUR</option></select></label><label>{{tr('color')}}<input v-model="form.color" type="color"></label></div><div class="form-actions"><button class="primary" :disabled="workspace.loading||!workspace.isOwner">{{tr('saveChanges')}}</button><button v-if="workspace.isOwner" type="button" class="ghost danger-text push-right" @click="deleting=true">{{tr('deleteGroup')}}</button></div></form></template>
  <AppDrawer :open="drawer" :title="tr('createGroup')" @close="drawer=false"><form class="form-card" @submit.prevent="submit"><label>{{tr('groupName')}}<input v-model="form.name" required maxlength="120" :placeholder="tr('groupNamePlaceholder')"></label><label>{{tr('groupDescription')}}<textarea v-model="form.description" rows="4" :placeholder="tr('groupDescriptionPlaceholder')"></textarea></label><div class="form-row"><label>{{tr('currency')}}<select v-model="form.currency"><option>TWD</option><option>USD</option><option>JPY</option><option>EUR</option></select></label><label>{{tr('color')}}<input v-model="form.color" type="color"></label></div><div class="form-actions"><button type="button" class="ghost" @click="drawer=false">{{tr('cancel')}}</button><button class="primary">{{tr('createGroup')}}</button></div></form></AppDrawer>
  <ConfirmDialog :open="deleting" :title="tr('deleteGroupConfirm',{name:workspace.currentGroup?.name||''})" danger @cancel="deleting=false" @confirm="remove"/>
</section></template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useWorkspaceStore } from '../stores/workspace'
import { useAuthStore } from '../stores/auth'
import AppDrawer from '../components/AppDrawer.vue'
import ConfirmDialog from '../components/ConfirmDialog.vue'
import EmptyState from '../components/EmptyState.vue'
import TimezoneSelect from '../components/TimezoneSelect.vue'
import CurrencySelect from '../components/CurrencySelect.vue'
import type { Currency, CurrencyChangePreview, Group } from '../api/types'
import { useI18n } from '../i18n'
import { timezoneLabel } from '../timezone'
import { currencyLabel } from '../currency'
import CategoryManagement from '../components/CategoryManagement.vue'

const workspace=useWorkspaceStore(),auth=useAuthStore(),router=useRouter(),route=useRoute(),{tr}=useI18n()
const settings=computed(()=>route.name==='group-settings'),drawer=ref(false),deleting=ref(false)
const currencyPreview=ref<CurrencyChangePreview>()
const currencyError=ref(''), currencyBusy=ref(false)
async function previewCurrency(){currencyError.value='';currencyBusy.value=true;try{currencyPreview.value=await workspace.previewGroupCurrency(form.currency)}catch{currencyError.value=tr('requestFailed')}finally{currencyBusy.value=false}}
const canManageSettings=computed(()=>workspace.groupPermissions.includes('group.settings.manage'))
const defaultTimezone=()=>String(auth.record?.timezone||Intl.DateTimeFormat().resolvedOptions().timeZone||'UTC')
const form=reactive({name:'',description:'',currency:'TWD' as Currency,timezone:defaultTimezone(),color:'#7357ff',attachmentMaxSize:0,attachmentMaxCount:0,settlementApprovalEnabled:false})
function reset(){Object.assign(form,{name:'',description:'',currency:'TWD',timezone:defaultTimezone(),color:'#7357ff',attachmentMaxSize:0,attachmentMaxCount:0,settlementApprovalEnabled:false})}
function create(){reset();drawer.value=true}
function fill(group:Group){Object.assign(form,{name:group.name,description:group.description,currency:group.currency,timezone:group.timezone||'UTC',color:group.color,attachmentMaxSize:Math.round((group.attachmentMaxSize||0)/1048576),attachmentMaxCount:group.attachmentMaxCount||0,settlementApprovalEnabled:!!group.settlementApprovalEnabled})}
async function open(group:Group){await router.push({name:'group-overview',params:{groupId:group.id}})}
async function submit(){const inherited=form.attachmentMaxSize===0&&form.attachmentMaxCount===0;const input={...form,attachmentMaxSize:form.attachmentMaxSize*1048576,attachmentLimitsInherited:inherited};if(settings.value){if(form.currency!==workspace.currentGroup?.currency){await previewCurrency();return}await workspace.updateGroup(input)}else{await workspace.createGroup(input);const id=workspace.currentGroupId;drawer.value=false;if(id)await router.push({name:'group-overview',params:{groupId:id}})}}
async function confirmCurrency(){if(!currencyPreview.value||currencyPreview.value.missing.length||currencyBusy.value)return;currencyBusy.value=true;currencyError.value='';try{await workspace.changeGroupCurrency(currencyPreview.value.to);currencyPreview.value=undefined;await workspace.updateGroup({...form,attachmentMaxSize:form.attachmentMaxSize*1048576,attachmentLimitsInherited:form.attachmentMaxSize===0&&form.attachmentMaxCount===0})}catch{currencyError.value=tr('requestFailed')}finally{currencyBusy.value=false}}
async function remove(){await workspace.deleteGroup();deleting.value=false;await router.push({name:'groups'})}
watch(()=>workspace.currentGroup,group=>{if(settings.value&&group){fill(group);void workspace.loadCategories?.('group',group.id)}},{immediate:true})
</script>

<template><section class="page">
  <template v-if="!settings">
    <div class="page-heading"><div><p class="eyebrow">{{tr('groups')}}</p><h1>{{tr('groupSpace')}}</h1><p>{{tr('groupSpaceDesc')}}</p></div><button class="primary" :disabled="!workspace.online" :title="workspace.online?'':tr('offlineActionDisabled')" @click="create">{{tr('createGroup')}}</button></div>
    <section v-if="workspace.pendingInvitations?.length" class="card pending-invitations"><div class="section-heading"><div><h2>{{tr('pendingInvitations')}}</h2><p>{{tr('pendingInvitationsDesc')}}</p></div></div><article v-for="invitation in workspace.pendingInvitations||[]" :key="invitation.id" class="pending-invitation"><div><strong>{{invitation.groupInfo?.name||tr('groups')}}</strong><small>{{invitation.email}}</small></div><div class="inline-actions"><button class="primary" @click="workspace.acceptPendingInvitation(invitation.id)">{{tr('acceptInvitation')}}</button><button class="ghost" @click="workspace.declinePendingInvitation(invitation.id)">{{tr('declineInvitation')}}</button></div></article></section>
    <div v-if="workspace.groups.length" class="group-directory"><article v-for="group in workspace.groups" :key="group.id" class="group-card"><span class="group-color" :style="{background:group.color}"></span><div><h2>{{group.name}}</h2><p>{{group.description||tr('noDescription')}}</p><small class="timezone-caption">{{timezoneLabel(group.timezone)}}</small></div><footer><span>{{currencyLabel(group.currency)}}</span><button class="ghost" @click="open(group)">{{tr('openGroup')}} →</button></footer></article></div>
    <div v-else class="card"><EmptyState :title="tr('noGroups')" :description="tr('noGroupsDesc')"/></div>
  </template>
  <template v-else>
    <div class="page-heading"><div><p class="eyebrow">{{tr('settings')}}</p><h1>{{tr('settings')}}</h1><p>{{tr('groupWorkspaceDesc')}}</p></div></div>
    <p v-if="currencyError" role="alert" class="notice danger">{{currencyError}}</p><button v-if="canManageSettings" class="ghost" type="button" :disabled="!workspace.online||currencyBusy" @click="previewCurrency">{{tr('previewCurrencyRepair')}}</button><form class="card form-card settings-form" @submit.prevent="submit"><label>{{tr('groupName')}}<input v-model="form.name" required maxlength="120" :disabled="!canManageSettings"></label><label>{{tr('groupDescription')}}<textarea v-model="form.description" rows="4" :disabled="!canManageSettings"></textarea></label><div class="form-row"><label>{{tr('reportingCurrency')}}<CurrencySelect v-model="form.currency" :currencies="workspace.currencies" :disabled="!canManageSettings"/></label><label>{{tr('color')}}<input v-model="form.color" type="color" :disabled="!canManageSettings"></label></div><label>{{tr('groupTimezone')}}<TimezoneSelect v-model="form.timezone" :disabled="!canManageSettings"/><small class="field-help">{{tr('groupTimezoneDesc')}}</small></label><fieldset class="settings-section settlement-approval-setting"><legend>{{tr('settlementApprovalSetting')}}</legend><label class="check"><input v-model="form.settlementApprovalEnabled" type="checkbox" :disabled="!canManageSettings||workspace.loading||!workspace.online"><span><strong>{{tr('settlementApprovalEnabled')}}</strong><small class="field-help">{{tr('settlementApprovalHelp')}}</small></span></label></fieldset><fieldset class="settings-section"><legend>{{tr('attachments')}}</legend><p class="field-help">{{tr('attachmentGroupLimitsHelp')}}</p><div class="form-row"><label>{{tr('attachmentMaxSizeMiB')}}<input v-model.number="form.attachmentMaxSize" type="number" min="0" step="1" :disabled="!canManageSettings"></label><label>{{tr('attachmentMaxCount')}}<input v-model.number="form.attachmentMaxCount" type="number" min="0" step="1" :disabled="!canManageSettings"></label></div></fieldset><div class="form-actions"><button v-if="canManageSettings" class="primary" :disabled="workspace.loading||!workspace.online" :title="workspace.online?'':tr('offlineActionDisabled')">{{tr('saveChanges')}}</button><button v-if="workspace.isOwner" type="button" class="ghost danger-text push-right" :disabled="!workspace.online" :title="workspace.online?'':tr('offlineActionDisabled')" @click="deleting=true">{{tr('deleteGroup')}}</button></div></form>
    <section class="card form-card"><CategoryManagement scope="group" :group-id="workspace.currentGroupId" :can-manage="workspace.groupPermissions.includes('categories.manage')" /></section>
  </template>
  <AppDrawer :open="drawer" :title="tr('createGroup')" @close="drawer=false"><form class="form-card" @submit.prevent="submit"><label>{{tr('groupName')}}<input v-model="form.name" required maxlength="120" :placeholder="tr('groupNamePlaceholder')"></label><label>{{tr('groupDescription')}}<textarea v-model="form.description" rows="4" :placeholder="tr('groupDescriptionPlaceholder')"></textarea></label><div class="form-row"><label>{{tr('reportingCurrency')}}<CurrencySelect v-model="form.currency" :currencies="workspace.currencies"/></label><label>{{tr('color')}}<input v-model="form.color" type="color"></label></div><label>{{tr('groupTimezone')}}<TimezoneSelect v-model="form.timezone"/><small class="field-help">{{tr('groupTimezoneDesc')}}</small></label><div class="form-actions"><button type="button" class="ghost" @click="drawer=false">{{tr('cancel')}}</button><button class="primary" :disabled="!workspace.online" :title="workspace.online?'':tr('offlineActionDisabled')">{{tr('createGroup')}}</button></div></form></AppDrawer>
  <ConfirmDialog :open="deleting" :title="tr('deleteGroupConfirm',{name:workspace.currentGroup?.name||''})" :description="tr('deleteGroupAttachmentsWarning')" danger @cancel="deleting=false" @confirm="remove"/>
  <ConfirmDialog :open="!!currencyPreview" :title="currencyPreview?.missing.length?tr('currencyChangeBlocked',{count:currencyPreview.missing.length}):tr('currencyChangeConfirm',{count:currencyPreview?.affected||0})" @cancel="currencyPreview=undefined" @confirm="confirmCurrency"/>
</section></template>
<style scoped>
.pending-invitations{display:grid;gap:.85rem;margin-bottom:1.25rem}.section-heading h2{margin:0}.section-heading p,.pending-invitation small{display:block;margin:.25rem 0 0;color:var(--muted)}.pending-invitation{display:flex;align-items:center;justify-content:space-between;gap:1rem;padding:.8rem 0;border-top:1px solid var(--line)}.pending-invitation:first-of-type{border-top:0}.inline-actions{display:flex;gap:.5rem}@media(max-width:640px){.pending-invitation{align-items:flex-start;flex-direction:column}.inline-actions{width:100%}.inline-actions button{flex:1}}
</style>

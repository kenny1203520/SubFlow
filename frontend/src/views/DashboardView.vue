<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useWorkspaceStore } from '../stores/workspace'
import { useAuthStore } from '../stores/auth'
import MoneyValue from '../components/MoneyValue.vue'
import EmptyState from '../components/EmptyState.vue'
import SyncBadge from '../components/SyncBadge.vue'
import { useI18n } from '../i18n'
import AppDrawer from '../components/AppDrawer.vue'
import ConfirmDialog from '../components/ConfirmDialog.vue'
import BaseCombobox from '../components/BaseCombobox.vue'
import MonthNav from '../components/MonthNav.vue'
import Pagination from '../components/Pagination.vue'
import PageSizeSelect from '../components/PageSizeSelect.vue'
import SettlementFilterBar, { type SettlementFilters } from '../components/SettlementFilterBar.vue'
import type { Settlement } from '../api/types'
import { amountStep, majorToMinor, minorToInput } from '../api/money'
import { timezoneLabel } from '../timezone'
import { currencyLabel } from '../currency'
import { fromDateInput, toDateInput, todayInput } from '../dateInput'
import { defaultPageSize } from '../pageSize'

type Scope='personal'|'group'|'all'
const workspace=useWorkspaceStore(),auth=useAuthStore(),route=useRoute(),router=useRouter()
const {tr,formatDate,formatMonth}=useI18n()
const nestedGroup=computed(()=>String(route.params.groupId||''))
const scope=ref<Scope>('personal')
const month=ref(new Date().toISOString().slice(0,7))
const selectedGroup=ref('')
const settlementOpen=ref(false)
const settlementError=ref('')
const pendingSettlementDelete=ref<Settlement>()
const editingSettlementId=ref('')
const settlementFilters=reactive<SettlementFilters>({memberId:'',from:'',to:'',sort:'-settled_on'})
const settlementPage=ref(1)
const settlementPerPage=ref(defaultPageSize.value)
const viewerTimezone=computed(()=>auth.record?.timezone||Intl.DateTimeFormat().resolvedOptions().timeZone||'UTC')
const settlementForm=reactive({fromUserId:'',toUserId:'',amount:'',settledOn:todayInput(viewerTimezone.value),notes:''})
const summary=computed(()=>scope.value==='group'?workspace.summary:workspace.personalSummary)
const hasSettlementPermission=(permission:string)=>workspace.groupPermissions.includes(permission)
const canCreateSettlement=computed(()=>hasSettlementPermission('ledger.settlements.create'))
const canManageOtherSettlements=computed(()=>hasSettlementPermission('ledger.settlements.manage'))
const canReadSettlements=computed(()=>hasSettlementPermission('ledger.settlements.read'))
const canCreateExpense=computed(()=>scope.value!=='group'||workspace.groupPermissions.includes('ledger.expenses.write'))
// A bound temp member has already been superseded by the real account that
// joined, so a new settlement should be recorded against that real member
// instead — history (balances/settlement list) still resolves their name.
const selectableMembers=computed(()=>workspace.members.filter(member=>!(member.user?.placeholder&&member.user?.linkedUserId)))
const actionExpense=computed(()=>scope.value==='group'&&selectedGroup.value?{name:'group-expenses',params:{groupId:selectedGroup.value}}:{name:'personal-expenses'})
const actionSubscriptions=computed(()=>scope.value==='group'&&selectedGroup.value?{name:'group-subscriptions',params:{groupId:selectedGroup.value}}:{name:'personal-subscriptions'})
const accountingGroup=computed(()=>scope.value==='group'?workspace.groups.find(group=>group.id===selectedGroup.value):undefined)
function viewerDate(value:string){return formatDate(value,{dateStyle:'medium',timeZone:viewerTimezone.value})}
function sourceGroup(groupId?:string){return workspace.groups.find(group=>group.id===groupId)}
function canEditSettlement(item:Settlement){return hasSettlementPermission('ledger.settlements.update')&&(item.createdBy===auth.record?.id||canManageOtherSettlements.value)}
function canDeleteSettlement(item:Settlement){return hasSettlementPermission('ledger.settlements.delete')&&(item.createdBy===auth.record?.id||canManageOtherSettlements.value)}
function memberName(userId:string){const user=workspace.members.find(member=>member.userId===userId)?.user;return user?.name||user?.email||tr('unnamedMember')}
function originalTime(value:string,groupId?:string){const group=sourceGroup(groupId);return group?tr('originalTimezone',{date:formatDate(value,{dateStyle:'medium',timeZone:group.timezone}),timezone:timezoneLabel(group.timezone,value)}):''}
// The upcoming-charges card otherwise shows the whole subscription's charge,
// not what the viewer themselves owes once it's split among the group.
function upcomingShare(item:{groupId?:string;splits?:{userId:string;amountMinor:number}[]}){return item.groupId?item.splits?.find(split=>split.userId===auth.record?.id)?.amountMinor:undefined}

async function syncFromRoute(){
  const defaultMonth=new Date().toISOString().slice(0,7)
  if(!route.query.month||(!nestedGroup.value&&!route.query.scope)){await router.replace({query:{...route.query,...(!nestedGroup.value&&!route.query.scope?{scope:'personal'}:{}),month:String(route.query.month||defaultMonth)}});return}
  const queryScope=String(route.query.scope||'personal') as Scope
  scope.value=nestedGroup.value?'group':(['personal','group','all'].includes(queryScope)?queryScope:'personal')
  month.value=/^\d{4}-\d{2}$/.test(String(route.query.month||''))?String(route.query.month):new Date().toISOString().slice(0,7)
  selectedGroup.value=nestedGroup.value||String(route.query.groupId||workspace.groups[0]?.id||'')
  if(scope.value==='group'&&!selectedGroup.value){scope.value='personal'}
  await workspace.refreshDashboard(scope.value,selectedGroup.value,month.value)
  // refreshDashboard's own settlement fetch (via refreshGroup) always uses the
  // plain default sort/perPage -- reload with the filter bar's current
  // sort/page/perPage so a filtered/sorted view survives a dashboard refresh.
  if(scope.value==='group'&&canReadSettlements.value) await loadSettlements(1)
}
function settlementQuery(page=settlementPage.value,perPage=settlementPerPage.value){
  const params:Record<string,string>={page:String(page),perPage:String(perPage)}
  if(settlementFilters.memberId)params.memberId=settlementFilters.memberId
  if(settlementFilters.from)params.from=settlementFilters.from
  if(settlementFilters.to)params.to=settlementFilters.to
  if(settlementFilters.sort)params.sort=settlementFilters.sort
  return new URLSearchParams(params).toString()
}
async function loadSettlements(page=1){settlementPage.value=page;await workspace.loadSettlementsPage(settlementQuery(page))}
function applySettlementFilters(){void loadSettlements(1)}
function resetSettlementFilters(){Object.assign(settlementFilters,{memberId:'',from:'',to:'',sort:'-settled_on'});void loadSettlements(1)}
function changeSettlementPageSize(value:number){settlementPerPage.value=value;void loadSettlements(1)}
async function updateQuery(next:Partial<{scope:Scope;month:string;groupId:string}>){
  if(nestedGroup.value){month.value=next.month||month.value;await router.replace({query:{...route.query,month:month.value}});return}
  await router.replace({query:{scope:next.scope||scope.value,month:next.month||month.value,...((next.groupId||selectedGroup.value)&& (next.scope||scope.value)==='group'?{groupId:next.groupId||selectedGroup.value}:{})}})
}
function openSettlement(){if(!canCreateSettlement.value)return;editingSettlementId.value='';settlementForm.fromUserId=String(workspace.currentMembership?.userId||'');settlementForm.toUserId='';settlementForm.amount='';settlementForm.settledOn=todayInput(viewerTimezone.value);settlementForm.notes='';settlementError.value='';settlementOpen.value=true}
function editSettlement(item:Settlement){editingSettlementId.value=item.id;settlementForm.fromUserId=item.fromUserId;settlementForm.toUserId=item.toUserId;settlementForm.amount=minorToInput(item.amountMinor,workspace.currentGroup?.currency);settlementForm.settledOn=toDateInput(item.settledOn,viewerTimezone.value);settlementForm.notes=item.notes;settlementError.value='';settlementOpen.value=true}
async function submitSettlement(){
  const input={fromUserId:settlementForm.fromUserId,toUserId:settlementForm.toUserId,amountMinor:majorToMinor(settlementForm.amount,workspace.currentGroup?.currency),settledOn:fromDateInput(settlementForm.settledOn,viewerTimezone.value),notes:settlementForm.notes}
  const ok=editingSettlementId.value?await workspace.updateSettlement(editingSettlementId.value,input):await workspace.addSettlement(input)
  if(!ok){settlementError.value=workspace.localizedError||tr('requestFailed');return}
  settlementOpen.value=false
  await loadSettlements(settlementPage.value)
}
async function deleteSettlement(){if(!pendingSettlementDelete.value)return;await workspace.deleteSettlement(pendingSettlementDelete.value.id);pendingSettlementDelete.value=undefined;await loadSettlements(settlementPage.value)}
watch(()=>[route.params.groupId,route.query.scope,route.query.groupId,route.query.month],()=>void syncFromRoute(),{immediate:true})
</script>

<template><section class="page dashboard-page">
  <div class="page-heading"><div><p class="eyebrow">{{tr('overview')}}</p><h1>{{scope==='personal'?tr('dashboardPersonal'):scope==='all'?tr('dashboardAll'):tr('dashboardGroup')}}</h1><p>{{tr('dashboardDesc')}}</p></div><RouterLink v-if="canCreateExpense" class="primary" :to="actionExpense">{{tr('addExpense')}}</RouterLink></div>
  <div class="dashboard-toolbar">
    <div v-if="!nestedGroup" class="segmented scope-switch"><button :class="{active:scope==='personal'}" @click="updateQuery({scope:'personal'})">{{tr('personal')}}</button><button :class="{active:scope==='group'}" :disabled="!workspace.groups.length" @click="updateQuery({scope:'group',groupId:selectedGroup||workspace.groups[0]?.id})">{{tr('singleGroup')}}</button><button :class="{active:scope==='all'}" @click="updateQuery({scope:'all'})">{{tr('allGroups')}}</button></div>
    <BaseCombobox v-if="!nestedGroup&&scope==='group'" :model-value="selectedGroup" :options="workspace.groups.map(group=>({value:group.id,label:group.name,searchText:`${group.name} ${group.currency}`}))" :label="tr('chooseGroup')" :placeholder="tr('chooseGroup')" :allow-create="false" @update:model-value="updateQuery({groupId:$event})" />
    <MonthNav :model-value="month" @update:model-value="value=>updateQuery({month:value})" />
    <div class="toolbar-timezone"><small>{{tr('yourTimezone',{timezone:timezoneLabel(viewerTimezone)})}}</small><small v-if="accountingGroup">{{tr('groupTimezoneValue',{timezone:timezoneLabel(accountingGroup.timezone)})}}</small></div>
  </div>
  <div v-if="summary?.currencies?.length" class="currency-sections"><section v-for="item in summary.currencies" :key="item.currency" class="currency-panel"><header><strong>{{currencyLabel(item.currency)}}</strong><span>{{formatMonth(month)}}</span></header><div class="metric-strip metric-strip-wide"><article><small>{{tr('cashOutflow')}}</small><MoneyValue :amount="item.cashOutflowMinor" :currency="item.currency" /></article><article><small>{{tr('personalShare')}}</small><MoneyValue :amount="item.personalShareMinor" :currency="item.currency" /></article><article><small>{{tr('reimbursable')}}</small><MoneyValue :amount="item.reimbursableMinor" :currency="item.currency" /></article><article><small>{{tr('monthlySubscription')}}</small><MoneyValue :amount="item.monthlySubscriptionMinor" :currency="item.currency" /></article><article><small>{{tr('personalMonthlySubscription')}}</small><MoneyValue :amount="item.personalMonthlySubscriptionMinor" :currency="item.currency" /></article><article><small>{{tr('activeSubscriptions')}}</small><strong>{{item.activeSubscriptions}}</strong></article></div></section></div>
  <EmptyState v-else :title="tr('noSummary')" :description="tr('noUpcomingDesc')" />
  <div class="dashboard-grid"><section class="card"><div class="card-title"><h2>{{tr('upcoming')}}</h2><RouterLink :to="actionSubscriptions">{{tr('manageSubscriptions')}} →</RouterLink></div><div v-if="summary?.upcoming?.length" class="data-list"><article v-for="item in summary.upcoming" :key="item.id" class="data-row"><div class="service-icon">{{item.name.slice(0,1)}}</div><div class="grow timezone-date"><strong>{{item.name}}</strong><small>{{viewerDate(item.nextBilling)}} · {{tr((item.lifecycleStatus||item.status) as 'active')}}</small><small v-if="item.groupId">{{originalTime(item.nextBilling,item.groupId)}}</small></div><span class="money"><MoneyValue :amount="item.amountMinor" :currency="item.currency" /><small v-if="upcomingShare(item)!==undefined">{{tr('personalShare')}}: <MoneyValue :amount="upcomingShare(item)!" :currency="item.currency" /></small></span></article></div><EmptyState v-else :title="tr('noUpcoming')" :description="tr('noUpcomingDesc')" /></section>
    <section v-if="scope==='group'" class="card"><div class="card-title"><div><h2>{{tr('balanceAsOfMonth')}}</h2><p class="setting-description">{{tr('balanceAsOfMonthDesc',{month:formatMonth(month)})}}</p></div></div><div v-if="summary?.balances?.length" class="data-list"><article v-for="balance in summary.balances" :key="balance.userId" class="data-row"><div class="avatar">{{(workspace.members.find(m=>m.userId===balance.userId)?.user?.name||'?').slice(0,1)}}</div><div class="grow"><strong>{{workspace.members.find(m=>m.userId===balance.userId)?.user?.name||tr('unnamedMember')}}</strong><small>{{balance.amountMinor>0?tr('receivable'):balance.amountMinor<0?tr('payable'):tr('settled')}}</small></div><MoneyValue :amount="Math.abs(balance.amountMinor)" :currency="workspace.currentGroup?.currency" /></article></div><EmptyState v-else :title="tr('noBalances')" :description="tr('settled')" /></section>
  </div>
  <section v-if="scope==='group'&&canReadSettlements" class="card settlement-history"><div class="card-title settlement-history-title"><div><h2>{{tr('settlements')}}</h2><span>{{tr('records',{count:workspace.settlementsMeta.totalItems})}}</span></div><div class="settlement-history-controls"><PageSizeSelect :model-value="settlementPerPage" @update:model-value="changeSettlementPageSize"/><button v-if="canCreateSettlement" class="primary" @click="openSettlement">{{tr('recordSettlement')}}</button></div></div>
    <SettlementFilterBar :model-value="settlementFilters" :members="selectableMembers.map(m=>({userId:m.userId,label:m.user?.name||m.user?.email||m.userId}))" @update:model-value="Object.assign(settlementFilters,$event)" @apply="applySettlementFilters" @reset="resetSettlementFilters" />
    <div v-if="workspace.settlements.length" class="data-list"><article v-for="item in workspace.settlements" :key="item.id" class="data-row settlement-row"><div class="settlement-parties"><div><small>{{tr('fromMember')}}</small><strong>{{memberName(item.fromUserId)}}</strong></div><span class="settlement-arrow" aria-hidden="true">→</span><div><small>{{tr('toMember')}}</small><strong>{{memberName(item.toUserId)}}</strong></div></div><dl class="settlement-details"><div><dt>{{tr('amount')}}</dt><dd><MoneyValue :amount="item.amountMinor" :currency="workspace.currentGroup?.currency"/></dd></div><div><dt>{{tr('settlementDate')}}</dt><dd>{{viewerDate(item.settledOn)}}</dd></div><div><dt>{{tr('recordedBy')}}</dt><dd>{{memberName(item.createdBy)}}</dd></div><div v-if="item.notes" class="settlement-note"><dt>{{tr('notes')}}</dt><dd>{{item.notes}}</dd></div></dl><div class="settlement-actions"><SyncBadge :pending-sync="item.pendingSync" :sync-error="item.syncError"/><button v-if="canEditSettlement(item)" class="icon-button" :aria-label="tr('edit')" @click="editSettlement(item)">✎</button><button v-if="canDeleteSettlement(item)" class="icon-button" :aria-label="tr('delete')" @click="pendingSettlementDelete=item">×</button></div></article></div><EmptyState v-else :title="tr('noSettlements')" :description="tr('recordSettlement')"/>
    <p v-if="workspace.settlements.some(item=>item.pendingSync||item.syncError)" class="field-help sync-legend"><strong>{{tr('syncLegendTitle')}}</strong> · ☁︎/ {{tr('syncLegendPending')}} · ⚠ {{tr('syncLegendError')}}</p>
    <Pagination :meta="workspace.settlementsMeta" @page="loadSettlements"/>
  </section>
  <AppDrawer :open="settlementOpen" :title="tr(editingSettlementId?'editSettlement':'recordSettlement')" @close="settlementOpen=false"><form class="form-card" @submit.prevent="submitSettlement"><div v-if="settlementError" class="notice danger inline">{{settlementError}}</div><div v-if="workspace.currentGroup" class="timezone-notice">{{tr('yourTimezone',{timezone:timezoneLabel(viewerTimezone)})}}<br>{{tr('groupTimezoneValue',{timezone:timezoneLabel(workspace.currentGroup.timezone)})}}</div><label>{{tr('fromMember')}}<select v-model="settlementForm.fromUserId" :disabled="!editingSettlementId&&!canManageOtherSettlements" required><option v-for="member in selectableMembers" :key="member.userId" :value="member.userId">{{member.user?.name||member.user?.email}}</option></select></label><label>{{tr('toMember')}}<select v-model="settlementForm.toUserId" required><option value="" disabled>{{tr('toMember')}}</option><option v-for="member in selectableMembers.filter(m=>m.userId!==settlementForm.fromUserId)" :key="member.userId" :value="member.userId">{{member.user?.name||member.user?.email}}</option></select></label><div class="form-row"><label>{{tr('amount')}}<input v-model="settlementForm.amount" type="number" :min="amountStep(workspace.currentGroup?.currency)" :step="amountStep(workspace.currentGroup?.currency)" required></label><label>{{tr('settlementDate')}}<input v-model="settlementForm.settledOn" type="date" required></label></div><label>{{tr('notes')}}<textarea v-model="settlementForm.notes" rows="3"></textarea></label><div class="form-actions"><button type="button" class="ghost" @click="settlementOpen=false">{{tr('cancel')}}</button><button class="primary">{{tr(editingSettlementId?'saveChanges':'recordSettlement')}}</button></div></form></AppDrawer>
  <ConfirmDialog :open="!!pendingSettlementDelete" :title="tr('deleteSettlementConfirm')" danger @cancel="pendingSettlementDelete=undefined" @confirm="deleteSettlement"/>
</section></template>

<style scoped>
.data-row.settlement-row{display:grid;grid-template-columns:minmax(190px,.85fr) minmax(0,1.5fr) auto;align-items:start;gap:16px;padding:16px 0}.settlement-history-title>div:first-child{display:flex;align-items:center;gap:8px}.settlement-history-controls{display:flex;align-items:center;gap:10px;margin-left:auto}.settlement-parties{display:grid;grid-template-columns:minmax(0,1fr) auto minmax(0,1fr);align-items:center;gap:8px}.settlement-parties small,.settlement-details dt{display:block;color:var(--muted);font-size:11px;font-weight:700}.settlement-parties strong{display:block;overflow-wrap:anywhere}.settlement-parties>div:last-child{text-align:right}.settlement-arrow{color:var(--brand);font-weight:800}.settlement-details{display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:12px;margin:0}.settlement-details>div{min-width:0}.settlement-details dd{margin:3px 0 0;font-size:13px;overflow-wrap:anywhere}.settlement-note{grid-column:1/-1}.settlement-note dd{white-space:pre-wrap}.settlement-actions{display:flex;align-items:center;gap:8px;min-height:28px}.settlement-actions :deep(.sync-badge){flex-shrink:0}@media(max-width:760px){.settlement-history-title{align-items:stretch;flex-wrap:wrap}.settlement-history-controls{width:100%;margin-left:0;justify-content:space-between}.settlement-history-controls .primary{flex:1}.data-row.settlement-row{grid-template-columns:1fr;gap:13px}.settlement-details{grid-template-columns:repeat(2,minmax(0,1fr))}.settlement-note{grid-column:1/-1}.settlement-actions{justify-content:flex-end;border-top:1px solid var(--line);padding-top:10px}}@media(max-width:420px){.settlement-details{grid-template-columns:1fr}.settlement-note{grid-column:auto}}
</style>

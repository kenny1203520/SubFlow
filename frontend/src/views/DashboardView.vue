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
import type { Settlement } from '../api/types'
import { amountStep, majorToMinor } from '../api/money'
import { timezoneLabel } from '../timezone'
import { currencyLabel } from '../currency'
import { fromDateInput, todayInput } from '../dateInput'

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
const viewerTimezone=computed(()=>auth.record?.timezone||Intl.DateTimeFormat().resolvedOptions().timeZone||'UTC')
const settlementForm=reactive({fromUserId:'',toUserId:'',amount:'',settledOn:todayInput(viewerTimezone.value),notes:''})
const summary=computed(()=>scope.value==='group'?workspace.summary:workspace.personalSummary)
// A bound temp member has already been superseded by the real account that
// joined, so a new settlement should be recorded against that real member
// instead — history (balances/settlement list) still resolves their name.
const selectableMembers=computed(()=>workspace.members.filter(member=>!(member.user?.placeholder&&member.user?.linkedUserId)))
const actionExpense=computed(()=>scope.value==='group'&&selectedGroup.value?{name:'group-expenses',params:{groupId:selectedGroup.value}}:{name:'personal-expenses'})
const actionSubscriptions=computed(()=>scope.value==='group'&&selectedGroup.value?{name:'group-subscriptions',params:{groupId:selectedGroup.value}}:{name:'personal-subscriptions'})
const accountingGroup=computed(()=>scope.value==='group'?workspace.groups.find(group=>group.id===selectedGroup.value):undefined)
function viewerDate(value:string){return formatDate(value,{dateStyle:'medium',timeZone:viewerTimezone.value})}
function sourceGroup(groupId?:string){return workspace.groups.find(group=>group.id===groupId)}
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
}
async function updateQuery(next:Partial<{scope:Scope;month:string;groupId:string}>){
  if(nestedGroup.value){month.value=next.month||month.value;await router.replace({query:{...route.query,month:month.value}});return}
  await router.replace({query:{scope:next.scope||scope.value,month:next.month||month.value,...((next.groupId||selectedGroup.value)&& (next.scope||scope.value)==='group'?{groupId:next.groupId||selectedGroup.value}:{})}})
}
function openSettlement(){settlementForm.fromUserId=String(workspace.currentMembership?.userId||'');settlementForm.toUserId='';settlementForm.amount='';settlementForm.settledOn=todayInput(viewerTimezone.value);settlementForm.notes='';settlementError.value='';settlementOpen.value=true}
async function submitSettlement(){const ok=await workspace.addSettlement({fromUserId:settlementForm.fromUserId,toUserId:settlementForm.toUserId,amountMinor:majorToMinor(settlementForm.amount,workspace.currentGroup?.currency),settledOn:fromDateInput(settlementForm.settledOn,viewerTimezone.value),notes:settlementForm.notes});if(!ok){settlementError.value=workspace.localizedError||tr('requestFailed');return}settlementOpen.value=false}
async function deleteSettlement(){if(!pendingSettlementDelete.value)return;await workspace.deleteSettlement(pendingSettlementDelete.value.id);pendingSettlementDelete.value=undefined}
watch(()=>[route.params.groupId,route.query.scope,route.query.groupId,route.query.month],()=>void syncFromRoute(),{immediate:true})
</script>

<template><section class="page dashboard-page">
  <div class="page-heading"><div><p class="eyebrow">{{tr('overview')}}</p><h1>{{scope==='personal'?tr('dashboardPersonal'):scope==='all'?tr('dashboardAll'):tr('dashboardGroup')}}</h1><p>{{tr('dashboardDesc')}}</p></div><RouterLink class="primary" :to="actionExpense">{{tr('addExpense')}}</RouterLink></div>
  <div class="dashboard-toolbar">
    <div v-if="!nestedGroup" class="segmented scope-switch"><button :class="{active:scope==='personal'}" @click="updateQuery({scope:'personal'})">{{tr('personal')}}</button><button :class="{active:scope==='group'}" :disabled="!workspace.groups.length" @click="updateQuery({scope:'group',groupId:selectedGroup||workspace.groups[0]?.id})">{{tr('singleGroup')}}</button><button :class="{active:scope==='all'}" @click="updateQuery({scope:'all'})">{{tr('allGroups')}}</button></div>
    <BaseCombobox v-if="!nestedGroup&&scope==='group'" :model-value="selectedGroup" :options="workspace.groups.map(group=>({value:group.id,label:group.name,searchText:`${group.name} ${group.currency}`}))" :label="tr('chooseGroup')" :placeholder="tr('chooseGroup')" :allow-create="false" @update:model-value="updateQuery({groupId:$event})" />
    <MonthNav :model-value="month" @update:model-value="value=>updateQuery({month:value})" />
    <div class="toolbar-timezone"><small>{{tr('yourTimezone',{timezone:timezoneLabel(viewerTimezone)})}}</small><small v-if="accountingGroup">{{tr('groupTimezoneValue',{timezone:timezoneLabel(accountingGroup.timezone)})}}</small></div>
  </div>
  <div v-if="summary?.currencies?.length" class="currency-sections"><section v-for="item in summary.currencies" :key="item.currency" class="currency-panel"><header><strong>{{currencyLabel(item.currency)}}</strong><span>{{formatMonth(month)}}</span></header><div class="metric-strip" :class="{'metric-strip-wide':scope!=='personal'}"><article><small>{{tr('cashOutflow')}}</small><MoneyValue :amount="item.cashOutflowMinor" :currency="item.currency" /></article><article><small>{{tr('personalShare')}}</small><MoneyValue :amount="item.personalShareMinor" :currency="item.currency" /></article><article><small>{{tr('reimbursable')}}</small><MoneyValue :amount="item.reimbursableMinor" :currency="item.currency" /></article><article><small>{{tr('monthlySubscription')}}</small><MoneyValue :amount="item.monthlySubscriptionMinor" :currency="item.currency" /></article><article v-if="scope!=='personal'"><small>{{tr('personalMonthlySubscription')}}</small><MoneyValue :amount="item.personalMonthlySubscriptionMinor" :currency="item.currency" /></article><article><small>{{tr('activeSubscriptions')}}</small><strong>{{item.activeSubscriptions}}</strong></article></div></section></div>
  <EmptyState v-else :title="tr('noSummary')" :description="tr('noUpcomingDesc')" />
  <div class="dashboard-grid"><section class="card"><div class="card-title"><h2>{{tr('upcoming')}}</h2><RouterLink :to="actionSubscriptions">{{tr('manageSubscriptions')}} →</RouterLink></div><div v-if="summary?.upcoming?.length" class="data-list"><article v-for="item in summary.upcoming" :key="item.id" class="data-row"><div class="service-icon">{{item.name.slice(0,1)}}</div><div class="grow timezone-date"><strong>{{item.name}}</strong><small>{{viewerDate(item.nextBilling)}} · {{tr((item.lifecycleStatus||item.status) as 'active')}}</small><small v-if="item.groupId">{{originalTime(item.nextBilling,item.groupId)}}</small></div><span class="money"><MoneyValue :amount="item.amountMinor" :currency="item.currency" /><small v-if="upcomingShare(item)!==undefined">{{tr('personalShare')}}: <MoneyValue :amount="upcomingShare(item)!" :currency="item.currency" /></small></span></article></div><EmptyState v-else :title="tr('noUpcoming')" :description="tr('noUpcomingDesc')" /></section>
    <section v-if="scope==='group'" class="card"><div class="card-title"><div><h2>{{tr('balanceAsOfMonth')}}</h2><p class="setting-description">{{tr('balanceAsOfMonthDesc',{month:formatMonth(month)})}}</p></div><button class="ghost" @click="openSettlement">{{tr('recordSettlement')}}</button></div><div v-if="summary?.balances?.length" class="data-list"><article v-for="balance in summary.balances" :key="balance.userId" class="data-row"><div class="avatar">{{(workspace.members.find(m=>m.userId===balance.userId)?.user?.name||'?').slice(0,1)}}</div><div class="grow"><strong>{{workspace.members.find(m=>m.userId===balance.userId)?.user?.name||tr('unnamedMember')}}</strong><small>{{balance.amountMinor>0?tr('receivable'):balance.amountMinor<0?tr('payable'):tr('settled')}}</small></div><MoneyValue :amount="Math.abs(balance.amountMinor)" :currency="workspace.currentGroup?.currency" /></article></div><EmptyState v-else :title="tr('noBalances')" :description="tr('settled')" /></section>
  </div>
  <section v-if="scope==='group'" class="card settlement-history"><div class="card-title"><h2>{{tr('settlements')}}</h2><span>{{tr('records',{count:workspace.settlements.length})}}</span></div><div v-if="workspace.settlements.length" class="data-list"><article v-for="item in workspace.settlements" :key="item.id" class="data-row"><div class="grow"><strong>{{workspace.members.find(m=>m.userId===item.fromUserId)?.user?.name||tr('unnamedMember')}} → {{workspace.members.find(m=>m.userId===item.toUserId)?.user?.name||tr('unnamedMember')}}</strong><small>{{formatDate(item.settledOn)}} · {{item.notes}}</small><SyncBadge :pending-sync="item.pendingSync" :sync-error="item.syncError"/></div><MoneyValue :amount="item.amountMinor" :currency="workspace.currentGroup?.currency"/><button class="icon-button" :aria-label="tr('delete')" @click="pendingSettlementDelete=item">×</button></article></div><EmptyState v-else :title="tr('noSettlements')" :description="tr('recordSettlement')"/>
    <p v-if="workspace.settlements.some(item=>item.pendingSync||item.syncError)" class="field-help sync-legend"><strong>{{tr('syncLegendTitle')}}</strong> · ☁︎/ {{tr('syncLegendPending')}} · ⚠ {{tr('syncLegendError')}}</p>
  </section>
  <AppDrawer :open="settlementOpen" :title="tr('recordSettlement')" @close="settlementOpen=false"><form class="form-card" @submit.prevent="submitSettlement"><div v-if="settlementError" class="notice danger inline">{{settlementError}}</div><div v-if="workspace.currentGroup" class="timezone-notice">{{tr('yourTimezone',{timezone:timezoneLabel(viewerTimezone)})}}<br>{{tr('groupTimezoneValue',{timezone:timezoneLabel(workspace.currentGroup.timezone)})}}</div><label>{{tr('fromMember')}}<select v-model="settlementForm.fromUserId" required><option v-for="member in selectableMembers" :key="member.userId" :value="member.userId">{{member.user?.name||member.user?.email}}</option></select></label><label>{{tr('toMember')}}<select v-model="settlementForm.toUserId" required><option value="" disabled>{{tr('toMember')}}</option><option v-for="member in selectableMembers.filter(m=>m.userId!==settlementForm.fromUserId)" :key="member.userId" :value="member.userId">{{member.user?.name||member.user?.email}}</option></select></label><div class="form-row"><label>{{tr('amount')}}<input v-model="settlementForm.amount" type="number" :min="amountStep(workspace.currentGroup?.currency)" :step="amountStep(workspace.currentGroup?.currency)" required></label><label>{{tr('settlementDate')}}<input v-model="settlementForm.settledOn" type="date" required></label></div><label>{{tr('notes')}}<textarea v-model="settlementForm.notes" rows="3"></textarea></label><div class="form-actions"><button type="button" class="ghost" @click="settlementOpen=false">{{tr('cancel')}}</button><button class="primary">{{tr('recordSettlement')}}</button></div></form></AppDrawer>
  <ConfirmDialog :open="!!pendingSettlementDelete" :title="tr('deleteSettlementConfirm')" danger @cancel="pendingSettlementDelete=undefined" @confirm="deleteSettlement"/>
</section></template>

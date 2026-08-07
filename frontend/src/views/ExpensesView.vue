<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useRoute } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import { useWorkspaceStore } from '../stores/workspace'
import MoneyValue from '../components/MoneyValue.vue'
import EmptyState from '../components/EmptyState.vue'
import AppDrawer from '../components/AppDrawer.vue'
import ConfirmDialog from '../components/ConfirmDialog.vue'
import PersonalLedgerNav from '../components/PersonalLedgerNav.vue'
import CurrencySelect from '../components/CurrencySelect.vue'
import CategorySelect from '../components/CategorySelect.vue'
import type { Currency, Expense, ExpenseSplit, SplitMode } from '../api/types'
import { majorToMinor, minorToInput } from '../api/money'
import { useI18n } from '../i18n'
import { timezoneLabel } from '../timezone'

const workspace=useWorkspaceStore(),auth=useAuthStore(),route=useRoute(),{tr,formatDate}=useI18n()
const personal=computed(()=>route.name==='personal-expenses')
const list=computed(()=>personal.value?workspace.personalExpenses:workspace.expenses)
const open=ref(false),editingId=ref(''),pendingDelete=ref<Expense>()
const form=reactive({title:'',category:'',categoryId:'',amount:'',currency:'TWD' as Currency,rateMode:'automatic' as 'automatic'|'manual',exchangeRate:'',paidBy:'',incurredOn:new Date().toISOString().slice(0,10),notes:'',splitMode:'equal' as SplitMode,participants:{} as Record<string,boolean>,values:{} as Record<string,string>})
const editing=computed(()=>list.value.find(item=>item.id===editingId.value))
const sourceGroup=computed(()=>workspace.groups.find(group=>group.id===editing.value?.groupId))
const currency=computed(()=>form.currency||workspace.currentGroup?.currency||'TWD')
const participants=computed(()=>workspace.members.filter(member=>form.participants[member.userId]))
const splitTotalMinor=computed(()=>participants.value.reduce((sum,member)=>sum+(form.splitMode==='amount'?majorToMinor(form.values[member.userId]||'0',currency.value):form.splitMode==='percentage'?Math.round(Number(form.values[member.userId]||0)*100):0),0))
const expenseMinor=computed(()=>majorToMinor(form.amount||'0',currency.value))
const splitValid=computed(()=>personal.value&&!editing.value?.groupId||form.splitMode==='equal'?participants.value.length>0:form.splitMode==='amount'?splitTotalMinor.value===expenseMinor.value:splitTotalMinor.value===10000)

function reset(){editingId.value='';Object.assign(form,{title:'',category:'',categoryId:'',amount:'',currency:workspace.currentGroup?.currency||'TWD',rateMode:'automatic',exchangeRate:'',paidBy:'',incurredOn:new Date().toISOString().slice(0,10),notes:'',splitMode:'equal',participants:{},values:{}})}
async function loadFormCategories(){await workspace.loadCategories(personal.value&&!editing.value?.groupId?'personal':'group',editing.value?.groupId||workspace.currentGroupId)}
async function create(){reset();if(!personal.value)workspace.members.forEach(m=>{form.participants[m.userId]=true});await loadFormCategories();open.value=true}
async function edit(item:Expense){editingId.value=item.id;if(item.groupId&&workspace.currentGroupId!==item.groupId)await workspace.selectGroup(item.groupId);const selected=item.splits?.map(split=>split.userId)||workspace.members.map(m=>m.userId);Object.assign(form,{title:item.title,category:item.category,categoryId:item.categoryId||'',amount:minorToInput(item.amountMinor,item.currency),currency:item.currency||'TWD',rateMode:item.rateMode||'automatic',exchangeRate:item.exchangeRate||'',paidBy:item.paidBy,incurredOn:item.incurredOn.slice(0,10),notes:item.notes,splitMode:item.splitMode||'equal',participants:Object.fromEntries(selected.map(id=>[id,true])),values:Object.fromEntries((item.splits||[]).map(split=>[split.userId,(item.splitMode==='percentage'?(split.percentageBasisPoints||0)/100:minorToInput(split.amountMinor,item.currency)).toString()]))});await loadFormCategories();open.value=true}
async function addCategory(name:string){const value=await workspace.createCategory(personal.value&&!editing.value?.groupId?'personal':'group',name,editing.value?.groupId||workspace.currentGroupId);form.categoryId=value.id}
function canonicalSplits():ExpenseSplit[]{return participants.value.map(member=>({userId:member.userId,amountMinor:form.splitMode==='amount'?majorToMinor(form.values[member.userId]||'0',currency.value):0,percentageBasisPoints:form.splitMode==='percentage'?Math.round(Number(form.values[member.userId]||0)*100):undefined}))}
async function submit(){if(!splitValid.value)return;const input={title:form.title,category:form.category||'',categoryId:form.categoryId,amountMinor:expenseMinor.value,currency:currency.value,rateMode:form.rateMode,exchangeRate:form.exchangeRate,paidBy:form.paidBy||auth.record?.id||'',incurredOn:new Date(`${form.incurredOn}T00:00:00`).toISOString(),notes:form.notes,splitMode:form.splitMode,splits:personal.value&&!editing.value?.groupId?undefined:canonicalSplits()};if(editingId.value)await workspace.updateExpense(editingId.value,input);else if(personal.value)await workspace.addPersonalExpense(input);else await workspace.addExpense(input);await workspace.refreshPersonal();open.value=false;reset()}
async function remove(){if(!pendingDelete.value)return;await workspace.deleteExpense(pendingDelete.value.id);await workspace.refreshPersonal();pendingDelete.value=undefined}
function recordCurrency(item:Expense){return item.currency||workspace.groups.find(group=>group.id===item.groupId)?.currency||'TWD'}
function recordCategory(item:Expense){const category=item.categoryInfo;if(category?.systemKey)return tr((`category_${category.systemKey}`) as Parameters<typeof tr>[0]);return category?.customName||item.category||tr('uncategorized')}
function viewerTimezone(){return auth.record?.timezone||Intl.DateTimeFormat().resolvedOptions().timeZone||'UTC'}
function itemGroup(item:Expense){return workspace.groups.find(group=>group.id===item.groupId)}
function viewerDate(value:string){return formatDate(value,{dateStyle:'medium',timeZone:viewerTimezone()})}
function originalTime(item:Expense){const group=itemGroup(item);return group?tr('originalTimezone',{date:formatDate(item.incurredOn,{dateStyle:'medium',timeZone:group.timezone}),timezone:timezoneLabel(group.timezone,item.incurredOn)}):''}
onMounted(()=>{if(personal.value)void workspace.refreshPersonal()})
</script>

<template><section class="page ledger-page"><PersonalLedgerNav v-if="personal"/><div class="page-heading"><div><p class="eyebrow">{{tr(personal?'expensePersonal':'splitExpenses')}}</p><h1>{{tr(personal?'expensePersonal':'expenseGroup')}}</h1><p>{{tr('expenseDesc')}}</p></div><button class="primary" @click="create">{{tr('createExpense')}}</button></div>
  <section class="card data-card"><div class="card-title"><h2>{{tr('recentExpenses')}}</h2><span>{{tr('records',{count:list.length})}}</span></div><div v-if="list.length" class="data-table expense-table"><div class="data-table-head"><span>{{tr('item')}}</span><span>{{tr('source')}}</span><span>{{tr('payer')}}</span><span>{{tr('date')}}</span><span>{{tr('amount')}}</span><span></span></div><article v-for="item in list" :key="item.id" class="data-table-row"><div class="item-cell"><span class="service-icon expense">{{item.title.slice(0,1)}}</span><span><strong>{{item.title}}</strong><small>{{recordCategory(item)}}</small></span></div><span><span class="source-badge" :class="{shared:item.groupId}">{{itemGroup(item)?.name||tr('privateRecord')}}</span></span><span>{{workspace.members.find(m=>m.userId===item.paidBy)?.user?.name|| (item.paidBy===auth.record?.id?tr('myself'):'—')}}</span><span class="timezone-date"><strong>{{viewerDate(item.incurredOn)}}</strong><small v-if="item.groupId">{{originalTime(item)}}</small></span><span class="money-stack"><MoneyValue :amount="item.amountMinor" :currency="recordCurrency(item)"/><small v-if="item.baseCurrency&&item.baseCurrency!==item.currency">{{tr('reportingAmount')}}: <MoneyValue :amount="item.baseAmountMinor" :currency="item.baseCurrency"/></small><small v-if="item.exchangeRate">{{tr('exchangeRate')}} {{item.exchangeRate}}</small></span><span class="row-actions"><button class="icon-button" :aria-label="tr('edit')" @click="edit(item)">✎</button><button class="icon-button" :aria-label="tr('remove')" @click="pendingDelete=item">×</button></span></article></div><EmptyState v-else :title="tr('noExpenses')" :description="tr('noExpensesDesc')"/></section>
  <AppDrawer :open="open" :title="tr(editingId?'editExpense':'createExpense')" @close="open=false"><form class="form-card" @submit.prevent="submit">
    <div v-if="editing?.groupId&&personal" class="notice inline">{{tr('sharedRecordWarning',{group:sourceGroup?.name||tr('groups')})}}</div>
    <div v-if="sourceGroup||(!personal&&workspace.currentGroup)" class="timezone-notice">{{tr('yourTimezone',{timezone:timezoneLabel(viewerTimezone())})}}<br>{{tr('groupTimezoneValue',{timezone:timezoneLabel((sourceGroup||workspace.currentGroup)?.timezone||'UTC')})}}</div>
    <label>{{tr('item')}}<input v-model="form.title" required :placeholder="tr('itemPlaceholder')"></label>
    <div class="form-row"><label>{{tr('currency')}}<CurrencySelect v-model="form.currency" :currencies="workspace.currencies"/></label><label>{{tr('category')}}<CategorySelect v-model="form.categoryId" :categories="workspace.categories" @create="addCategory"/></label></div>
    <div class="form-row"><label>{{tr('amount')}}<input v-model="form.amount" type="number" min="0" step="0.01" required></label><label>{{tr('exchangeRate')}}<select v-model="form.rateMode"><option value="automatic">{{tr('automaticRate')}}</option><option value="manual">{{tr('manualRate')}}</option></select><input v-if="form.rateMode==='manual'" v-model="form.exchangeRate" inputmode="decimal" required></label></div>
    <div class="form-row"><label>{{tr('payer')}}<select v-model="form.paidBy"><option value="">{{tr('myself')}}</option><option v-for="member in workspace.members" :key="member.userId" :value="member.userId">{{member.user?.name||member.user?.email}}</option></select></label><label>{{tr('date')}}<input v-model="form.incurredOn" type="date" required></label></div>
    <template v-if="!personal||editing?.groupId"><label>{{tr('splitMode')}}<select v-model="form.splitMode"><option value="equal">{{tr('splitEqual')}}</option><option value="amount">{{tr('splitAmount')}}</option><option value="percentage">{{tr('splitPercentage')}}</option></select></label><fieldset class="split-editor"><legend>{{tr('participants')}}</legend><label v-for="member in workspace.members" :key="member.userId" class="split-member"><input v-model="form.participants[member.userId]" type="checkbox"><span>{{member.user?.name||member.user?.email}}</span><input v-if="form.participants[member.userId]&&form.splitMode!=='equal'" v-model="form.values[member.userId]" type="number" min="0" step="0.01" :aria-label="member.user?.name"></label><p :class="splitValid?'success':'form-error'">{{splitValid?tr('splitValid'):tr(form.splitMode==='percentage'?'splitInvalidPercentage':'splitInvalidAmount')}}</p></fieldset></template>
    <label>{{tr('notes')}}<textarea v-model="form.notes" rows="3"></textarea></label><div class="form-actions"><button type="button" class="ghost" @click="open=false">{{tr('cancel')}}</button><button class="primary" :disabled="workspace.loading||!splitValid">{{tr(editingId?'saveChanges':'createExpense')}}</button></div></form></AppDrawer>
  <ConfirmDialog :open="!!pendingDelete" :title="pendingDelete?tr('removeExpenseConfirm',{name:pendingDelete.title}):''" danger @cancel="pendingDelete=undefined" @confirm="remove"/>
</section></template>

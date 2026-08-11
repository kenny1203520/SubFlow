<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { useWorkspaceStore } from '../stores/workspace'
import { useAuthStore } from '../stores/auth'
import MoneyValue from '../components/MoneyValue.vue'
import EmptyState from '../components/EmptyState.vue'
import AppDrawer from '../components/AppDrawer.vue'
import ConfirmDialog from '../components/ConfirmDialog.vue'
import PersonalLedgerNav from '../components/PersonalLedgerNav.vue'
import CurrencySelect from '../components/CurrencySelect.vue'
import CategorySelect from '../components/CategorySelect.vue'
import PayerSelect from '../components/PayerSelect.vue'
import ConversionPreview from '../components/ConversionPreview.vue'
import SourceDialog from '../components/SourceDialog.vue'
import BaseCombobox from '../components/BaseCombobox.vue'
import type { BillingCycle, ExpenseSplit, SplitMode, Subscription, SubscriptionStatus } from '../api/types'
import { majorToMinor, minorToInput } from '../api/money'
import { useI18n } from '../i18n'
import { timezoneLabel } from '../timezone'
import { categoryGlyph, categoryLabel } from '../category'
import { fromDateInput, fromDateTimeInput, toDateInput, toDateTimeInput, todayInput } from '../dateInput'

const workspace = useWorkspaceStore(), auth = useAuthStore(), route = useRoute(), { tr, formatDate } = useI18n()
function viewerTimezone() { return auth.record?.timezone || Intl.DateTimeFormat().resolvedOptions().timeZone || 'UTC' }
// A bound temp member has already been superseded by the real account that
// joined, so new records should use that real member instead — but keep
// showing bound placeholders everywhere history is displayed (member.user
// still resolves their name), just not as a choice for new participation.
const selectableMembers = computed(() => workspace.members.filter(member => !(member.user?.placeholder && member.user?.linkedUserId)))
const personal = computed(() => route.name === 'personal-subscriptions')
const list = computed(() => personal.value ? workspace.personalSubscriptions : workspace.subscriptions)
const drawer = ref(false), editingId = ref(''), pendingDelete = ref<Subscription>(), sourceItem = ref<Subscription>(), stopping = ref<Subscription>()
const dates = ref<string[]>([]), cursor = ref(''), chosenDate = ref(''), datesLoading = ref(false), formError = ref(''), rateValid = ref(true)
const revisionDates = ref<string[]>([]), revisionCursor = ref(''), revisionDatesLoading = ref(false)
const form = reactive({ name:'', category:'', categoryId:'', amount:'', currency:'TWD', rateMode:'automatic' as 'automatic'|'manual', exchangeRate:'', paidBy:'', splitMode:'equal' as SplitMode, participants:{} as Record<string,boolean>, values:{} as Record<string,string>, revisionScope:'future' as 'future'|'one_off', effectiveBillingAt:'', billingCycle:'monthly' as BillingCycle, billingInterval:1, startsOn:todayInput(viewerTimezone()), status:'active' as SubscriptionStatus, notes:'' })
// Set only when the user actually edits the date input, so an untouched
// startsOn is omitted from the update payload and the backend keeps the
// exact original instant instead of a same-day-but-shifted reconstruction.
const startsOnTouched = ref(false)
const cycleOptions = computed(() => ([
  { value:'daily', label:tr('daily') }, { value:'every_n_days', label:tr('everyNDays') },
  { value:'weekly', label:tr('weekly') }, { value:'every_n_weeks', label:tr('everyNWeeks') },
  { value:'every_n_hours', label:tr('everyNHours') }, { value:'monthly', label:tr('monthly') },
  { value:'quarterly', label:tr('quarterly') }, { value:'yearly', label:tr('yearly') },
]))
const needsInterval = computed(() => ['every_n_days','every_n_weeks','every_n_hours'].includes(form.billingCycle))
const hourlyCycle = computed(() => form.billingCycle === 'every_n_hours')
const editing = computed(() => list.value.find(item => item.id === editingId.value))
const sourceGroup = computed(() => workspace.groups.find(group => group.id === editing.value?.groupId))
const reportingCurrency = computed(() => sourceGroup.value?.currency || (!personal.value ? workspace.currentGroup?.currency : form.currency) || form.currency)
const currency = computed(() => form.currency || workspace.currentGroup?.currency || 'TWD')
const participants = computed(() => workspace.members.filter(member => form.participants[member.userId]))
const splitTotalMinor = computed(() => participants.value.reduce((sum, member) => sum + (form.splitMode === 'amount' ? majorToMinor(form.values[member.userId] || '0', currency.value) : form.splitMode === 'percentage' ? Math.round(Number(form.values[member.userId] || 0) * 100) : 0), 0))
const subscriptionMinor = computed(() => majorToMinor(form.amount || '0', currency.value))
const splitValid = computed(() => personal.value && !editing.value?.groupId || form.splitMode === 'equal' ? participants.value.length > 0 : form.splitMode === 'amount' ? splitTotalMinor.value === subscriptionMinor.value : splitTotalMinor.value === 10000)
watch(() => [form.currency, reportingCurrency.value], ([from, to]) => { if (from === to) { form.rateMode = 'automatic'; form.exchangeRate = '' } })

function reset() { editingId.value=''; formError.value=''; rateValid.value=true; revisionDates.value=[]; revisionCursor.value=''; startsOnTouched.value=false; Object.assign(form,{name:'',category:'',categoryId:'',amount:'',currency:workspace.currentGroup?.currency||auth.record?.defaultCurrency||'TWD',rateMode:'automatic',exchangeRate:'',paidBy:auth.record?.id||'',splitMode:'equal',participants:{},values:{},revisionScope:'future',effectiveBillingAt:'',billingCycle:'monthly',billingInterval:1,startsOn:todayInput(viewerTimezone()),status:'active',notes:''}) }
async function loadFormCategories() { try { await workspace.loadCategories(personal.value && !editing.value?.groupId ? 'personal' : 'group', editing.value?.groupId || workspace.currentGroupId) } catch { formError.value = workspace.localizedError || tr('requestFailed') } }
async function create() { reset(); if(!personal.value) selectableMembers.value.forEach(member=>{form.participants[member.userId]=true}); drawer.value = true; await loadFormCategories() }
function startInput(item: Subscription) { const value=item.startsOn||item.nextBilling; return item.billingCycle==='every_n_hours' ? toDateTimeInput(value,viewerTimezone()) : toDateInput(value,viewerTimezone()) }
const canEditHistory = computed(() => workspace.groupPermissions.includes('*') || workspace.groupPermissions.includes('ledger.records.historical_write'))
async function loadRevisionDates(item: Subscription, more=false) { revisionDatesLoading.value=true; try { const result=await workspace.billingDates(item.id,more?revisionCursor.value:'',canEditHistory.value); revisionDates.value=more?[...revisionDates.value,...result.dates]:result.dates; revisionCursor.value=result.nextCursor||''; if(!form.effectiveBillingAt) form.effectiveBillingAt=revisionDates.value[0]||item.nextBilling } catch { formError.value=workspace.localizedError||tr('requestFailed') } finally { revisionDatesLoading.value=false } }
const revisionDateOptions = computed(() => revisionDates.value.map(value => ({ value, label: pastBillingDate(value) ? `${viewerDate(value)} · ${tr('subscriptionPastPeriod')}` : viewerDate(value), searchText: `${viewerDate(value)} ${value}` })))
// A closed period may only be revised on its own; rewriting "this and future"
// from the past would also change what the subscription bills from now on.
function pastBillingDate(value: string) { return !!editing.value?.nextBilling && new Date(value) < new Date(editing.value.nextBilling) }
const editingPastPeriod = computed(() => !!form.effectiveBillingAt && pastBillingDate(form.effectiveBillingAt))
const scopeOptions = computed(() => editingPastPeriod.value ? [{ value:'one_off', label:tr('subscriptionOnlyThisPeriod') }] : [{ value:'one_off', label:tr('subscriptionOnlyThisPeriod') }, { value:'future', label:tr('subscriptionThisAndFuture') }])
watch(editingPastPeriod, past => { if (past) form.revisionScope = 'one_off' })
async function edit(item: Subscription) { reset(); editingId.value=item.id; if(item.groupId&&workspace.currentGroupId!==item.groupId) await workspace.selectGroup(item.groupId); const selected=item.splits?.map(split=>split.userId)||workspace.members.map(member=>member.userId); Object.assign(form,{name:item.name,category:item.category,categoryId:item.categoryId||'',amount:minorToInput(item.amountMinor,item.currency),currency:item.currency,rateMode:item.rateMode||'automatic',exchangeRate:item.exchangeRate||'',paidBy:item.paidBy||auth.record?.id||'',splitMode:item.splitMode||'equal',participants:Object.fromEntries(selected.map(id=>[id,true])),values:Object.fromEntries((item.splits||[]).map(split=>[split.userId,(item.splitMode==='percentage'?(split.percentageBasisPoints||0)/100:minorToInput(split.amountMinor,item.currency)).toString()])),revisionScope:'future',effectiveBillingAt:item.nextBilling||item.startsOn,billingCycle:item.billingCycle,billingInterval:item.billingInterval||1,startsOn:startInput(item),status:item.status,notes:item.notes}); drawer.value=true; await Promise.all([loadFormCategories(), item.groupId ? loadRevisionDates(item) : Promise.resolve()]) }
async function addCategory(name: string, icon = 'tag') { try { const value = await workspace.createCategory(personal.value && !editing.value?.groupId ? 'personal' : 'group',name,editing.value?.groupId || workspace.currentGroupId,icon); form.categoryId = value.id } catch { formError.value = workspace.localizedError || tr('requestFailed') } }
function canonicalSplits():ExpenseSplit[]{return participants.value.map(member=>({userId:member.userId,amountMinor:form.splitMode==='amount'?majorToMinor(form.values[member.userId]||'0',currency.value):0,percentageBasisPoints:form.splitMode==='percentage'?Math.round(Number(form.values[member.userId]||0)*100):undefined}))}
async function submit() {
  if (!rateValid.value || !splitValid.value) return
  const startsOn = hourlyCycle.value ? fromDateTimeInput(form.startsOn,viewerTimezone()) : fromDateInput(form.startsOn,viewerTimezone())
  const effectiveBillingAt = form.effectiveBillingAt || undefined
  const input = { name:form.name, category:form.category||'', categoryId:form.categoryId, amountMinor:subscriptionMinor.value, currency:currency.value as Subscription['currency'], rateMode:form.rateMode, exchangeRate:form.exchangeRate, paidBy:form.paidBy||auth.record?.id||'', splitMode:form.splitMode, splits:personal.value&&!editing.value?.groupId?undefined:canonicalSplits(), revisionScope:form.revisionScope, effectiveBillingAt, billingCycle:form.billingCycle, billingInterval:needsInterval.value ? Number(form.billingInterval) : 1, ...(!editingId.value||startsOnTouched.value?{startsOn}:{}), status:form.status, notes:form.notes }
  const ok = editingId.value ? await workspace.updateSubscription(editingId.value,input) : personal.value ? await workspace.addPersonalSubscription(input) : await workspace.addSubscription(input)
  if (!ok) { formError.value = workspace.localizedError || tr('requestFailed'); return }
  await workspace.refreshPersonal()
  drawer.value = false
  reset()
}
async function loadDates(more=false) { if(!stopping.value) return; datesLoading.value=true; try { const result=await workspace.billingDates(stopping.value.id,more?cursor.value:''); dates.value=more?[...dates.value,...result.dates]:result.dates; cursor.value=result.nextCursor||''; if(!chosenDate.value) chosenDate.value=dates.value[0]||'' } finally { datesLoading.value=false } }
async function openStop(item: Subscription) { stopping.value=item; dates.value=[]; cursor.value=''; chosenDate.value=''; await loadDates() }
async function confirmStop() { if(!stopping.value||!chosenDate.value) return; await workspace.stopSubscription(stopping.value.id,chosenDate.value); stopping.value=undefined }
async function cancelStop(item: Subscription) { await workspace.cancelSubscriptionStop(item.id) }
async function remove() { if(!pendingDelete.value) return; await workspace.deleteSubscription(pendingDelete.value.id); await workspace.refreshPersonal(); pendingDelete.value=undefined }
function statusKey(item: Subscription) { return (item.lifecycleStatus||item.status) as 'active' }
function cycleKey(item: Subscription): BillingCycle { return item.billingCycle }
function cycleLabel(item: Subscription) { const key = cycleKey(item); return ['every_n_days','every_n_weeks','every_n_hours'].includes(key) ? tr(key === 'every_n_days' ? 'everyNDaysValue' : key === 'every_n_weeks' ? 'everyNWeeksValue' : 'everyNHoursValue',{count:item.billingInterval||1}) : tr(key) }
function recordCategory(item: Subscription) { return `${categoryGlyph(item.categoryInfo)} ${categoryLabel(item.categoryInfo,item.category,tr)}` }
function itemGroup(item: Subscription) { return workspace.groups.find(group => group.id === item.groupId) }
function viewerDate(value: string) { return formatDate(value,{dateStyle:'medium',timeZone:viewerTimezone()}) }
function originalTime(item: Subscription,value=item.nextBilling) { const group=itemGroup(item); return group?tr('originalTimezone',{date:formatDate(value,{dateStyle:'medium',timeZone:group.timezone}),timezone:timezoneLabel(group.timezone,value)}):'' }
watch(()=>form.billingCycle, cycle=>{ if(cycle==='every_n_hours'&&form.startsOn.length===10) form.startsOn=`${form.startsOn}T00:00`; if(cycle!=='every_n_hours'&&form.startsOn.length>10) form.startsOn=form.startsOn.slice(0,10); if(!['every_n_days','every_n_weeks','every_n_hours'].includes(cycle)) form.billingInterval=1 })
onMounted(() => { if(personal.value) void workspace.refreshPersonal() })
</script>

<template>
  <section class="page ledger-page">
    <PersonalLedgerNav v-if="personal" />
    <div class="page-heading"><div><p class="eyebrow">{{ tr('subscriptions') }}</p><h1>{{ tr(personal ? 'subscriptionPersonal' : 'subscriptionGroup') }}</h1><p>{{ tr('subscriptionDesc') }}</p></div><button class="primary" @click="create">{{ tr('createSubscription') }}</button></div>
    <section class="card data-card">
      <div class="card-title"><h2>{{ tr('allSubscriptions') }}</h2><span>{{ tr('records',{count:list.length}) }}</span></div>
      <div v-if="!personal && workspace.groupErrors.subscriptions" class="resource-error"><p>{{ workspace.groupErrors.subscriptions }}</p><button class="ghost" @click="workspace.refreshGroup()">{{ tr('retry') }}</button></div>
      <div v-else-if="list.length" class="data-table subscription-table">
        <div class="data-table-head"><span>{{tr('name')}}</span><span>{{tr('source')}}</span><span>{{tr('nextBilling')}}</span><span>{{tr('cycle')}}</span><span>{{tr('status')}}</span><span>{{tr('amount')}}</span><span></span></div>
        <article v-for="item in list" :key="item.id" class="data-table-row">
          <div class="item-cell"><span class="service-icon">{{item.name.slice(0,1)}}</span><span><strong>{{item.name}}</strong><small>{{recordCategory(item)}}</small></span></div>
          <span><button class="source-badge" :class="{shared:item.groupId}" @click="sourceItem=item">{{itemGroup(item)?.name||tr('privateRecord')}}</button></span>
          <span class="timezone-date"><strong>{{viewerDate(item.nextBilling)}}</strong><small v-if="item.groupId">{{originalTime(item)}}</small></span><span>{{cycleLabel(item)}}</span><span class="pill">{{tr(statusKey(item))}}</span>
          <span class="money-stack"><MoneyValue :amount="item.amountMinor" :currency="item.currency"/><small v-if="item.baseCurrency&&item.baseCurrency!==item.currency">{{tr('reportingAmount')}}: <MoneyValue :amount="item.baseAmountMinor" :currency="item.baseCurrency"/></small><small v-if="item.exchangeRate">{{tr('exchangeRate')}} {{item.exchangeRate}}</small></span>
          <span class="row-actions"><button v-if="item.endsOn&&item.lifecycleStatus==='ending'" class="ghost" @click="cancelStop(item)">{{tr('cancelStop')}}</button><button v-else-if="item.lifecycleStatus!=='ended'&&item.lifecycleStatus!=='cancelled'" class="ghost" @click="openStop(item)">{{tr('stop')}}</button><button class="icon-button" :aria-label="tr('edit')" @click="edit(item)">✎</button><button class="icon-button" :aria-label="tr('delete')" @click="pendingDelete=item">×</button></span>
        </article>
      </div>
      <EmptyState v-else :title="tr('noSubscriptions')" :description="tr('noSubscriptionsDesc')"/>
    </section>
    <AppDrawer :open="drawer" :title="tr(editingId?'editSubscription':'createSubscription')" @close="drawer=false"><form class="form-card ledger-form" @submit.prevent="submit">
      <div v-if="formError" class="notice danger inline">{{formError}}</div><div v-if="editing?.groupId&&personal" class="notice inline">{{tr('sharedRecordWarning',{group:sourceGroup?.name||tr('groups')})}}</div><div v-if="sourceGroup||(!personal&&workspace.currentGroup)" class="timezone-notice">{{tr('yourTimezone',{timezone:timezoneLabel(viewerTimezone())})}}<br>{{tr('groupTimezoneValue',{timezone:timezoneLabel((sourceGroup||workspace.currentGroup)?.timezone||'UTC')})}}</div>
      <section class="ledger-form-section"><div class="ledger-section-heading"><strong>{{tr('name')}}</strong></div><div class="ledger-form-grid"><label class="ledger-wide">{{tr('name')}}<input v-model="form.name" required :placeholder="tr('namePlaceholder')"></label><label class="ledger-wide">{{tr('category')}}<CategorySelect v-model="form.categoryId" :categories="workspace.categories" @create="addCategory"/></label></div></section>
      <section class="ledger-form-section"><div class="ledger-section-heading"><strong>{{tr('amount')}} · {{tr('currency')}}</strong></div><div class="ledger-form-grid"><label>{{tr('amount')}}<input v-model="form.amount" type="number" min="0" step="0.01" required></label><label>{{tr('currency')}}<CurrencySelect v-model="form.currency" :currencies="workspace.currencies"/></label><label v-if="!personal">{{tr('payer')}}<PayerSelect v-model="form.paidBy" :members="selectableMembers" :self-id="auth.record?.id"/></label></div></section>
      <section class="ledger-form-section"><div class="ledger-section-heading"><strong>{{tr('exchangeRate')}}</strong></div><div class="ledger-form-grid"><label>{{tr('exchangeRate')}}<select v-model="form.rateMode" :disabled="form.currency===reportingCurrency"><option value="automatic">{{tr('automaticRate')}}</option><option value="manual">{{tr('manualRate')}}</option></select></label><label v-if="form.rateMode==='manual'&&form.currency!==reportingCurrency">{{tr('manualRate')}}<input v-model="form.exchangeRate" inputmode="decimal" required></label><div v-else class="ledger-rate-placeholder"></div><ConversionPreview class="ledger-wide" :from="form.currency" :to="reportingCurrency" :amount="form.amount" :date="form.startsOn.slice(0,10)" :mode="form.rateMode" :manual-rate="form.exchangeRate" @validity="rateValid=$event"/></div></section>
      <section v-if="!personal||editing?.groupId" class="ledger-form-section"><div class="ledger-section-heading"><strong>{{tr('splitMode')}}</strong></div><label>{{tr('splitMode')}}<select v-model="form.splitMode"><option value="equal">{{tr('splitEqual')}}</option><option value="amount">{{tr('splitAmount')}}</option><option value="percentage">{{tr('splitPercentage')}}</option></select></label><fieldset class="split-editor"><legend>{{tr('participants')}}</legend><label v-for="member in selectableMembers" :key="member.userId" class="split-member"><input v-model="form.participants[member.userId]" type="checkbox"><span>{{member.user?.name||member.user?.email}}</span><input v-if="form.participants[member.userId]&&form.splitMode!=='equal'" v-model="form.values[member.userId]" type="number" min="0" step="0.01" :aria-label="member.user?.name"></label><p :class="splitValid?'success':'form-error'">{{splitValid?tr('splitValid'):tr(form.splitMode==='percentage'?'splitInvalidPercentage':'splitInvalidAmount')}}</p></fieldset></section>
      <section class="ledger-form-section"><div class="ledger-section-heading"><strong>{{tr('cycle')}}</strong></div><div class="ledger-form-grid"><BaseCombobox v-model="form.billingCycle" :options="cycleOptions" :label="tr('cycle')" :allow-create="false"/><label v-if="needsInterval">{{tr('billingInterval')}}<input v-model.number="form.billingInterval" type="number" min="1" max="8760" required></label><label>{{tr('firstBilling')}}<input v-model="form.startsOn" :type="hourlyCycle?'datetime-local':'date'" required @change="startsOnTouched=true"></label><label>{{tr('status')}}<select v-model="form.status"><option value="active">{{tr('active')}}</option><option value="paused">{{tr('paused')}}</option><option value="cancelled">{{tr('cancelled')}}</option></select></label></div></section><section v-if="editing?.groupId" class="ledger-form-section"><div class="ledger-section-heading"><strong>{{tr('subscriptionChangeScope')}}</strong></div><div class="ledger-form-grid"><BaseCombobox v-model="form.revisionScope" :options="scopeOptions" :label="tr('subscriptionChangeScope')" :allow-create="false" :help="editingPastPeriod?tr('subscriptionPastScopeLocked'):''"/><BaseCombobox v-model="form.effectiveBillingAt" :options="revisionDateOptions" :label="tr('effectiveBillingDate')" :help="canEditHistory?tr('subscriptionHistoricalHelp'):tr('subscriptionChangeScopeHelp')" :allow-create="false" :disabled="revisionDatesLoading"/></div><button v-if="revisionCursor" type="button" class="ghost" :disabled="revisionDatesLoading" @click="editing&&loadRevisionDates(editing,true)">{{tr('loadMore')}}</button></section><section class="ledger-form-section"><label>{{tr('notes')}}<textarea v-model="form.notes" rows="3"></textarea></label></section><div class="form-actions ledger-form-actions"><button type="button" class="ghost" @click="drawer=false">{{tr('cancel')}}</button><button class="primary" :disabled="workspace.loading||!rateValid||!splitValid||!!editing?.groupId&&!form.effectiveBillingAt">{{tr(editingId?'saveChanges':'createSubscription')}}</button></div>
    </form></AppDrawer>
    <Teleport to="body"><div v-if="stopping" class="modal-backdrop" @click.self="stopping=undefined"><section class="billing-dialog" role="dialog" aria-modal="true" :aria-label="tr('stopSubscription')"><header><div><h2>{{tr('stopSubscription')}}</h2><p>{{stopping.name}} · {{tr('chooseFinalBilling')}}</p></div><button class="icon-button" :aria-label="tr('close')" @click="stopping=undefined">×</button></header><div v-if="dates.length" class="billing-dates"><label v-for="date in dates" :key="date" :class="{selected:chosenDate===date}"><input v-model="chosenDate" type="radio" :value="date"><span><strong>{{viewerDate(date)}}</strong><small>{{tr('finalBilling')}}</small><small v-if="stopping.groupId">{{originalTime(stopping,date)}}</small></span></label></div><EmptyState v-else :title="tr('noBillingDates')" :description="tr('subscriptionDesc')"/><button v-if="cursor" class="ghost wide" :disabled="datesLoading" @click="loadDates(true)">{{tr('loadMore')}}</button><div class="form-actions"><button class="ghost" @click="stopping=undefined">{{tr('cancel')}}</button><button class="primary" :disabled="!chosenDate" @click="confirmStop">{{tr('confirm')}}</button></div></section></div></Teleport>
    <SourceDialog :open="!!sourceItem" kind="subscriptions" :group="sourceItem?itemGroup(sourceItem):undefined" :currency="sourceItem?.currency||'TWD'" @close="sourceItem=undefined"/><ConfirmDialog :open="!!pendingDelete" :title="pendingDelete?tr('deleteSubscriptionConfirm',{name:pendingDelete.name}):''" danger @cancel="pendingDelete=undefined" @confirm="remove"/>
  </section>
</template>

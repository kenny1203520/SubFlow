<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { RouterLink } from 'vue-router'
import { ApiClient } from '../api/client'
import type { HomeDashboard, SettlementReview } from '../api/types'
import EmptyState from '../components/EmptyState.vue'
import MoneyValue from '../components/MoneyValue.vue'
import { useAuthStore } from '../stores/auth'
import { useI18n } from '../i18n'

const auth = useAuthStore()
const api = new ApiClient(() => auth.token, () => auth.logout())
const { tr, formatDate } = useI18n()
const data = ref<HomeDashboard>()
const error = ref('')
const busy = ref(false)
const upcomingDays = ref(7)
const rejectingId = ref('')
const rejectionReason = ref('')
const approvingId = ref('')
const approvalReason = ref('')
function dateKey(value: Date, timeZone: string) {
  const parts = new Intl.DateTimeFormat('en', { year: 'numeric', month: '2-digit', day: '2-digit', timeZone }).formatToParts(value)
  const part = (type: Intl.DateTimeFormatPartTypes) => parts.find(value => value.type === type)?.value || '00'
  return `${part('year')}-${part('month')}-${part('day')}`
}
function addDays(value: string, days: number) {
  const [year, month, day] = value.split('-').map(Number)
  return new Date(Date.UTC(year, month - 1, day + days)).toISOString().slice(0, 10)
}
const visibleUpcoming = computed(() => (data.value?.upcoming || []).filter(item => {
  const zone = item.timezone || auth.record?.timezone || Intl.DateTimeFormat().resolvedOptions().timeZone
  const today = dateKey(new Date(), zone)
  const date = dateKey(new Date(item.nextBilling), zone)
  return date >= today && date <= addDays(today, upcomingDays.value)
}))

async function load() {
  busy.value = true
  error.value = ''
  try { data.value = (await api.get<HomeDashboard>('/home')).data }
  catch { error.value = tr('homeLoadFailed') }
  finally { busy.value = false }
}

async function respond(review: SettlementReview, decision: 'accept'|'reject') {
  const reason = decision === 'reject' ? rejectionReason.value.trim() : approvalReason.value.trim()
  if (decision === 'reject' && !reason) return
  if (decision === 'accept' && requiresManagerReason(review) && !reason) return
  busy.value = true
  error.value = ''
  try {
    await api.post(`/settlement-reviews/${encodeURIComponent(review.id)}/respond`, { decision, reason })
    rejectingId.value = ''
    approvingId.value = ''
    rejectionReason.value = ''
    approvalReason.value = ''
    await load()
  } catch { error.value = tr('homeActionFailed') }
  finally { busy.value = false }
}

function requiresManagerReason(review: SettlementReview) {
  return !!review.canManage && review.proposal.toUserId !== auth.record?.id
}

function requestApprove(review: SettlementReview) {
  if (requiresManagerReason(review)) {
    approvingId.value = review.id
    approvalReason.value = ''
    rejectingId.value = ''
    return
  }
  void respond(review, 'accept')
}

async function withdraw(review: SettlementReview) {
  busy.value = true
  error.value = ''
  try {
    await api.post(`/settlement-reviews/${encodeURIComponent(review.id)}/withdraw`, {})
    await load()
  } catch { error.value = tr('homeActionFailed') }
  finally { busy.value = false }
}

function reviewTitle(review: SettlementReview) {
  if (review.action === 'change') return tr('settlementChangeReview')
  if (review.action === 'void') return tr('settlementVoidReview')
  return tr('settlementNewReview')
}

function actionTitle(action: HomeDashboard['actions'][number]) {
  if (action.type === 'group_invitation') return tr('pendingInvitations')
  return action.title || tr(action.type)
}

function actionPath(action: HomeDashboard['actions'][number]) {
  const groupPath = action.groupId ? '/groups/' + action.groupId : ''
  if (action.type === 'group_invitation') return '/notifications'
  if (action.type === 'ownership_transfer') return groupPath + '/settings'
  if (action.type === 'member_transfer') return groupPath + '/members'
  if (action.type === 'subscription_post_failed') return (groupPath || '/personal') + '/subscriptions'
  if (action.type === 'recurring_income_post_failed') return (groupPath || '/personal') + '/incomes'
  if (action.type === 'recurring_income_manual_pending') return (groupPath || '/personal') + '/incomes?tab=pending'
  return '/notifications'
}

function totals(item: {incomeReceivedMinor:number;expensePaidMinor:number;settlementReceivedMinor:number;settlementPaidMinor:number}) {
  return { received:item.incomeReceivedMinor + item.settlementReceivedMinor, paid:item.expensePaidMinor + item.settlementPaidMinor }
}

onMounted(() => void load())
</script>

<template>
  <section class="page home-page">
    <header class="page-heading home-heading">
      <div><p class="eyebrow">{{ tr('home') }}</p><h1>{{ tr('homeWelcome',{name:auth.name}) }}</h1><p>{{ tr('homeDesc') }}</p></div>
      <div class="home-quick-actions"><RouterLink class="primary" to="/personal/expenses">{{ tr('addExpense') }}</RouterLink><RouterLink class="ghost" to="/personal/incomes">{{ tr('addIncome') }}</RouterLink></div>
    </header>

    <div v-if="error" class="notice danger home-error" role="alert"><span>{{ error }}</span><button type="button" :disabled="busy" @click="load">{{ tr('retry') }}</button></div>
    <div v-if="busy&&!data" class="card"><p>{{ tr('loading') }}</p></div>

    <template v-if="data">
      <section class="home-section" aria-labelledby="home-balances-title">
        <div class="home-section-heading"><div><p class="eyebrow">{{ tr('asOf',{date:formatDate(data.asOf,{dateStyle:'medium',timeStyle:'short',timeZone:auth.record?.timezone||undefined})}) }}</p><h2 id="home-balances-title">{{ tr('homeOutstanding') }}</h2></div><RouterLink class="text-link" to="/groups">{{ tr('openGroups') }} →</RouterLink></div>
        <div class="home-balance-columns">
          <article class="card home-balance-card receivable-card"><div class="home-card-label"><span class="balance-mark received-mark" aria-hidden="true">↓</span><h3>{{ tr('homeToReceive') }}</h3></div><div v-if="data.receivables.length" class="home-currency-list"><div v-for="item in data.receivables" :key="item.currency"><span>{{ item.currency }}</span><MoneyValue :amount="item.amountMinor" :currency="item.currency" /></div></div><p v-else class="home-muted">{{ tr('homeNoOutstanding') }}</p></article>
          <article class="card home-balance-card payable-card"><div class="home-card-label"><span class="balance-mark paid-mark" aria-hidden="true">↑</span><h3>{{ tr('homeToPay') }}</h3></div><div v-if="data.payables.length" class="home-currency-list"><div v-for="item in data.payables" :key="item.currency"><span>{{ item.currency }}</span><MoneyValue :amount="item.amountMinor" :currency="item.currency" /></div></div><p v-else class="home-muted">{{ tr('homeNoOutstanding') }}</p></article>
        </div>
        <div v-if="data.groups.length" class="home-group-positions"><RouterLink v-for="group in data.groups" :key="group.groupId" class="home-group-position" :to="`/groups/${group.groupId}/overview`"><span><strong>{{ group.groupName }}</strong><small v-if="group.balanceAvailable">{{ group.currency }} / {{ group.balanceMinor>0?tr('homeGroupReceivable'):group.balanceMinor<0?tr('homeGroupPayable'):tr('settled') }}</small><small v-else>{{ tr('homeGroupIncomplete') }}</small></span><MoneyValue v-if="group.balanceAvailable" :amount="Math.abs(group.balanceMinor)" :currency="group.currency" /><span class="row-arrow" aria-hidden="true">›</span></RouterLink></div>
        <p v-if="data.incompleteGroupIds?.length" class="field-help" role="status">{{ tr('homeIncompleteGroups',{count:data.incompleteGroupIds.length}) }}</p>
      </section>

      <section class="home-section" aria-labelledby="home-actions-title">
        <div class="home-section-heading"><div><p class="eyebrow">{{ tr('homeActionEyebrow') }}</p><h2 id="home-actions-title">{{ tr('homeNeedsAttention') }}</h2></div><RouterLink class="text-link" to="/notifications">{{ tr('notificationCenter') }} →</RouterLink></div>
        <div v-if="data.reviews.length||data.actions.length" class="card home-action-list">
          <article v-for="review in data.reviews" :key="review.id" class="home-action-item">
            <div class="action-icon" aria-hidden="true">&#8596;</div>
            <div class="home-action-copy">
              <strong>{{ reviewTitle(review) }}</strong>
              <small>{{ review.fromName||review.proposal.fromUserId }} &rarr; {{ review.toName||review.proposal.toUserId }} &middot; {{ review.groupName||data.groups.find(g=>g.groupId===review.groupId)?.groupName||tr('groups') }}</small>
              <span><MoneyValue :amount="review.proposal.amountMinor" :currency="review.proposal.currency"/></span>
            </div>
            <div v-if="review.canRespond" class="home-action-controls">
              <button class="primary" type="button" :disabled="busy" @click="requestApprove(review)">{{ tr('settlementApprove') }}</button>
              <button class="ghost" type="button" :disabled="busy" @click="rejectingId=review.id;approvingId='';rejectionReason=''">{{ tr('settlementReject') }}</button>
            </div>
            <div v-else class="home-action-controls home-waiting-controls">
              <span role="status">{{ tr('homeWaitingForOther') }}</span>
              <button v-if="review.submittedBy===auth.record?.id" class="ghost" type="button" :disabled="busy" @click="withdraw(review)">{{ tr('withdrawRequest') }}</button>
            </div>
            <form v-if="rejectingId===review.id" class="home-reject-form" @submit.prevent="respond(review,'reject')">
              <label>{{ tr('rejectionReason') }}<textarea v-model="rejectionReason" rows="2" required maxlength="1000"></textarea></label>
              <div class="form-actions"><button class="ghost" type="button" @click="rejectingId='';rejectionReason=''">{{ tr('cancel') }}</button><button class="primary" :disabled="busy||!rejectionReason.trim()">{{ tr('settlementReject') }}</button></div>
            </form>
            <form v-if="approvingId===review.id" class="home-reject-form" @submit.prevent="respond(review,'accept')">
              <label>{{ tr('managerApprovalReason') }}<textarea v-model="approvalReason" rows="2" required maxlength="1000"></textarea></label>
              <div class="form-actions"><button class="ghost" type="button" @click="approvingId='';approvalReason=''">{{ tr('cancel') }}</button><button class="primary" :disabled="busy||!approvalReason.trim()">{{ tr('settlementApprove') }}</button></div>
            </form>
          </article>
          <RouterLink v-for="action in data.actions" :key="action.id" class="home-action-item home-notification-action" :to="actionPath(action)"><div class="action-icon" aria-hidden="true">!</div><div class="home-action-copy"><strong>{{ actionTitle(action) }}</strong><small v-if="action.type==='recurring_income_manual_pending'">{{ tr('homeManualIncomePrompt',{date:formatDate(action.scheduledOn||'',{dateStyle:'medium'})}) }}<template v-if="action.message"> / {{ action.message }}</template></small><small v-else>{{ action.message||tr('homeOpenToResolve') }}</small></div><MoneyValue v-if="action.currency" :amount="action.amountMinor||0" :currency="action.currency"/><span class="row-arrow" aria-hidden="true">›</span></RouterLink>
        </div>
        <div v-else class="card home-empty"><EmptyState :title="tr('homeNoActions')" :description="tr('homeNoActionsDesc')"/></div>
      </section>

      <section class="home-section" aria-labelledby="home-upcoming-title">
        <div class="home-section-heading"><div><p class="eyebrow">{{ tr('homePlanAhead') }}</p><h2 id="home-upcoming-title">{{ tr('homeUpcoming') }}</h2></div><label class="home-range-select"><span>{{ tr('period') }}</span><select v-model.number="upcomingDays"><option :value="7">{{ tr('nextSevenDays') }}</option><option :value="30">{{ tr('nextThirtyDays') }}</option></select></label></div>
        <div v-if="visibleUpcoming.length" class="card home-upcoming-list"><article v-for="item in visibleUpcoming" :key="item.id" class="home-upcoming-item"><span class="upcoming-dot" aria-hidden="true"></span><span class="home-upcoming-copy"><strong>{{ item.name }}</strong><small>{{ tr(item.kind==='income'?'homeExpectedIncome':'homeExpectedOutflow') }} / {{ formatDate(item.nextBilling,{dateStyle:'medium',timeZone:item.timezone}) }}<template v-if="item.groupId"> · {{ data.groups.find(g=>g.groupId===item.groupId)?.groupName }}</template></small></span><MoneyValue :amount="item.amountMinor" :currency="item.currency"/></article></div>
        <div v-else class="card home-empty"><EmptyState :title="tr('homeNoUpcoming')" :description="tr('homeNoUpcomingDesc')"/></div>
      </section>

      <section class="home-section" aria-labelledby="home-monthly-title">
        <div class="home-section-heading"><div><p class="eyebrow">{{ data.month }}</p><h2 id="home-monthly-title">{{ tr('homeMonthly') }}</h2></div><RouterLink class="text-link" :to="{name:'dashboard',query:{scope:'personal',month:data.month}}">{{ tr('fullAnalysis') }} →</RouterLink></div>
        <div v-if="data.monthly.length" class="home-monthly-grid">
          <article v-for="item in data.monthly" :key="(item.groupId||'personal')+'-'+item.currency" class="card home-monthly-card">
            <header><strong>{{ item.groupName||tr('personalLedger') }} / {{ item.month }}</strong><span>{{ item.currency }} / {{ tr('homeActualCashflow') }}</span></header>
            <div class="home-monthly-row"><span>{{ tr('homeGeneralReceived') }}</span><MoneyValue :amount="item.incomeReceivedMinor" :currency="item.currency"/></div>
            <div class="home-monthly-row"><span>{{ tr('homeSettlementReceived') }}</span><MoneyValue :amount="item.settlementReceivedMinor" :currency="item.currency"/></div>
            <div class="home-monthly-row total"><strong>{{ tr('homeReceivedTotal') }}</strong><MoneyValue :amount="totals(item).received" :currency="item.currency"/></div>
            <div class="home-monthly-row"><span>{{ tr('homeGeneralPaid') }}</span><MoneyValue :amount="item.expensePaidMinor" :currency="item.currency"/></div>
            <div class="home-monthly-row"><span>{{ tr('homeSettlementPaid') }}</span><MoneyValue :amount="item.settlementPaidMinor" :currency="item.currency"/></div>
            <div class="home-monthly-row total"><strong>{{ tr('homePaidTotal') }}</strong><MoneyValue :amount="totals(item).paid" :currency="item.currency"/></div>
            <div class="home-monthly-divider"></div>
            <div class="home-monthly-row"><span>{{ tr('beneficialIncome') }}</span><MoneyValue :amount="item.beneficialIncomeMinor" :currency="item.currency"/></div>
            <div class="home-monthly-row"><span>{{ tr('personalShare') }}</span><MoneyValue :amount="item.personalExpenseShareMinor" :currency="item.currency"/></div>
            <RouterLink class="home-monthly-analysis" :to="item.groupId?{name:'group-overview',params:{groupId:item.groupId},query:{month:item.month}}:{name:'dashboard',query:{scope:'personal',month:item.month}}">{{ tr('fullAnalysis') }}</RouterLink>
          </article>
        </div>
        <div v-else class="card home-empty"><EmptyState :title="tr('noSummary')" :description="tr('homeMonthlyEmptyDesc')"/></div>
      </section>
    </template>
  </section>
</template>

<style scoped>
.home-page{max-width:1320px}.home-heading{align-items:center}.home-quick-actions{display:flex;flex-wrap:wrap;gap:10px}.home-quick-actions .ghost{display:inline-flex;align-items:center;justify-content:center;min-height:44px;padding-inline:16px;text-decoration:none}.home-error{display:flex;align-items:center;justify-content:space-between;gap:12px;margin:0 0 20px}.home-error button{min-height:44px;padding-inline:14px;border-radius:10px;background:var(--surface);color:var(--danger);font-weight:700}.home-section{display:grid;gap:14px;margin-block:30px}.home-section-heading{display:flex;align-items:end;justify-content:space-between;gap:16px}.home-section-heading .eyebrow{margin-bottom:5px}.home-section-heading h2{margin:0;font-size:clamp(21px,2.2vw,27px);letter-spacing:-.035em}.text-link{min-height:44px;display:inline-flex;align-items:center;color:var(--brand);font-size:14px;font-weight:750;text-decoration:none}.home-balance-columns{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:14px}.home-balance-card{display:grid;gap:18px;min-width:0}.receivable-card{border-color:color-mix(in srgb,var(--success) 38%,var(--line))}.payable-card{border-color:color-mix(in srgb,var(--danger) 32%,var(--line))}.home-card-label{display:flex;align-items:center;gap:11px}.home-card-label h3{margin:0;font-size:16px}.balance-mark{width:36px;height:36px;display:grid;place-items:center;border-radius:12px;font-size:21px;font-weight:800}.received-mark{color:var(--success);background:var(--success-soft)}.paid-mark{color:var(--danger);background:var(--danger-soft)}.home-currency-list{display:grid;gap:10px}.home-currency-list>div,.home-monthly-row{display:flex;align-items:center;justify-content:space-between;gap:12px;min-width:0}.home-currency-list>div{font-size:14px}.home-currency-list :deep(.money-value){font-size:clamp(21px,3vw,30px);font-weight:800}.home-muted{margin:0;color:var(--muted)}.home-group-positions,.home-action-list,.home-upcoming-list{display:grid}.home-group-position{display:grid;grid-template-columns:minmax(0,1fr) auto 20px;align-items:center;gap:12px;padding:14px 0;border-bottom:1px solid var(--line);color:var(--ink);text-decoration:none}.home-group-position:last-child,.home-action-item:last-child,.home-upcoming-item:last-child{border-bottom:0}.home-group-position>span:first-child,.home-action-copy,.home-upcoming-copy{display:grid;gap:4px;min-width:0}.home-group-position strong,.home-action-copy strong,.home-upcoming-copy strong{overflow-wrap:anywhere}.home-group-position small,.home-action-copy small,.home-upcoming-copy small{color:var(--muted);overflow-wrap:anywhere}.home-group-position:hover,.home-notification-action:hover{color:var(--brand)}.row-arrow{justify-self:end;color:var(--brand);font-size:22px}.home-action-item{display:grid;grid-template-columns:40px minmax(0,1fr) auto;align-items:center;gap:13px;padding:15px 0;border-bottom:1px solid var(--line)}.action-icon{width:36px;height:36px;display:grid;place-items:center;border-radius:12px;background:var(--brand-soft);color:var(--brand);font-size:18px;font-weight:800}.home-action-copy>span{font-weight:750}.home-action-controls{display:flex;flex-wrap:wrap;justify-content:end;gap:8px}.home-action-controls button{min-height:44px;padding-inline:14px}.home-reject-form{grid-column:2/-1;display:grid;gap:8px}.home-reject-form label{display:grid;gap:6px;font-size:13px;font-weight:700}.home-reject-form textarea{width:100%;resize:vertical}.home-notification-action{text-decoration:none;color:var(--ink)}.home-range-select{display:flex;align-items:center;gap:8px;color:var(--muted);font-size:13px}.home-range-select select{min-height:44px;max-width:190px}.home-upcoming-item{display:grid;grid-template-columns:10px minmax(0,1fr) auto;align-items:center;gap:13px;padding:14px 0;border-bottom:1px solid var(--line)}.upcoming-dot{width:9px;height:9px;border-radius:50%;background:var(--brand);box-shadow:0 0 0 4px var(--brand-soft)}.home-monthly-grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(min(100%,310px),1fr));gap:14px;align-items:start}.home-monthly-card{display:grid;gap:10px;min-width:0}.home-monthly-card header{display:flex;justify-content:space-between;gap:8px;padding-bottom:10px;border-bottom:1px solid var(--line)}.home-monthly-card header span{color:var(--muted);font-size:13px}.home-monthly-row{font-size:13px}.home-monthly-row>span{color:var(--muted)}.home-monthly-row.total{font-size:14px}.home-monthly-divider{height:1px;margin-block:3px;background:var(--line)}.home-empty{min-height:130px;display:grid;place-items:center}.home-page :deep(.money-value){font-variant-numeric:tabular-nums}@media(max-width:700px){.home-section{margin-block:25px}.home-balance-columns{grid-template-columns:1fr}.home-section-heading{align-items:flex-start;flex-wrap:wrap}.home-action-item{grid-template-columns:36px minmax(0,1fr);align-items:start}.home-action-controls{grid-column:2;justify-content:start}.home-reject-form{grid-column:1/-1}.home-quick-actions{width:100%}.home-quick-actions>*{flex:1}.home-upcoming-item{align-items:start}.home-range-select{width:100%;justify-content:space-between}.home-range-select select{max-width:none;flex:1}}
.home-waiting-controls>span{align-self:center;color:var(--muted);font-size:13px}
.home-monthly-analysis{display:inline-flex;align-items:center;min-height:44px;color:var(--brand);font-weight:750;text-decoration:none}
</style>

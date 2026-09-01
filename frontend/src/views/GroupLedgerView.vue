<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import ConfirmDialog from '../components/ConfirmDialog.vue'
import { useWorkspaceStore } from '../stores/workspace'
import { useI18n } from '../i18n'
import LedgerQuickAddDrawer from '../components/LedgerQuickAddDrawer.vue'
import type { Currency, Expense, Income, LedgerItem, LedgerKind, Subscription } from '../api/types'

const route = useRoute()
const workspace = useWorkspaceStore()
const { tr, formatDate, locale } = useI18n()
const groupId = computed(() => String(route.params.groupId || ''))
const date = ref(new Date().toISOString().slice(0, 10))
const loading = ref(false)
const loadError = ref('')
const quickAdd = ref(false)
const editRecord = ref<Expense | Income | Subscription>()
const editKind = ref<LedgerKind>()
const pendingDelete = ref<LedgerItem>()
const ledger = computed(() => workspace.groupLedger)
const items = computed(() => ledger.value?.items || [])
const group = computed(() => workspace.groups.find(value => value.id === groupId.value))
const currentDate = () => new Date().toISOString().slice(0, 10)

function money(minor: number, currency: Currency) {
  return new Intl.NumberFormat(locale.value, { style: 'currency', currency, maximumFractionDigits: 2 }).format(minor / 100)
}
function statusLabel(status: string) {
  return tr(status === 'recorded' ? 'recorded' : status === 'scheduled' ? 'scheduled' : status === 'failed' ? 'failed' : 'pending')
}
function move(days: number) {
  const next = new Date(date.value + 'T12:00:00')
  next.setDate(next.getDate() + days)
  date.value = next.toISOString().slice(0, 10)
}
function today() { date.value = currentDate() }
async function saved() { quickAdd.value = false; editRecord.value = undefined; editKind.value = undefined; await load() }
function canEdit(item: LedgerItem) {
  return item.kind === 'expense' ? workspace.groupPermissions.includes('ledger.expenses.write') : item.kind === 'income' ? workspace.groupPermissions.includes('ledger.incomes.write') : workspace.groupPermissions.includes('ledger.subscriptions.write')
}
function canDelete(item: LedgerItem) {
  return item.kind === 'expense' ? workspace.groupPermissions.includes('ledger.expenses.delete') : item.kind === 'income' ? workspace.groupPermissions.includes('ledger.incomes.delete') : workspace.groupPermissions.includes('ledger.subscriptions.delete')
}
function resourceFor(item: LedgerItem) {
  if (!item.recordId && !item.subscriptionId) return undefined
  if (item.kind === 'expense') return workspace.expenses.find(value => value.id === item.recordId)
  if (item.kind === 'income') return workspace.groupIncomes.find(value => value.id === item.recordId)
  return workspace.subscriptions.find(value => value.id === (item.subscriptionId || item.recordId))
}
function editItem(item: LedgerItem) {
  const record = resourceFor(item)
  if (!record || !canEdit(item)) return
  editKind.value = item.kind
  editRecord.value = record
}
async function removeItem() {
  const item = pendingDelete.value
  if (!item || !canDelete(item)) return
  if (item.kind === 'expense' && item.recordId) await workspace.deleteExpense(item.recordId)
  else if (item.kind === 'income' && item.recordId) await workspace.deleteGroupIncome(item.recordId)
  else if (item.kind === 'subscription') { const id = item.subscriptionId || item.recordId; if (id) await workspace.deleteSubscription(id) }
  if (!workspace.error) await load()
  pendingDelete.value = undefined
}

async function load() {
  if (!groupId.value) return
  loading.value = true
  loadError.value = ''
  try {
    await workspace.refreshGroupLedger(date.value)
  } catch (error) {
    loadError.value = error instanceof Error ? error.message : tr('requestFailed')
  } finally {
    loading.value = false
  }
}
onMounted(() => void load())
watch([groupId, date], () => { if (groupId.value) void load() })
</script>

<template>
  <section class="page ledger-page">
    <header class="page-heading">
      <div>
        <p class="eyebrow">{{ tr('groupLedger') }}</p>
        <h1>{{ group?.name || tr('groupLedger') }}</h1>
        <p>{{ tr('groupLedgerDesc') }}</p>
      </div>
      <button class="primary" @click="quickAdd=true">+ {{ tr('addRecord') }}</button>
    </header>
    <p v-if="loadError" class="notice danger">{{ loadError }}</p>
    <section class="ledger-datebar card">
      <button class="icon-button" :aria-label="tr('previousDay')" @click="move(-1)">&lsaquo;</button>
      <div class="ledger-date-copy">
        <strong>{{ formatDate(date + 'T12:00:00') }}</strong>
        <span>{{ ledger?.timezone || group?.timezone || 'UTC' }} &middot; {{ ledger?.date || date }}</span>
      </div>
      <button class="icon-button" :aria-label="tr('nextDay')" @click="move(1)">&rsaquo;</button>
      <button v-if="date !== currentDate()" class="ghost today-button" @click="today">{{ tr('backToToday') }}</button>
    </section>
    <p v-if="loading" class="empty-inline">{{ tr('processing') }}</p>
    <section v-if="ledger" class="ledger-summary-grid">
      <article v-for="summary in ledger.summaries" :key="summary.currency" class="ledger-summary card">
        <div class="summary-top"><span>{{ summary.currency }}</span><span>{{ tr('ledgerRecords', { count: summary.count }) }}</span></div>
        <strong>{{ money(summary.netMinor, summary.currency) }}</strong>
        <div class="summary-lines">
          <span class="income">{{ tr('income') }} {{ money(summary.incomeMinor, summary.currency) }}</span>
          <span class="expense">{{ tr('expense') }} {{ money(summary.expenseMinor, summary.currency) }}</span>
          <span class="subscription">{{ tr('subscriptions') }} {{ money(summary.subscriptionMinor, summary.currency) }}</span>
        </div>
      </article>
      <article v-if="!ledger.summaries.length" class="ledger-summary card empty-summary">
        <strong>{{ tr('ledgerEmpty') }}</strong><span>{{ tr('ledgerEmptyDesc') }}</span>
      </article>
    </section>
    <section class="ledger-list card">
      <div class="section-heading">
        <div><p class="eyebrow">{{ tr('timeline') }}</p><h2>{{ tr('ledger') }}</h2></div>
        <span>{{ tr('ledgerRecords', { count: items.length }) }}</span>
      </div>
      <div v-if="!items.length" class="ledger-empty">
        <div class="empty-icon">*</div><strong>{{ tr('ledgerEmpty') }}</strong><p>{{ tr('ledgerEmptyDesc') }}</p>
      </div>
      <article v-for="item in items" :key="item.id" class="ledger-item">
        <div class="item-icon">{{ item.kind === 'income' ? '+' : item.kind === 'subscription' ? '~' : '-' }}</div>
        <div class="item-main">
          <strong>{{ item.title }}</strong>
          <span>{{ item.category || tr('uncategorized') }} &middot; {{ statusLabel(item.status) }}</span>
          <small v-if="item.notes">{{ item.notes }}</small>
        </div>
        <div class="item-side">
          <strong :class="item.kind === 'income' ? 'income' : 'expense'">{{ item.kind === 'income' ? '+' : '-' }}{{ money(item.amountMinor, item.currency) }}</strong>
          <time>{{ new Date(item.occurredAt).toLocaleTimeString(locale, { hour: '2-digit', minute: '2-digit' }) }}</time><span class="row-actions"><button v-if="canEdit(item)" class="text-button" :aria-label="tr('edit')" @click="editItem(item)">{{ tr('edit') }}</button><button v-if="canDelete(item)" class="text-button danger-text" :aria-label="tr('remove')" @click="pendingDelete=item">{{ tr('remove') }}</button></span>
        </div>
      </article>
    </section>
        <LedgerQuickAddDrawer :open="quickAdd" :group-id="groupId" :date="date" @close="quickAdd=false" @saved="saved" /><LedgerQuickAddDrawer :open="Boolean(editRecord)" :group-id="groupId" :date="date" :edit-record="editRecord" :edit-kind="editKind" @close="editRecord=undefined;editKind=undefined" @saved="saved" /><ConfirmDialog :open="Boolean(pendingDelete)" :title="pendingDelete ? tr(pendingDelete.kind === 'income' ? 'deleteIncomeConfirm' : pendingDelete.kind === 'subscription' ? 'deleteSubscriptionConfirm' : 'removeExpenseConfirm', { name: pendingDelete.title }) : ''" danger @cancel="pendingDelete=undefined" @confirm="removeItem" />
    <button class="ledger-fab" :aria-label="tr('addRecord')" @click="quickAdd=true">+</button>
  </section>
</template>

<style scoped>
.ledger-page{max-width:1120px;margin:auto}.ledger-datebar{display:flex;align-items:center;gap:1rem;padding:1rem 1.25rem;margin-bottom:1rem}.ledger-date-copy{display:flex;flex:1;flex-direction:column;text-align:center}.ledger-date-copy strong{font-size:1.2rem}.ledger-date-copy span,.summary-top,.item-main span,.item-main small,.item-side time{color:var(--muted);font-size:.82rem}.ledger-summary-grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(220px,1fr));gap:1rem;margin-bottom:1rem}.ledger-summary{padding:1.2rem}.summary-top{display:flex;justify-content:space-between}.ledger-summary>strong{display:block;font-size:1.65rem;margin:.65rem 0}.summary-lines{display:grid;gap:.25rem;font-size:.8rem}.income{color:#72d6ad}.expense{color:#ff9b9b}.subscription{color:#aaa4ff}.ledger-list{padding:1.25rem}.section-heading{display:flex;justify-content:space-between;align-items:center}.section-heading h2{margin:.2rem 0 1rem}.section-heading>span{color:var(--muted)}.ledger-item{display:flex;align-items:center;gap:.8rem;padding:1rem 0;border-top:1px solid var(--line)}.item-icon{display:grid;place-items:center;width:2.5rem;height:2.5rem;border-radius:50%;background:var(--surface-soft);font-size:1.3rem}.item-main{display:grid;gap:.2rem;flex:1;min-width:0}.item-main strong{overflow:hidden;text-overflow:ellipsis}.item-main small{white-space:nowrap;overflow:hidden;text-overflow:ellipsis}.item-side{text-align:right;display:grid;gap:.2rem;justify-items:end}.item-side strong{white-space:nowrap}.ledger-empty,.empty-summary{text-align:center;padding:2.5rem;color:var(--muted)}.empty-summary{display:flex;flex-direction:column;gap:.4rem}.empty-summary strong,.ledger-empty strong{color:var(--ink)}.empty-icon{font-size:2rem}.ledger-fab{display:none}
@media(max-width:900px){.ledger-page{padding-bottom:calc(7rem + env(safe-area-inset-bottom))}.ledger-fab{display:grid;place-items:center;position:fixed;right:1.1rem;bottom:calc(88px + env(safe-area-inset-bottom));z-index:101;width:3.5rem;height:3.5rem;border-radius:50%;background:var(--brand);color:#fff;text-decoration:none;font-size:1.7rem;box-shadow:0 10px 30px #0008}.ledger-summary-grid{grid-template-columns:1fr 1fr;gap:.65rem}.ledger-summary{padding:.85rem}.ledger-summary>strong{font-size:1.15rem}.summary-lines{font-size:.7rem}}@media(max-width:420px){.ledger-summary-grid{grid-template-columns:1fr}}
.ledger-page{padding-bottom:2rem}.icon-button{border:1px solid transparent;border-radius:10px;background:transparent;color:inherit;font-size:1.8rem;padding:.25rem .6rem}.icon-button:hover{border-color:var(--line);background:var(--surface-soft)}.item-icon{border:1px solid var(--line);background:var(--surface-soft)}.ledger-datebar{border-color:var(--line-strong);box-shadow:var(--shadow)}.ledger-summary{border-color:var(--line);box-shadow:var(--shadow)}@media(max-width:900px){.ledger-page{padding:1rem 1rem calc(7rem + env(safe-area-inset-bottom))}.ledger-datebar{padding:.8rem}.ledger-fab{border:1px solid color-mix(in srgb,var(--brand) 70%,transparent);box-shadow:var(--shadow-lg)}}</style>

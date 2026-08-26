<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import PersonalLedgerNav from '../components/PersonalLedgerNav.vue'
import MoneyValue from '../components/MoneyValue.vue'
import EmptyState from '../components/EmptyState.vue'
import AppDrawer from '../components/AppDrawer.vue'
import SyncBadge from '../components/SyncBadge.vue'
import { useWorkspaceStore } from '../stores/workspace'
import { useI18n } from '../i18n'
import type { Currency, Income } from '../api/types'

const workspace = useWorkspaceStore()
const { tr, formatDate } = useI18n()
const open = ref(false)
const saving = ref(false)
const editing = ref<Income | null>(null)
const form = reactive({ title: '', amount: '', currency: 'TWD' as Currency, category: '', date: '', notes: '' })
const list = computed(() => [...workspace.personalIncomes].sort((a, b) => b.receivedOn.localeCompare(a.receivedOn)))

function dateValue(value = new Date()) {
  return value.toISOString().slice(0, 10)
}
function resetForm() {
  form.title = ''
  form.amount = ''
  form.currency = (workspace.currencies[0]?.code || 'TWD') as Currency
  form.category = ''
  form.date = dateValue()
  form.notes = ''
}
function create() {
  editing.value = null
  resetForm()
  open.value = true
}
function edit(item: Income) {
  editing.value = item
  form.title = item.title
  form.amount = String(item.amountMinor / 100)
  form.currency = item.currency
  form.category = item.category
  form.date = item.receivedOn.slice(0, 10)
  form.notes = item.notes
  open.value = true
}
async function submit() {
  if (!form.title.trim() || !Number.isFinite(Number(form.amount)) || Number(form.amount) <= 0 || !form.date) return
  saving.value = true
  const input = {
    title: form.title.trim(),
    amountMinor: Math.round(Number(form.amount) * 100),
    currency: form.currency,
    category: form.category,
    receivedOn: form.date + 'T12:00:00.000Z',
    notes: form.notes,
  }
  try {
    const ok = editing.value ? await workspace.updateIncome(editing.value.id, input) : await workspace.addIncome(input)
    if (ok) open.value = false
  } finally {
    saving.value = false
  }
}
async function remove(item: Income) {
  if (window.confirm(tr('deleteIncomeConfirm', { name: item.title }))) await workspace.deleteIncome(item.id)
}
onMounted(() => { void workspace.refreshPersonal() })
</script>

<template>
<section class="page ledger-page income-management-page">
  <PersonalLedgerNav />
  <div class="page-heading">
    <div><p class="eyebrow">{{ tr('income') }}</p><h1>{{ tr('income') }}</h1><p>{{ tr('incomeManagementDesc') }}</p></div>
    <button class="primary" @click="create">{{ tr('createIncome') }}</button>
  </div>
  <p v-if="workspace.error" class="inline-error">{{ workspace.localizedError }}</p>
  <section class="card data-card">
    <div class="card-title"><h2>{{ tr('recentIncomes') }}</h2><span>{{ tr('records', { count: list.length }) }}</span></div>
    <div v-if="list.length" class="data-table income-table">
      <div class="data-table-head"><span>{{ tr('incomeTitle') }}</span><span>{{ tr('category') }}</span><span>{{ tr('date') }}</span><span>{{ tr('amount') }}</span><span></span></div>
      <article v-for="item in list" :key="item.id" class="data-table-row">
        <div class="item-cell"><span class="service-icon income">+</span><span><strong>{{ item.title }}</strong><small>{{ item.category || tr('uncategorized') }}</small><SyncBadge :pending-sync="item.pendingSync" :sync-error="item.syncError" /></span></div>
        <span>{{ item.category || tr('uncategorized') }}</span>
        <span class="timezone-date"><strong>{{ formatDate(item.receivedOn) }}</strong></span>
        <span class="money-stack income"><MoneyValue :amount="item.amountMinor" :currency="item.currency" /></span>
        <span class="row-actions"><button class="icon-button" :aria-label="tr('editIncome')" @click="edit(item)">&#9998;</button><button class="icon-button" :aria-label="tr('deleteIncome')" @click="remove(item)">&times;</button></span>
      </article>
    </div>
    <EmptyState v-else :title="tr('noIncomes')" :description="tr('noIncomesDesc')" />
  </section>
  <AppDrawer :open="open" :title="tr(editing ? 'editIncome' : 'createIncome')" @close="open = false">
    <form class="form-card income-form" @submit.prevent="submit">
      <label>{{ tr('incomeTitle') }}<input v-model="form.title" required /></label>
      <div class="form-row"><label>{{ tr('amount') }}<input v-model="form.amount" type="number" min="0.01" step="0.01" required /></label><label>{{ tr('currency') }}<select v-model="form.currency"><option>TWD</option><option>USD</option><option>JPY</option><option>EUR</option></select></label></div>
      <label>{{ tr('category') }}<input v-model="form.category" :placeholder="tr('uncategorized')" /></label>
      <label>{{ tr('date') }}<input v-model="form.date" type="date" required /></label>
      <label>{{ tr('notes') }}<textarea v-model="form.notes" rows="3"></textarea></label>
      <button class="primary full-width" :disabled="saving">{{ saving ? tr('processing') : tr('saveRecord') }}</button>
    </form>
  </AppDrawer>
</section>
</template>

<style scoped>
.income-management-page{max-width:1200px;margin:auto}.income-table .income{color:#72d6ad}.income-form{display:grid;gap:1rem}.income-form label{display:grid;gap:.4rem;color:var(--muted)}.income-form input,.income-form select,.income-form textarea{box-sizing:border-box;width:100%;border:1px solid var(--border);border-radius:10px;background:var(--surface-strong);color:var(--text);padding:.75rem;font:inherit}.full-width{width:100%}.inline-error{color:#ff9b9b;margin:.75rem 0}
@media(max-width:900px){.income-management-page{padding-bottom:calc(7rem + env(safe-area-inset-bottom))}.income-management-page .page-heading{align-items:flex-start}.income-management-page .page-heading .primary{white-space:nowrap}.income-table .data-table-head{display:none}.income-table .data-table-row{grid-template-columns:1fr auto;gap:.5rem}.income-table .data-table-row>span:nth-child(2),.income-table .data-table-row>span:nth-child(3){display:none}.income-table .row-actions{grid-column:2;grid-row:1}.income-table .money-stack{grid-column:1;grid-row:2;justify-self:start}.income-table .item-cell{grid-column:1;grid-row:1}}
</style>
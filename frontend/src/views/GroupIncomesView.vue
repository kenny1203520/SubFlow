<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useRoute } from 'vue-router'
import { useWorkspaceStore } from '../stores/workspace'
import { useAuthStore } from '../stores/auth'
import { useI18n } from '../i18n'
import type { Currency, ExpenseSplit, Income, SplitMode } from '../api/types'

const route = useRoute()
const workspace = useWorkspaceStore()
const auth = useAuthStore()
const { tr, formatDate, locale } = useI18n()
const groupId = computed(() => String(route.params.groupId || ''))
const group = computed(() => workspace.groups.find(value => value.id === groupId.value))
const incomes = computed(() => workspace.groupIncomes)
const editingId = ref('')
const saving = ref(false)
const error = ref('')
const form = reactive({ title: '', amount: '', currency: 'TWD' as Currency, category: '', date: new Date().toISOString().slice(0, 10), notes: '', paidBy: '', splitMode: 'equal' as SplitMode, splits: {} as Record<string, string> })
const memberOptions = computed(() => workspace.members.filter(member => !member.user?.placeholder))

function nameOf(id: string) { return memberOptions.value.find(member => member.userId === id)?.user?.name || id }
function money(minor: number, currency: Currency) {
  return new Intl.NumberFormat(locale.value, { style: 'currency', currency, maximumFractionDigits: 2 }).format(minor / 100)
}
function reset() {
  editingId.value = ''
  form.title = ''
  form.amount = ''
  form.category = ''
  form.currency = (group.value?.currency || auth.record?.defaultCurrency || 'TWD') as Currency
  form.date = new Date().toISOString().slice(0, 10)
  form.notes = ''
  form.paidBy = auth.record?.id || memberOptions.value[0]?.userId || ''
  form.splitMode = 'equal'
  form.splits = {}
}
function edit(item: Income) {
  editingId.value = item.id
  form.title = item.title
  form.amount = String(item.amountMinor / 100)
  form.currency = item.currency
  form.category = item.category
  form.date = item.receivedOn.slice(0, 10)
  form.notes = item.notes
  form.paidBy = item.paidBy || auth.record?.id || ''
  form.splitMode = item.splitMode || 'equal'
  form.splits = {}
  for (const split of item.splits || []) form.splits[split.userId] = form.splitMode === 'percentage' ? String((split.percentageBasisPoints || 0) / 100) : String((split.amountMinor || 0) / 100)
}
function splitPayload(_amount: number): ExpenseSplit[] {
  const ids = memberOptions.value.map(member => member.userId)
  const result: ExpenseSplit[] = []
  for (const userId of ids) {
    const raw = Number(form.splits[userId] || 0)
    if (form.splitMode === 'equal') result.push({ userId, amountMinor: 0 })
    else if (form.splitMode === 'percentage') result.push({ userId, amountMinor: 0, percentageBasisPoints: Math.round(raw * 100) })
    else result.push({ userId, amountMinor: Math.round(raw * 100) })
  }
  return result.filter(split => split.amountMinor !== 0 || (split.percentageBasisPoints || 0) !== 0)
}
function validate(amount: number) {
  if (!form.title.trim() || !Number.isFinite(amount) || amount <= 0 || !form.paidBy) return tr('invalidRequest')
  const values = splitPayload(amount)
  if (form.splitMode === 'amount' && values.reduce((sum, value) => sum + value.amountMinor, 0) !== amount) return tr('splitInvalidAmount')
  if (form.splitMode === 'percentage' && values.reduce((sum, value) => sum + (value.percentageBasisPoints || 0), 0) !== 10000) return tr('splitInvalidPercentage')
  return ''
}
async function submit() {
  const amount = Math.round(Number(form.amount) * 100)
  const validation = validate(amount)
  if (validation) { error.value = validation; return }
  saving.value = true
  error.value = ''
  const input = { title: form.title.trim(), amountMinor: amount, currency: form.currency as Currency, category: form.category, paidBy: form.paidBy, splitMode: form.splitMode, splits: splitPayload(amount), receivedOn: form.date + 'T12:00:00.000Z', notes: form.notes }
  try {
    if (editingId.value) await workspace.updateGroupIncome(editingId.value, input)
    else await workspace.addGroupIncome(input)
    if (!workspace.error) { const selected = form.date; reset(); await workspace.refreshGroupLedger(selected) }
  } finally { saving.value = false }
}
async function remove(item: Income) {
  if (typeof window !== 'undefined' && !window.confirm(tr('deleteIncomeConfirm', { name: item.title }))) return
  await workspace.deleteGroupIncome(item.id)
}
onMounted(() => reset())
</script>

<template>
  <section class="page group-income-page">
    <header class="page-heading">
      <div><p class="eyebrow">{{ tr('groupIncomes') }}</p><h1>{{ group?.name || tr('groupIncomes') }}</h1><p>{{ tr('groupIncomeDesc') }}</p></div>
      <RouterLink class="ghost" :to="'/groups/' + groupId + '/ledger'">{{ tr('groupLedger') }}</RouterLink>
    </header>
    <p v-if="error || workspace.groupErrors.incomes" class="notice danger">{{ error || workspace.groupErrors.incomes }}</p>
    <div class="two-column content-heavy">
      <section class="card form-card">
        <div class="card-title"><h2>{{ editingId ? tr('editIncome') : tr('addGroupIncome') }}</h2><button v-if="editingId" class="ghost" @click="reset">{{ tr('cancel') }}</button></div>
        <label>{{ tr('incomeTitle') }}<input v-model="form.title" required :placeholder="tr('itemPlaceholder')"></label>
        <div class="form-row"><label>{{ tr('amount') }}<input v-model="form.amount" type="number" min="0" step="0.01" required></label><label>{{ tr('currency') }}<select v-model="form.currency"><option>{{ group?.currency || 'TWD' }}</option><option>USD</option><option>JPY</option><option>EUR</option></select></label></div>
        <div class="form-row"><label>{{ tr('category') }}<input v-model="form.category"></label><label>{{ tr('receivedBy') }}<select v-model="form.paidBy"><option v-for="member in memberOptions" :key="member.userId" :value="member.userId">{{ member.user?.name || member.userId }}</option></select></label></div>
        <label>{{ tr('splitMode') }}<select v-model="form.splitMode"><option value="equal">{{ tr('splitEqual') }}</option><option value="amount">{{ tr('splitAmount') }}</option><option value="percentage">{{ tr('splitPercentage') }}</option></select></label>
        <div v-if="form.splitMode !== 'equal'" class="split-editor"><label v-for="member in memberOptions" :key="member.userId">{{ member.user?.name || member.userId }}<input v-model="form.splits[member.userId]" type="number" min="0" step="0.01" placeholder="0"><small v-if="form.splitMode === 'percentage'">{{ tr('basisPointsHint') }}</small></label></div>
        <label>{{ tr('date') }}<input v-model="form.date" type="date" required></label>
        <label>{{ tr('notes') }}<textarea v-model="form.notes" rows="3"></textarea></label>
        <button class="primary wide" :disabled="saving" @click="submit">{{ saving ? tr('processing') : tr('save') }}</button>
      </section>
      <section class="card">
        <div class="card-title"><h2>{{ tr('groupIncomes') }}</h2><span>{{ tr('records', { count: incomes.length }) }}</span></div>
        <div v-if="!incomes.length" class="empty"><h3>{{ tr('noGroupIncomes') }}</h3><p>{{ tr('groupIncomeDesc') }}</p></div>
        <div class="rows">
          <article v-for="item in incomes" :key="item.id" class="row">
            <div class="service-icon">+</div>
            <div class="grow"><strong>{{ item.title }}</strong><small>{{ formatDate(item.receivedOn) }} &middot; {{ nameOf(item.paidBy || '') }}</small></div>
            <div class="money"><strong class="income">+{{ money(item.amountMinor, item.currency) }}</strong><small>{{ item.category || tr('uncategorized') }}</small></div>
            <div class="actions"><button class="ghost" @click="edit(item)">{{ tr('edit') }}</button><button class="ghost danger-text" @click="remove(item)">{{ tr('deleteIncome') }}</button></div>
          </article>
        </div>
      </section>
    </div>
  </section>
</template>

<style scoped>
.group-income-page{max-width:1180px;margin:auto}.income{color:#72d6ad}.split-editor{display:grid;gap:10px;padding:12px;border:1px solid var(--line);border-radius:12px;background:var(--surface-soft)}.split-editor label{font-size:12px}.split-editor small{color:var(--muted);font-weight:500}.row .actions{margin-left:auto}.row .actions .ghost{padding:5px 7px}@media(max-width:900px){.group-income-page{padding-bottom:calc(7rem + env(safe-area-inset-bottom))}.row .actions{width:100%;margin-left:0}.row .actions .ghost{flex:1}}
</style>

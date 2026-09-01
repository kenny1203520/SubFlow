<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import AppDrawer from './AppDrawer.vue'
import { useAuthStore } from '../stores/auth'
import { useWorkspaceStore } from '../stores/workspace'
import type { Currency, ExpenseSplit, IncomeSplit, SplitMode } from '../api/types'
import { majorToMinor } from '../api/money'
import { fromDateInput } from '../dateInput'
import { useI18n } from '../i18n'

const props = defineProps<{ open: boolean; groupId: string; date: string }>()
const emit = defineEmits<{ close: []; saved: [] }>()

const workspace = useWorkspaceStore()
const auth = useAuthStore()
const { tr } = useI18n()
const mode = ref<'expense' | 'income'>('expense')
const saving = ref(false)
const error = ref('')
const group = computed(() => workspace.groups.find(value => value.id === props.groupId))
const members = computed(() => workspace.members.filter(member => !(member.user?.placeholder && member.user?.linkedUserId)))
const canExpense = computed(() => workspace.groupPermissions.includes('ledger.expenses.write'))
const canIncome = computed(() => workspace.groupPermissions.includes('ledger.incomes.write'))
const form = reactive({
  title: '',
  amount: '',
  currency: 'TWD' as Currency,
  category: '',
  date: '',
  notes: '',
  payer: '',
  rateMode: 'automatic' as 'automatic' | 'manual',
  exchangeRate: '',
  splitMode: 'equal' as SplitMode,
  participants: {} as Record<string, boolean>,
  values: {} as Record<string, string>,
})

const selectedMembers = computed(() => members.value.filter(member => form.participants[member.userId]))
const amountMinor = computed(() => majorToMinor(form.amount || '0', form.currency))
const splitValid = computed(() => {
  if (!selectedMembers.value.length) return false
  if (form.splitMode === 'equal') return true
  const total = selectedMembers.value.reduce((sum, member) => sum + (form.splitMode === 'amount'
    ? majorToMinor(form.values[member.userId] || '0', form.currency)
    : Math.round(Number(form.values[member.userId] || 0) * 100)), 0)
  return form.splitMode === 'amount' ? total === amountMinor.value : total === 10000
})

function reset() {
  error.value = ''
  form.title = ''
  form.amount = ''
  form.currency = group.value?.currency || auth.record?.defaultCurrency || 'TWD'
  form.category = ''
  form.date = props.date
  form.notes = ''
  form.payer = auth.record?.id || members.value[0]?.userId || ''
  form.rateMode = 'automatic'
  form.exchangeRate = ''
  form.splitMode = 'equal'
  form.participants = Object.fromEntries(members.value.map(member => [member.userId, true]))
  form.values = {}
}

watch(() => props.open, async open => {
  if (!open) return
  if (workspace.currentGroupId !== props.groupId) await workspace.selectGroup(props.groupId)
  await workspace.loadCategories('group', props.groupId)
  if (!canExpense.value && canIncome.value) mode.value = 'income'
  reset()
})

function expenseSplits(): ExpenseSplit[] {
  return selectedMembers.value.map(member => ({
    userId: member.userId,
    amountMinor: form.splitMode === 'amount' ? majorToMinor(form.values[member.userId] || '0', form.currency) : 0,
    percentageBasisPoints: form.splitMode === 'percentage' ? Math.round(Number(form.values[member.userId] || 0) * 100) : undefined,
  }))
}

function incomeSplits(): IncomeSplit[] {
  return selectedMembers.value.map(member => ({
    userId: member.userId,
    amountMinor: form.splitMode === 'amount' ? majorToMinor(form.values[member.userId] || '0', form.currency) : 0,
    percentageBasisPoints: form.splitMode === 'percentage' ? Math.round(Number(form.values[member.userId] || 0) * 100) : undefined,
  }))
}

async function submit() {
  if (!form.title.trim() || amountMinor.value <= 0 || !form.payer || !splitValid.value) {
    error.value = tr('invalidRequest')
    return
  }
  saving.value = true
  error.value = ''
  const occurred = fromDateInput(form.date, group.value?.timezone || 'UTC')
  const common = {
    title: form.title.trim(),
    category: form.category,
    amountMinor: amountMinor.value,
    currency: form.currency,
    rateMode: form.rateMode,
    exchangeRate: form.exchangeRate,
    notes: form.notes,
    splitMode: form.splitMode,
  }
  const ok = mode.value === 'expense'
    ? await workspace.addExpense({ ...common, paidBy: form.payer, incurredOn: occurred, splits: expenseSplits() })
    : await workspace.addGroupIncome({ ...common, earnedBy: form.payer, receivedOn: occurred, splits: incomeSplits() })
  saving.value = false
  if (!ok) {
    error.value = workspace.localizedError || tr('requestFailed')
    return
  }
  emit('saved')
  emit('close')
}
</script>

<template>
  <AppDrawer :open="open" :title="tr('addRecord')" @close="emit('close')">
    <form class="ledger-form" @submit.prevent="submit">
      <div v-if="error" class="notice danger inline">{{ error }}</div>
      <div class="mode-tabs">
        <button type="button" :class="{active:mode==='expense'}" :disabled="!canExpense" @click="mode='expense'">{{ tr('expense') }}</button>
        <button type="button" :class="{active:mode==='income'}" :disabled="!canIncome" @click="mode='income'">{{ tr('income') }}</button>
      </div>
      <label>{{ tr('item') }}<input v-model="form.title" required :placeholder="tr('itemPlaceholder')"></label>
      <div class="form-row">
        <label>{{ tr('amount') }}<input v-model="form.amount" type="number" min="0" step="0.01" required></label>
        <label>{{ tr('currency') }}<select v-model="form.currency"><option>TWD</option><option>USD</option><option>JPY</option><option>EUR</option></select></label>
      </div>
      <div class="form-row">
        <label>{{ tr('category') }}<input v-model="form.category"></label>
        <label>{{ mode==='expense' ? tr('payer') : tr('receivedBy') }}<select v-model="form.payer"><option v-for="member in members" :key="member.userId" :value="member.userId">{{ member.user?.name || member.user?.email || member.userId }}</option></select></label>
      </div>
      <div class="form-row">
        <label>{{ tr('exchangeRate') }}<select v-model="form.rateMode"><option value="automatic">{{ tr('automaticRate') }}</option><option value="manual">{{ tr('manualRate') }}</option></select></label>
        <label v-if="form.rateMode==='manual'">{{ tr('manualRate') }}<input v-model="form.exchangeRate" inputmode="decimal" required></label>
      </div>
      <label>{{ tr('splitMode') }}<select v-model="form.splitMode"><option value="equal">{{ tr('splitEqual') }}</option><option value="amount">{{ tr('splitAmount') }}</option><option value="percentage">{{ tr('splitPercentage') }}</option></select></label>
      <fieldset class="split-editor">
        <legend>{{ tr('participants') }}</legend>
        <label v-for="member in members" :key="member.userId" class="split-member"><input v-model="form.participants[member.userId]" type="checkbox"><span>{{ member.user?.name || member.user?.email || member.userId }}</span><input v-if="form.participants[member.userId] && form.splitMode !== 'equal'" v-model="form.values[member.userId]" type="number" min="0" :step="form.splitMode==='percentage' ? '0.01' : '0.01'"></label>
        <p :class="splitValid ? 'success' : 'form-error'">{{ splitValid ? tr('splitValid') : tr(form.splitMode==='percentage' ? 'splitInvalidPercentage' : 'splitInvalidAmount') }}</p>
      </fieldset>
      <label>{{ tr('date') }}<input v-model="form.date" type="date" required></label>
      <label>{{ tr('notes') }}<textarea v-model="form.notes" rows="3"></textarea></label>
      <div class="form-actions"><button type="button" class="ghost" @click="emit('close')">{{ tr('cancel') }}</button><button class="primary" :disabled="saving || !splitValid">{{ saving ? tr('processing') : tr('saveRecord') }}</button></div>
    </form>
  </AppDrawer>
</template>

<style scoped>
.mode-tabs{display:grid;grid-template-columns:repeat(2,1fr);gap:.4rem;background:var(--surface-strong);padding:.3rem;border-radius:12px}.mode-tabs button{border:0;background:none;color:var(--muted);padding:.65rem;border-radius:9px;cursor:pointer}.mode-tabs button.active{background:var(--accent);color:white}.ledger-form{display:grid;gap:.8rem}.ledger-form label{display:grid;gap:.35rem;color:var(--muted);font-size:.85rem}.ledger-form input,.ledger-form select,.ledger-form textarea{width:100%;box-sizing:border-box;border:1px solid var(--border);border-radius:10px;background:var(--surface-strong);color:var(--text);padding:.75rem;font:inherit}.form-row{display:grid;grid-template-columns:1fr 1fr;gap:.7rem}.split-editor{display:grid;gap:.55rem;border:1px solid var(--border);border-radius:10px;padding:.75rem}.split-member{display:grid!important;grid-template-columns:auto 1fr minmax(0,110px);align-items:center}.form-actions{display:flex;justify-content:flex-end;gap:.7rem}.success{color:#72d6ad}.form-error{color:#ff9b9b}
</style>
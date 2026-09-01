<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import AppDrawer from './AppDrawer.vue'
import BaseInput from './BaseInput.vue'
import BaseCombobox from './BaseCombobox.vue'
import CategorySelect from './CategorySelect.vue'
import CurrencySelect from './CurrencySelect.vue'
import PayerSelect from './PayerSelect.vue'
import ConversionPreview from './ConversionPreview.vue'
import { useAuthStore } from '../stores/auth'
import { useWorkspaceStore } from '../stores/workspace'
import type { BillingCycle, Currency, Expense, ExpenseSplit, Income, IncomeSplit, LedgerKind, SplitMode, Subscription, SubscriptionStatus } from '../api/types'
import { amountStep, majorToMinor, minorToInput } from '../api/money'

import { fromDateInput, fromDateTimeInput } from '../dateInput'
import { useI18n } from '../i18n'

const props = defineProps<{ open: boolean; groupId?: string; date: string; personal?: boolean; editRecord?: Expense | Income | Subscription; editKind?: LedgerKind }>()
const emit = defineEmits<{ close: []; saved: [] }>()

const workspace = useWorkspaceStore()
const auth = useAuthStore()
const { tr } = useI18n()
const mode = ref<'expense' | 'income' | 'subscription'>('expense')
const saving = ref(false)
const error = ref('')
const personal = computed(() => props.personal === true); const group = computed(() => workspace.groups.find(value => value.id === props.groupId))




const members = computed(() => personal.value ? [] : workspace.members.filter(member => !(member.user?.placeholder && member.user?.linkedUserId)))
const canExpense = computed(() => personal.value || workspace.groupPermissions.includes('ledger.expenses.write'))
const canIncome = computed(() => personal.value || workspace.groupPermissions.includes('ledger.incomes.write'))
const canSubscription = computed(() => personal.value || workspace.groupPermissions.includes('ledger.subscriptions.write')); const editing = computed(() => Boolean(props.editRecord))
const form = reactive({
  title: '',
  amount: '',
  currency: 'TWD' as Currency,
  category: '',
  categoryId: '',
  date: '',
  notes: '',
  payer: '',
  rateMode: 'automatic' as 'automatic' | 'manual',
  exchangeRate: '',
  splitMode: 'equal' as SplitMode,
  participants: {} as Record<string, boolean>,
  values: {} as Record<string, string>,
  billingCycle: 'monthly' as BillingCycle, billingInterval: '1', status: 'active' as SubscriptionStatus, revisionScope: 'future' as 'future' | 'one_off', effectiveBillingAt: '', endBillingAt: '',
})
const reportingCurrency = computed(() => personal.value ? (auth.record?.defaultCurrency || form.currency) : (group.value?.currency || form.currency))
const rateValid = ref(true)
const rateOptions = computed(() => [{ value: 'automatic', label: tr('automaticRate') }, { value: 'manual', label: tr('manualRate') }])
const splitOptions = computed(() => [{ value: 'equal', label: tr('splitEqual') }, { value: 'amount', label: tr('splitAmount') }, { value: 'percentage', label: tr('splitPercentage') }])
const cycleOptions = computed(() => [{ value: 'daily', label: tr('daily') }, { value: 'every_n_days', label: tr('everyNDays') }, { value: 'weekly', label: tr('weekly') }, { value: 'every_n_weeks', label: tr('everyNWeeks') }, { value: 'every_n_hours', label: tr('everyNHours') }, { value: 'monthly', label: tr('monthly') }, { value: 'quarterly', label: tr('quarterly') }, { value: 'yearly', label: tr('yearly') }])

const selectedMembers = computed(() => members.value.filter(member => form.participants[member.userId]))
const amountMinor = computed(() => majorToMinor(form.amount || '0', form.currency))
const splitValid = computed(() => {
  if (personal.value) return true; if (!selectedMembers.value.length) return false
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
  form.currency = personal.value ? (auth.record?.defaultCurrency || 'TWD') : (group.value?.currency || auth.record?.defaultCurrency || 'TWD')
  form.category = ''
  form.categoryId = ''
  form.date = props.date
  form.notes = ''
  form.payer = auth.record?.id || members.value[0]?.userId || ''
  form.rateMode = 'automatic'
  rateValid.value = true
  form.exchangeRate = ''
  form.splitMode = 'equal'; form.billingCycle = 'monthly'; form.billingInterval = '1'; form.status = 'active'; form.revisionScope = 'future'; form.effectiveBillingAt = ''; form.endBillingAt = '';
  form.participants = Object.fromEntries(members.value.map(member => [member.userId, true]))
  form.values = {}
}

watch(() => props.open, async open => {
  if (!open) return
  if (!personal.value) { if (workspace.currentGroupId !== props.groupId && props.groupId) await workspace.selectGroup(props.groupId); await workspace.loadCategories('group', props.groupId) } else await workspace.loadCategories('personal')
  if (props.editRecord) loadRecord(props.editRecord)
  else {
    if (!canExpense.value && canIncome.value) mode.value = 'income'
    reset()
  }
})

async function addCategory(name: string, icon = 'tag') { try { const value = await workspace.createCategory(personal.value ? 'personal' : 'group', name, personal.value ? '' : props.groupId, icon); form.categoryId = value.id } catch { error.value = workspace.localizedError || tr('requestFailed') } }

function loadRecord(record: Expense | Income | Subscription) {
  reset()
  if (props.editKind === 'income' || ('receivedOn' in record)) {
    const value = record as Income
    mode.value = 'income'
    Object.assign(form, { title: value.title, amount: minorToInput(value.amountMinor, value.currency), currency: value.currency, category: value.category, categoryId: value.categoryId || '', date: value.receivedOn.slice(0, 10), notes: value.notes, payer: value.earnedBy || auth.record?.id || '', rateMode: value.rateMode || 'automatic', exchangeRate: value.exchangeRate || '', splitMode: value.splitMode || 'equal' })
    form.participants = Object.fromEntries((value.splits || members.value.map(member => ({ userId: member.userId }))).map(split => [split.userId, true]))
    form.values = Object.fromEntries((value.splits || []).map(split => [split.userId, value.splitMode === 'percentage' ? String((split.percentageBasisPoints || 0) / 100) : minorToInput(split.amountMinor, value.currency)]))
    return
  }
  if (props.editKind === 'subscription' || ('name' in record)) {
    const value = record as Subscription
    mode.value = 'subscription'
    Object.assign(form, { title: value.name, amount: minorToInput(value.amountMinor, value.currency), currency: value.currency, category: value.category, categoryId: value.categoryId || '', date: value.billingCycle === 'every_n_hours' ? (value.startsOn || value.nextBilling).slice(0, 16) : (value.startsOn || value.nextBilling).slice(0, 10), notes: value.notes, payer: value.paidBy || auth.record?.id || '', rateMode: value.rateMode || 'automatic', exchangeRate: value.exchangeRate || '', splitMode: value.splitMode || 'equal', billingCycle: value.billingCycle, billingInterval: String(value.billingInterval || 1), status: value.status, revisionScope: value.revisionScope || 'future', effectiveBillingAt: value.effectiveBillingAt || value.nextBilling, endBillingAt: value.endBillingAt || '' })
    form.participants = Object.fromEntries((value.splits || members.value.map(member => ({ userId: member.userId }))).map(split => [split.userId, true]))
    form.values = Object.fromEntries((value.splits || []).map(split => [split.userId, value.splitMode === 'percentage' ? String((split.percentageBasisPoints || 0) / 100) : minorToInput(split.amountMinor, value.currency)]))
    return
  }
  const value = record as Expense
  mode.value = 'expense'
  Object.assign(form, { title: value.title, amount: minorToInput(value.amountMinor, value.currency), currency: value.currency, category: value.category, categoryId: value.categoryId || '', date: value.incurredOn.slice(0, 10), notes: value.notes, payer: value.paidBy || auth.record?.id || '', rateMode: value.rateMode || 'automatic', exchangeRate: value.exchangeRate || '', splitMode: value.splitMode || 'equal' })
  form.participants = Object.fromEntries((value.splits || members.value.map(member => ({ userId: member.userId }))).map(split => [split.userId, true]))
  form.values = Object.fromEntries((value.splits || []).map(split => [split.userId, value.splitMode === 'percentage' ? String((split.percentageBasisPoints || 0) / 100) : minorToInput(split.amountMinor, value.currency)]))
}

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
  if (!form.title.trim() || amountMinor.value <= 0 || !form.payer || !splitValid.value || !rateValid.value) {
    error.value = tr('invalidRequest')
    return
  }
  saving.value = true
  error.value = ''
  const occurred = form.billingCycle === 'every_n_hours' ? fromDateTimeInput(form.date, personal.value ? (auth.record?.timezone || 'UTC') : (group.value?.timezone || 'UTC')) : fromDateInput(form.date, personal.value ? (auth.record?.timezone || 'UTC') : (group.value?.timezone || 'UTC'))
  const common = {
    title: form.title.trim(),
    category: form.category,
    categoryId: form.categoryId,
    amountMinor: amountMinor.value,
    currency: form.currency,
    rateMode: form.rateMode,
    exchangeRate: form.exchangeRate,
    notes: form.notes,
    splitMode: form.splitMode,
  }
  const record = props.editRecord
  let ok = false
  if (mode.value === 'expense') {
    const input = { ...common, paidBy: form.payer, incurredOn: occurred, ...(personal.value ? {} : { splits: expenseSplits() }) }
    ok = personal.value ? (record && props.editKind === 'expense' ? await workspace.updateExpense(record.id, input) : await workspace.addPersonalExpense(input)) : (record && props.editKind === 'expense' ? await workspace.updateExpense(record.id, input) : await workspace.addExpense(input))
  } else if (mode.value === 'income') {
    const input = { ...common, earnedBy: form.payer, receivedOn: occurred, ...(personal.value ? {} : { splits: incomeSplits() }) }
    ok = personal.value ? (record && props.editKind === 'income' ? await workspace.updateIncome(record.id, input) : await workspace.addIncome(input)) : (record && props.editKind === 'income' ? await workspace.updateGroupIncome(record.id, input) : await workspace.addGroupIncome(input))
  } else {
    const input = { name: common.title, category: common.category, categoryId: common.categoryId, amountMinor: common.amountMinor, currency: common.currency, rateMode: common.rateMode, exchangeRate: common.exchangeRate, paidBy: form.payer, splitMode: common.splitMode, ...(personal.value ? {} : { splits: expenseSplits() }), billingCycle: form.billingCycle, billingInterval: Number(form.billingInterval), startsOn: occurred, nextBilling: occurred, status: form.status, notes: common.notes, revisionScope: form.revisionScope, effectiveBillingAt: form.effectiveBillingAt || occurred, endBillingAt: form.endBillingAt || undefined }
    ok = record && props.editKind === 'subscription' ? await workspace.updateSubscription(record.id, input) : personal.value ? await workspace.addPersonalSubscription(input) : await workspace.addSubscription(input)
  }
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
  <AppDrawer :open="open" :title="tr(editing ? (mode === 'subscription' ? 'editSubscription' : mode === 'income' ? 'editIncome' : 'editExpense') : 'addRecord')" @close="emit('close')">
    <form class="form-card ledger-form" @submit.prevent="submit">
      <div v-if="error" class="notice danger inline">{{ error }}</div>
      <div class="mode-tabs">
        <button type="button" :class="{active:mode==='expense'}" :disabled="editing || !canExpense" @click="mode='expense'">{{ tr('expense') }}</button>
        <button type="button" :class="{active:mode==='income'}" :disabled="editing || !canIncome" @click="mode='income'">{{ tr('income') }}</button>
        <button type="button" :class="{active:mode==='subscription'}" :disabled="editing || !canSubscription" @click="mode='subscription'">{{ tr('subscriptions') }}</button>
      </div>
      <section class="ledger-form-section">
        <div class="ledger-section-heading"><strong>{{ tr('item') }}</strong></div>
        <div class="ledger-form-grid">
          <BaseInput v-model="form.title" class="ledger-wide" :label="tr('item')" required :placeholder="tr('itemPlaceholder')" />
          <div class="ledger-field ledger-wide"><span>{{ tr('category') }}</span><CategorySelect v-model="form.categoryId" :categories="workspace.categories" @create="addCategory" /></div>
        </div>
      </section>
      <section class="ledger-form-section">
        <div class="ledger-section-heading"><strong>{{ tr('amount') }} · {{ tr('currency') }}</strong></div>
        <div class="ledger-form-grid">
          <BaseInput v-model="form.amount" :label="tr('amount')" type="number" inputmode="decimal" min="0" :step="amountStep(form.currency)" required />
          <div class="ledger-field"><span>{{ tr('currency') }}</span><CurrencySelect v-model="form.currency" :currencies="workspace.currencies" /></div>
          <div v-if="!personal" class="ledger-field"><span>{{ mode==='income' ? tr('receivedBy') : tr('payer') }}</span><PayerSelect v-model="form.payer" :members="members" :self-id="auth.record?.id" /></div>
          <BaseInput v-if="mode!=='subscription'" v-model="form.date" :label="tr('date')" type="date" required />
        </div>
      </section>
      <section class="ledger-form-section">
        <div class="ledger-section-heading"><strong>{{ tr('exchangeRate') }}</strong></div>
        <div class="ledger-form-grid">
          <BaseCombobox v-model="form.rateMode" :options="rateOptions" :label="tr('exchangeRate')" :allow-create="false" />
          <BaseInput v-if="form.rateMode==='manual' && form.currency!==reportingCurrency" v-model="form.exchangeRate" :label="tr('manualRate')" inputmode="decimal" required />
          <ConversionPreview class="ledger-wide" :from="form.currency" :to="reportingCurrency" :amount="form.amount" :date="form.date" :mode="form.rateMode" :manual-rate="form.exchangeRate" @validity="rateValid=$event" />
        </div>
      </section>
      <section v-if="!personal" class="ledger-form-section"><div class="ledger-section-heading"><strong>{{ tr('splitMode') }}</strong></div>
        <BaseCombobox v-model="form.splitMode" :options="splitOptions" :label="tr('splitMode')" :allow-create="false" />
        <fieldset class="split-editor">
          <legend>{{ tr('participants') }}</legend>
          <label v-for="member in members" :key="member.userId" class="split-member">
            <input v-model="form.participants[member.userId]" type="checkbox">
            <span>{{ member.user?.name || member.user?.email || member.userId }}</span>
            <input v-if="form.participants[member.userId] && form.splitMode !== 'equal'" v-model="form.values[member.userId]" type="number" min="0" :step="form.splitMode==='percentage' ? '0.01' : amountStep(form.currency)" :aria-label="member.user?.name || member.userId">
          </label>
          <p :class="splitValid ? 'success' : 'form-error'">{{ splitValid ? tr('splitValid') : tr(form.splitMode==='percentage' ? 'splitInvalidPercentage' : 'splitInvalidAmount') }}</p>
        </fieldset>
      </section>
      <section v-if="mode==='subscription'" class="ledger-form-section">
        <div class="ledger-section-heading"><strong>{{ tr('cycle') }}</strong></div>
        <div class="ledger-form-grid">
          <BaseCombobox v-model="form.billingCycle" :options="cycleOptions" :label="tr('cycle')" :allow-create="false" />
          <BaseInput v-if="['every_n_days','every_n_weeks','every_n_hours'].includes(form.billingCycle)" v-model="form.billingInterval" :label="tr('billingInterval')" type="number" min="1" max="8760" required />
          <BaseInput v-model="form.date" class="ledger-wide" :label="tr('firstBilling')" :type="form.billingCycle === 'every_n_hours' ? 'datetime-local' : 'date'" required />
          <BaseCombobox v-model="form.status" :options="[{value:'active',label:tr('active')},{value:'paused',label:tr('paused')},{value:'cancelled',label:tr('cancelled')}]" :label="tr('status')" :allow-create="false" />
        </div>
      </section>      <section class="ledger-form-section">
        <label class="ledger-field"><span>{{ tr('notes') }}</span><textarea v-model="form.notes" rows="3" :placeholder="tr('notes')"></textarea></label>
      </section>
      <div class="form-actions ledger-form-actions">
        <button type="button" class="ghost" @click="emit('close')">{{ tr('cancel') }}</button>
        <button class="primary" :disabled="saving || workspace.loading || !splitValid || !rateValid">{{ saving ? tr('processing') : tr('saveRecord') }}</button>
      </div>
    </form>
  </AppDrawer>
</template>

<style scoped>
.ledger-form{display:grid;gap:18px;padding-bottom:0}.ledger-form-section{display:grid;gap:12px;padding:16px;border:1px solid var(--line);border-radius:14px;background:var(--surface-soft)}.ledger-section-heading{display:flex;align-items:center;justify-content:space-between;gap:10px;color:var(--ink);font-size:13px}.ledger-form-grid{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:12px}.ledger-form-grid>label,.ledger-form-section>label{min-width:0}.ledger-wide{grid-column:1/-1}.ledger-field{display:grid;gap:7px;color:var(--ink);font-size:var(--font-size-label)}.mode-tabs{display:grid;grid-template-columns:repeat(3,1fr);gap:.4rem;background:var(--surface-soft);border:1px solid var(--line);padding:.3rem;border-radius:12px;margin:0}.mode-tabs button{border:1px solid transparent;background:none;color:var(--muted);padding:.65rem;border-radius:9px;cursor:pointer}.mode-tabs button:hover:not(:disabled){color:var(--ink);background:var(--surface)}.mode-tabs button.active{border-color:var(--brand);background:var(--brand-soft);color:var(--brand)}.mode-tabs button:disabled{cursor:not-allowed;opacity:.5}.split-editor{display:grid;gap:8px;margin:0;padding:14px;border:1px solid var(--line);border-radius:12px;background:var(--surface)}.split-editor legend{padding:0 5px;color:var(--ink);font-size:12px;font-weight:800}.split-member{display:grid!important;grid-template-columns:20px minmax(0,1fr) 110px;align-items:center!important;gap:9px!important}.split-member input[type=checkbox]{width:16px;height:16px;accent-color:var(--brand)}.split-member input:last-child{width:100%;min-height:38px;padding:8px 10px;border:1px solid var(--line-strong);border-radius:9px;background:var(--surface-soft);color:var(--ink)}.split-member span{min-width:0;overflow:hidden;text-overflow:ellipsis;white-space:nowrap}.ledger-form textarea{width:100%;box-sizing:border-box;min-height:88px;padding:10px 12px;border:1px solid var(--line-strong);border-radius:10px;background:var(--surface-soft);color:var(--ink);resize:vertical}.ledger-form textarea:focus{border-color:var(--brand)}.form-actions{display:flex;justify-content:flex-end;gap:10px}.ledger-form-actions{position:sticky;bottom:0;z-index:2;margin:0 -24px;padding:14px 24px;background:color-mix(in srgb,var(--surface) 92%,transparent);border-top:1px solid var(--line);backdrop-filter:blur(12px)}.success{color:var(--success)}.form-error{color:var(--danger)}@media(max-width:640px){.ledger-form-grid{grid-template-columns:minmax(0,1fr)}.ledger-wide{grid-column:auto}.split-member{grid-template-columns:20px minmax(0,1fr) 92px}.ledger-form-actions{margin:0 -18px;padding:12px 18px}}
</style>
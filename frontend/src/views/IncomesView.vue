<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import { useWorkspaceStore } from '../stores/workspace'
import MoneyValue from '../components/MoneyValue.vue'
import EmptyState from '../components/EmptyState.vue'
import AppDrawer from '../components/AppDrawer.vue'
import ConfirmDialog from '../components/ConfirmDialog.vue'
import PersonalLedgerNav from '../components/PersonalLedgerNav.vue'
import SyncBadge from '../components/SyncBadge.vue'
import CurrencySelect from '../components/CurrencySelect.vue'
import CategorySelect from '../components/CategorySelect.vue'
import PayerSelect from '../components/PayerSelect.vue'
import ConversionPreview from '../components/ConversionPreview.vue'
import SourceDialog from '../components/SourceDialog.vue'
import Pagination from '../components/Pagination.vue'
import PageSizeSelect from '../components/PageSizeSelect.vue'
import type { Currency, Income, IncomeSplit, SplitMode } from '../api/types'
import { amountStep, majorToMinor, minorToInput } from '../api/money'
import { useI18n } from '../i18n'
import { timezoneLabel } from '../timezone'
import { categoryGlyph, categoryLabel } from '../category'
import { fromDateInput, toDateInput, todayInput } from '../dateInput'
import { defaultPageSize } from '../pageSize'

const workspace = useWorkspaceStore()
const auth = useAuthStore()
const route = useRoute()
const { tr, formatDate } = useI18n()

const selectableMembers = computed(() => workspace.members.filter(m => !(m.user?.placeholder && m.user?.linkedUserId)))
function viewerTimezone() { return auth.record?.timezone || Intl.DateTimeFormat().resolvedOptions().timeZone || 'UTC' }
const personal = computed(() => route.name === 'personal-incomes')
const canWrite = computed(() => personal.value || workspace.groupPermissions.includes('ledger.incomes.write'))
const canDelete = computed(() => personal.value || workspace.groupPermissions.includes('ledger.incomes.delete'))
const canExport = computed(() => personal.value || ['group.view', 'ledger.incomes.read', 'ledger.subscriptions.read', 'ledger.settlements.read'].every(permission => workspace.groupPermissions.includes(permission)))
const list = computed(() => personal.value ? workspace.personalIncomes : workspace.groupIncomes)
const hasUnsynced = computed(() => list.value.some(item => item.pendingSync || item.syncError))
const listMeta = computed(() => personal.value ? workspace.personalIncomesMeta : workspace.groupIncomesMeta)
const perPage = ref(defaultPageSize.value)
function goToPage(page: number) { void (personal.value ? workspace.loadPersonalIncomesPage(page, perPage.value) : workspace.loadGroupIncomesPage(page, perPage.value)) }
function changePageSize(value: number) { perPage.value = value; goToPage(1) }
async function reloadCurrentPage(page: number) {
    if (personal.value) await workspace.loadPersonalIncomesPage(page, perPage.value)
    else await workspace.loadGroupIncomesPage(page, perPage.value)
    if (listMeta.value.totalPages && listMeta.value.page > listMeta.value.totalPages) {
        if (personal.value) await workspace.loadPersonalIncomesPage(listMeta.value.totalPages, perPage.value)
        else await workspace.loadGroupIncomesPage(listMeta.value.totalPages, perPage.value)
    }
}
const open = ref(false), editingId = ref(''), pendingDelete = ref<Income>(), sourceItem = ref<Income>(), formError = ref(''), rateValid = ref(true), exportError = ref(''), exporting = ref(false)
const form = reactive({ title: '', category: '', categoryId: '', amount: '', currency: 'TWD' as Currency, rateMode: 'automatic' as 'automatic' | 'manual', exchangeRate: '', earnedBy: '', receivedOn: todayInput(viewerTimezone()), notes: '', splitMode: 'equal' as SplitMode, participants: {} as Record<string, boolean>, values: {} as Record<string, string> })
// See SubscriptionsView.vue: only send receivedOn back when the user actually
// touched the date input, so an untouched value keeps its exact stored
// instant instead of a same-day-but-shifted reconstruction.
const receivedOnTouched = ref(false)
const editing = computed(() => list.value.find(item => item.id === editingId.value))
const sourceGroup = computed(() => workspace.groups.find(group => group.id === editing.value?.groupId))
const currency = computed(() => form.currency || workspace.currentGroup?.currency || 'TWD')
const reportingCurrency = computed(() => sourceGroup.value?.currency || (!personal.value ? workspace.currentGroup?.currency : currency.value) || currency.value)
const participants = computed(() => workspace.members.filter(member => form.participants[member.userId]))
const splitTotalMinor = computed(() => participants.value.reduce((sum, member) => sum + (form.splitMode === 'amount' ? majorToMinor(form.values[member.userId] || '0', currency.value) : form.splitMode === 'percentage' ? Math.round(Number(form.values[member.userId] || 0) * 100) : 0), 0))
const incomeMinor = computed(() => majorToMinor(form.amount || '0', currency.value))
// A true personal (groupless) expense has no split editor at all (see the
// v-if below), so form.participants is never populated for one — it must
// always be considered valid rather than falling into the "needs at least
// one participant" branch below, or the create/save button stays disabled
// forever.
const splitValid = computed(() => personal.value && !editing.value?.groupId ? true : form.splitMode === 'equal' ? participants.value.length > 0 : form.splitMode === 'amount' ? splitTotalMinor.value === incomeMinor.value : splitTotalMinor.value === 10000)
watch([currency, reportingCurrency], ([from, to]) => { if (from === to) { form.rateMode = 'automatic'; form.exchangeRate = '' } })

function reset() { editingId.value = ''; formError.value = ''; rateValid.value = true; receivedOnTouched.value = false; Object.assign(form, { title: '', category: '', categoryId: '', amount: '', currency: workspace.currentGroup?.currency || auth.record?.defaultCurrency || 'TWD', rateMode: 'automatic', exchangeRate: '', earnedBy: auth.record?.id || '', receivedOn: todayInput(viewerTimezone()), notes: '', splitMode: 'equal', participants: {}, values: {} }) }
async function loadFormCategories() { try { await workspace.loadCategories(personal.value && !editing.value?.groupId ? 'personal' : 'group', editing.value?.groupId || workspace.currentGroupId) } catch { formError.value = workspace.localizedError || tr('requestFailed') } }
async function create() { if (!canWrite.value) return; reset(); if (!personal.value) selectableMembers.value.forEach(m => { form.participants[m.userId] = true }); open.value = true; await loadFormCategories() }
async function edit(item: Income) { if (!canWrite.value) return; reset(); editingId.value = item.id; if (item.groupId && workspace.currentGroupId !== item.groupId) await workspace.selectGroup(item.groupId); const selected = item.splits?.map(split => split.userId) || workspace.members.map(m => m.userId); Object.assign(form, { title: item.title, category: item.category, categoryId: item.categoryId || '', amount: minorToInput(item.amountMinor, item.currency), currency: item.currency || 'TWD', rateMode: item.rateMode || 'automatic', exchangeRate: item.exchangeRate || '', earnedBy: item.earnedBy, receivedOn: toDateInput(item.receivedOn, viewerTimezone()), notes: item.notes, splitMode: item.splitMode || 'equal', participants: Object.fromEntries(selected.map(id => [id, true])), values: Object.fromEntries((item.splits || []).map(split => [split.userId, (item.splitMode === 'percentage' ? (split.percentageBasisPoints || 0) / 100 : minorToInput(split.amountMinor, item.currency)).toString()])) }); open.value = true; await loadFormCategories() }
async function addCategory(name: string, icon = 'tag') { try { const value = await workspace.createCategory(personal.value && !editing.value?.groupId ? 'personal' : 'group', name, editing.value?.groupId || workspace.currentGroupId, icon); form.categoryId = value.id } catch { formError.value = workspace.localizedError || tr('requestFailed') } }
function canonicalSplits(): IncomeSplit[] { return participants.value.map(member => ({ userId: member.userId, amountMinor: form.splitMode === 'amount' ? majorToMinor(form.values[member.userId] || '0', currency.value) : 0, percentageBasisPoints: form.splitMode === 'percentage' ? Math.round(Number(form.values[member.userId] || 0) * 100) : undefined })) }
async function submit() {
    if (!splitValid.value || !rateValid.value) return
    const page = listMeta.value.page
    const receivedOn = fromDateInput(form.receivedOn, viewerTimezone())
    const input = { title: form.title, category: form.category || '', categoryId: form.categoryId, amountMinor: incomeMinor.value, currency: currency.value, rateMode: form.rateMode, exchangeRate: form.exchangeRate, earnedBy: form.earnedBy || auth.record?.id || '', ...(!editingId.value || receivedOnTouched.value ? { receivedOn } : {}), notes: form.notes, splitMode: form.splitMode, splits: personal.value && !editing.value?.groupId ? undefined : canonicalSplits() }
    const ok = editingId.value ? personal.value ? await workspace.updateIncome(editingId.value, input) : await workspace.updateGroupIncome(editingId.value, input) : personal.value ? await workspace.addIncome(input) : await workspace.addGroupIncome(input)
    if (!ok) { formError.value = workspace.localizedError || tr('requestFailed'); return }
    await reloadCurrentPage(page)
    open.value = false
    reset()
}
async function remove() { if (!pendingDelete.value || !canDelete.value) return; const page = listMeta.value.page; if (personal.value) await workspace.deleteIncome(pendingDelete.value.id); else await workspace.deleteGroupIncome(pendingDelete.value.id); await reloadCurrentPage(page); pendingDelete.value = undefined }
async function exportLedger() { exportError.value = ''; exporting.value = true; try { await workspace.exportLedger(personal.value ? undefined : workspace.currentGroupId) } catch { exportError.value = tr('exportFailed') } finally { exporting.value = false } }
function recordCurrency(item: Income) { return item.currency || workspace.groups.find(group => group.id === item.groupId)?.currency || 'TWD' }
function recordCategory(item: Income) { return `${categoryGlyph(item.categoryInfo)} ${categoryLabel(item.categoryInfo, item.category, tr)}` }
function itemGroup(item: Income) { return workspace.groups.find(group => group.id === item.groupId) }
function viewerDate(value: string) { return formatDate(value, { dateStyle: 'medium', timeZone: viewerTimezone() }) }
function originalTime(item: Income) { const group = itemGroup(item); return group ? tr('originalTimezone', { date: formatDate(item.receivedOn, { dateStyle: 'medium', timeZone: group.timezone }), timezone: timezoneLabel(group.timezone, item.receivedOn) }) : '' }
function openSourceFromBadge(event: MouseEvent) {
    const badge = (event.target as HTMLElement).closest('.source-badge')
    if (!badge) return
    const name = badge.textContent?.trim() || ''
    sourceItem.value = list.value.find(item => (itemGroup(item)?.name || tr('privateRecord')) === name)
}
onMounted(() => { if (personal.value) void workspace.refreshPersonal(); document.addEventListener('click', openSourceFromBadge) })
onBeforeUnmount(() => document.removeEventListener('click', openSourceFromBadge))
</script>

<template>
    <section class="page ledger-page">
        <PersonalLedgerNav v-if="personal" />
        <div class="page-heading">
            <div>
                <p class="eyebrow">{{ tr(personal ? 'incomePersonal' : 'groupIncomes') }}</p>
                <h1>{{ tr(personal ? 'incomePersonal' : 'groupIncomes') }}</h1>
                <p>{{ tr('incomeDesc') }}</p>
            </div>
            <div class="page-heading-actions"><button v-if="canExport" class="ghost"
                    :disabled="exporting || !workspace.online"
                    :title="workspace.online ? '' : tr('offlineActionDisabled')" @click="exportLedger">{{
                        tr('exportLedger') }}</button><button v-if="canWrite" class="primary" @click="create">{{
                        tr('createIncome') }}</button></div>
        </div>
        <div v-if="exportError" class="notice danger inline">{{ exportError }}</div>
        <section class="card data-card">
            <div class="card-title">
                <h2>{{ tr('recentIncomes') }}</h2><span>{{ tr('records', { count: listMeta.totalItems }) }}</span>
                <PageSizeSelect :model-value="perPage" @update:model-value="changePageSize" />
            </div>
            <div v-if="!personal && workspace.groupErrors.incomes" class="resource-error">
                <p>{{ workspace.groupErrors.incomes }}</p><button class="ghost" @click="workspace.refreshGroup()">{{
                    tr('retry') }}</button>
            </div>
            <div v-else-if="list.length" class="data-table income-table">
                <div class="data-table-head">
                    <span>{{ tr('item') }}</span><span>{{ tr('source') }}</span><span>{{ tr('receivedBy') }}</span><span>{{
                        tr('date') }}</span><span>{{ tr('amount') }}</span><span></span>
                </div>
                <article v-for="item in list" :key="item.id" class="data-table-row">
                    <div class="item-cell"><span class="service-icon income">{{ item.title.slice(0, 1)
                            }}</span><span><strong>{{ item.title }}</strong><small>{{ recordCategory(item) }}</small>
                            <SyncBadge :pending-sync="item.pendingSync" :sync-error="item.syncError" />
                        </span></div><span><span class="source-badge" :class="{ shared: item.groupId }">{{
                            itemGroup(item)?.name || tr('privateRecord')
                            }}</span></span><span>{{workspace.members.find(m => m.userId === item.earnedBy)?.user?.name ||
                                (item.earnedBy === auth.record?.id ? tr('myself') : '—')}}</span><span
                        class="timezone-date"><strong>{{ viewerDate(item.receivedOn) }}</strong><small
                            v-if="item.groupId">{{ originalTime(item) }}</small></span><span class="money-stack">
                        <MoneyValue :amount="item.amountMinor" :currency="recordCurrency(item)" /><small
                            v-if="item.baseCurrency && item.baseCurrency !== item.currency">{{ tr('reportingAmount') }}:
                            <MoneyValue :amount="item.baseAmountMinor" :currency="item.baseCurrency" />
                        </small><small v-if="item.exchangeRate">{{ tr('exchangeRate') }} {{ item.exchangeRate }}</small>
                    </span><span class="row-actions"><button v-if="canWrite" class="icon-button"
                            :aria-label="tr('edit')" @click="edit(item)">✎</button><button v-if="canDelete"
                            class="icon-button" :aria-label="tr('remove')"
                            @click="pendingDelete = item">×</button></span>
                </article>
            </div>
            <EmptyState v-else :title="tr('noIncomes')" :description="tr('noIncomesDesc')" />
            <p v-if="hasUnsynced" class="field-help sync-legend"><strong>{{ tr('syncLegendTitle') }}</strong> · ☁︎/
                {{ tr('syncLegendPending') }} · ⚠ {{ tr('syncLegendError') }}</p>
            <Pagination :meta="listMeta" @page="goToPage" />
        </section>
        <AppDrawer :open="open" :title="tr(editingId ? 'editIncome' : 'createIncome')" @close="open = false">
            <form class="form-card ledger-form" @submit.prevent="submit">
                <div v-if="formError" class="notice danger inline">{{ formError }}</div>
                <div v-if="editing?.groupId && personal" class="notice inline">
                    {{ tr('sharedRecordWarning', { group: sourceGroup?.name || tr('groups') }) }}</div>
                <div v-if="sourceGroup || (!personal && workspace.currentGroup)" class="timezone-notice">
                    {{ tr('yourTimezone', { timezone: timezoneLabel(viewerTimezone()) }) }}<br>{{
                        tr('groupTimezoneValue', {
                            timezone: timezoneLabel((sourceGroup || workspace.currentGroup)?.timezone
                    || 'UTC') }) }}
                </div>
                <section class="ledger-form-section">
                    <div class="ledger-section-heading"><strong>{{ tr('item') }}</strong></div>
                    <div class="ledger-form-grid"><label class="ledger-wide">{{ tr('item') }}<input v-model="form.title"
                                required :placeholder="tr('itemPlaceholder')"></label><label class="ledger-wide">{{
                            tr('category') }}
                            <CategorySelect v-model="form.categoryId" :categories="workspace.categories"
                                @create="addCategory" />
                        </label></div>
                </section>
                <section class="ledger-form-section">
                    <div class="ledger-section-heading"><strong>{{ tr('amount') }} · {{ tr('currency') }}</strong></div>
                    <div class="ledger-form-grid"><label>{{ tr('amount') }}<input v-model="form.amount" type="number"
                                min="0" :step="amountStep(form.currency)" required></label><label>{{ tr('currency') }}
                            <CurrencySelect v-model="form.currency" :currencies="workspace.currencies" />
                        </label><label v-if="!personal">{{ tr('receivedBy') }}
                            <PayerSelect v-model="form.earnedBy" :members="selectableMembers"
                                :self-id="auth.record?.id" />
                        </label><label>{{ tr('date') }}<input v-model="form.receivedOn" type="date" required
                                @change="receivedOnTouched = true"></label></div>
                </section>
                <section class="ledger-form-section">
                    <div class="ledger-section-heading"><strong>{{ tr('exchangeRate') }}</strong></div>
                    <div class="ledger-form-grid"><label>{{ tr('exchangeRate') }}<select v-model="form.rateMode"
                                :disabled="currency === reportingCurrency">
                                <option value="automatic">{{ tr('automaticRate') }}</option>
                                <option value="manual">{{ tr('manualRate') }}</option>
                            </select></label><label
                            v-if="form.rateMode === 'manual' && currency !== reportingCurrency">{{ tr('manualRate')
                            }}<input v-model="form.exchangeRate" inputmode="decimal" required></label>
                        <div v-else class="ledger-rate-placeholder"></div>
                        <ConversionPreview class="ledger-wide" :from="currency" :to="reportingCurrency"
                            :amount="form.amount" :date="form.receivedOn.slice(0, 10)" :mode="form.rateMode"
                            :manual-rate="form.exchangeRate" @validity="rateValid = $event" />
                    </div>
                </section>
                <section v-if="!personal || editing?.groupId" class="ledger-form-section">
                    <div class="ledger-section-heading"><strong>{{ tr('splitMode') }}</strong></div>
                    <label>{{ tr('splitMode') }}<select v-model="form.splitMode">
                            <option value="equal">{{ tr('splitEqual') }}</option>
                            <option value="amount">{{ tr('splitAmount') }}</option>
                            <option value="percentage">{{ tr('splitPercentage') }}</option>
                        </select></label>
                    <fieldset class="split-editor">
                        <legend>{{ tr('participants') }}</legend><label v-for="member in selectableMembers"
                            :key="member.userId" class="split-member"><input v-model="form.participants[member.userId]"
                                type="checkbox"><span>{{ member.user?.name || member.user?.email }}</span><input
                                v-if="form.participants[member.userId] && form.splitMode !== 'equal'"
                                v-model="form.values[member.userId]" type="number" min="0"
                                :step="form.splitMode === 'percentage' ? '0.01' : amountStep(form.currency)"
                                :aria-label="member.user?.name"></label>
                        <p :class="splitValid ? 'success' : 'form-error'">
                            {{ splitValid ? tr('splitValid') : tr(form.splitMode === 'percentage' ?
                                'splitInvalidPercentage' : 'splitInvalidAmount') }}
                        </p>
                    </fieldset>
                </section>
                <section class="ledger-form-section"><label>{{ tr('notes') }}<textarea v-model="form.notes"
                            rows="3"></textarea></label></section>
                <div class="form-actions ledger-form-actions"><button type="button" class="ghost"
                        @click="open = false">{{ tr('cancel') }}</button><button class="primary"
                        :disabled="workspace.loading || !splitValid || !rateValid">{{ tr(editingId ? 'saveChanges' :
                        'createIncome') }}</button>
                </div>
            </form>
        </AppDrawer>
        <ConfirmDialog :open="!!pendingDelete"
            :title="pendingDelete ? tr('removeIncomeConfirm', { name: pendingDelete.title }) : ''" danger
            @cancel="pendingDelete = undefined" @confirm="remove" />
        <SourceDialog :open="!!sourceItem" kind="incomes" :group="sourceItem ? itemGroup(sourceItem) : undefined"
            :currency="sourceItem?.currency || 'TWD'" @close="sourceItem = undefined" />
    </section>
</template>
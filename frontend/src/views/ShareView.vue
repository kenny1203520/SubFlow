<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { pb } from '../pocketbase'
import SharePagination from '../components/SharePagination.vue'
import type { SharePage, ShareRecord } from '../api/types'
import { useI18n } from '../i18n'

const route = useRoute()
const router = useRouter()
const { tr, locale, formatDate } = useI18n()
const token = String(route.params.token || '')
const page = ref<SharePage>()
const password = ref('')
const loading = ref(true)
const error = ref('')
const accountAccess = ref(false)
const activeSection = ref<'all' | 'expenses' | 'subscriptions' | 'settlements'>('all')
const search = ref('')
const sortNewest = ref(true)
const pageNumber = ref(1)
const pageSize = ref(25)
const pageSizes = [5, 10, 15, 25]
let requestSequence = 0
let filterTimer: ReturnType<typeof setTimeout> | undefined

const records = computed(() => page.value?.records || [])
const balances = computed(() => page.value?.balances || [])
const hasRecords = computed(() => (page.value?.totalItems || records.value.length) > 0)
const totals = computed(() => Object.entries(page.value?.summary?.expenseTotals || {}).map(([currency, amount]) => ({ currency, amount })))
const sectionTabs = computed(() => {
  const counts = page.value?.recordCounts || { all: 0, expenses: 0, subscriptions: 0, settlements: 0 }
  return [
    { key: 'all' as const, label: tr('shareAllRecords'), count: counts.all || 0 },
    { key: 'expenses' as const, label: tr('shareExpenses'), count: counts.expenses || 0 },
    { key: 'subscriptions' as const, label: tr('shareSubscriptions'), count: counts.subscriptions || 0 },
    { key: 'settlements' as const, label: tr('shareSettlements'), count: counts.settlements || 0 },
  ].filter(tab => tab.key === 'all' || tab.count > 0)
})

function text(value: unknown) { return value == null ? '' : String(value) }
function money(amount: unknown, currency: unknown) {
  return new Intl.NumberFormat(locale.value, { style: 'currency', currency: text(currency) || 'TWD' }).format(Number(amount || 0) / 100)
}
function date(value: unknown) { return value ? formatDate(text(value)) : '' }
function rangeLabel(value: unknown) {
  const raw = text(value)
  if (raw === 'all') return tr('shareRangeAll')
  if (raw === 'rolling') return tr('shareRangeRolling')
  if (raw === 'fixed') return tr('shareRangeFixed')
  return raw
}
function cycleLabel(value: unknown) {
  const key = text(value)
  return ({ monthly: tr('monthly'), quarterly: tr('quarterly'), yearly: tr('yearly') } as Record<string, string>)[key] || key
}
function statusLabel(value: unknown) {
  const key = text(value)
  return ({ active: tr('active'), paused: tr('paused'), cancelled: tr('cancelled'), ending: tr('ending'), ended: tr('ended') } as Record<string, string>)[key] || key
}
function recordTitle(record: ShareRecord) {
  if (record.type === 'expense') return text(record.title) || tr('shareExpenses')
  if (record.type === 'subscription') return text(record.name) || tr('shareSubscriptions')
  if (record.from || record.to) return tr('sharePublicFromTo', { from: text(record.from), to: text(record.to) })
  return tr('shareSettlements')
}
function recordTypeLabel(type: unknown) {
  return ({ expense: tr('shareExpenses'), subscription: tr('shareSubscriptions'), settlement: tr('shareSettlements') } as Record<string, string>)[text(type)] || tr('sharePublicRecords')
}
function recordDate(record: ShareRecord) { return record.occurredOn || record.incurredOn || record.nextBilling || record.settledOn }
function recordMeta(record: ShareRecord) {
  if (record.type === 'expense') return [record.category, date(recordDate(record)), record.paidBy ? tr('sharePublicPaidBy', { value: text(record.paidBy) }) : ''].filter(Boolean)
  if (record.type === 'subscription') return [record.category, cycleLabel(record.billingCycle), statusLabel(record.status), date(recordDate(record)), record.paidBy ? tr('sharePublicPaidBy', { value: text(record.paidBy) }) : ''].filter(Boolean)
  return [date(recordDate(record))]
}
function recordKey(record: ShareRecord, index: number) { return [record.type, record.occurredOn, record.title, record.name, record.amountMinor, index].map(text).join('-') }
function queryFor(target: number, size = pageSize.value) {
  const query = new URLSearchParams({ page: String(target), perPage: String(size), section: activeSection.value, sort: sortNewest.value ? 'newest' : 'oldest' })
  if (search.value.trim()) query.set('q', search.value.trim())
  return query.toString()
}
async function request(path: string, options: RequestInit = {}) {
  const headers = new Headers(options.headers)
  headers.set('Accept', 'application/json')
  if (pb.authStore.token) headers.set('Authorization', `Bearer ${pb.authStore.token}`)
  if (options.body) headers.set('Content-Type', 'application/json')
  const response = await fetch(`/api/subflow/v1${path}`, { ...options, headers })
  const body = await response.json().catch(() => null)
  if (!response.ok) throw new Error(body?.error?.message || tr('sharePublicUnavailable'))
  return body.data as SharePage
}
async function load() {
  const sequence = ++requestSequence
  loading.value = true
  error.value = ''
  try {
    const result = await request('/shares/' + token + '?' + queryFor(1))
    if (sequence !== requestSequence) return
    if (result.requiresLogin) {
      if (!pb.authStore.isValid) { await router.push({ name: 'auth', query: { redirect: route.fullPath } }); return }
      accountAccess.value = true
      page.value = await request('/shares/' + token + '/account?' + queryFor(1))
    } else page.value = result
    pageNumber.value = page.value?.page || 1
    pageSize.value = page.value?.perPage || pageSize.value
  } catch {
    if (sequence === requestSequence) error.value = tr('sharePublicUnavailable')
  } finally {
    if (sequence === requestSequence) loading.value = false
  }
}
async function unlock() {
  const sequence = ++requestSequence
  loading.value = true
  error.value = ''
  try {
    const result = await request('/shares/' + token + '/access?' + queryFor(1), { method: 'POST', body: JSON.stringify({ password: password.value }) })
    if (sequence !== requestSequence) return
    page.value = result
    pageNumber.value = result.page || 1
    pageSize.value = result.perPage || pageSize.value
  } catch {
    if (sequence === requestSequence) error.value = tr('sharePublicUnavailable')
  } finally {
    if (sequence === requestSequence) loading.value = false
  }
}
async function loadPage(target: number) {
  const max = page.value?.totalPages || 1
  if (target < 1 || target > max) return
  const sequence = ++requestSequence
  loading.value = true
  error.value = ''
  try {
    const base = accountAccess.value ? '/shares/' + token + '/account' : '/shares/' + token
    const result = await request(base + '?' + queryFor(target))
    if (sequence !== requestSequence) return
    page.value = result
    pageNumber.value = result.page || target
    pageSize.value = result.perPage || pageSize.value
  } catch {
    if (sequence === requestSequence) error.value = tr('sharePublicUnavailable')
  } finally {
    if (sequence === requestSequence) loading.value = false
  }
}
async function changePageSize(size: number) {
  pageSize.value = size
  pageNumber.value = 1
  await loadPage(1)
}
function changeSection(section: typeof activeSection.value) {
  activeSection.value = section
  pageNumber.value = 1
  scheduleFilterReload()
}
function scheduleFilterReload() {
  if (filterTimer) clearTimeout(filterTimer)
  filterTimer = setTimeout(() => loadPage(1), 180)
}
function changeSort() {
  sortNewest.value = !sortNewest.value
  pageNumber.value = 1
  scheduleFilterReload()
}
onMounted(load)
onUnmounted(() => { if (filterTimer) clearTimeout(filterTimer) })
</script>

<template>
  <main class="shared-ledger">
    <div class="shared-backdrop" aria-hidden="true"></div>
    <section v-if="loading && !page" class="share-state card"><div class="state-orbit" aria-hidden="true"></div><p class="eyebrow">SubFlow</p><h1>{{ tr('sharePublicLoading') }}</h1></section>
    <section v-else-if="error" class="share-state card error-state"><div class="state-mark" aria-hidden="true">!</div><p class="eyebrow">SubFlow</p><h1>{{ tr('sharePublicUnavailable') }}</h1><p>{{ tr('sharePublicUnavailableDesc') }}</p><button class="primary" type="button" @click="load">{{ tr('sharePublicRetry') }}</button></section>
    <section v-else-if="page?.requiresPassword" class="unlock-card card"><div class="lock-mark" aria-hidden="true"></div><p class="eyebrow">{{ tr('sharePublicLedger') }}</p><h1>{{ tr('sharePublicProtected') }}</h1><p>{{ tr('sharePublicProtectedDesc') }}</p><form @submit.prevent="unlock"><label><span>{{ tr('password') }}</span><input v-model="password" type="password" required autofocus autocomplete="current-password"></label><button class="primary" :disabled="loading">{{ tr('sharePublicOpen') }}</button></form></section>

    <template v-else-if="page">
      <header class="share-hero"><div class="share-brand"><span aria-hidden="true">SF</span>SubFlow</div><div class="hero-content"><p class="eyebrow">{{ tr('sharePublicLedger') }}</p><h1>{{ page.name }}</h1><div class="hero-meta"><span>{{ rangeLabel(page.rangeLabel) }}</span><span>{{ tr('sharePublicReadOnly') }}</span></div></div><div class="hero-seal" aria-hidden="true">�</div></header>

      <section v-if="page.showSummary && page.summary" class="summary-section" aria-labelledby="share-summary-title"><div class="section-title"><div><p class="eyebrow">{{ tr('sharePublicSummary') }}</p><h2 id="share-summary-title">{{ tr('sharePublicSummary') }}</h2></div><div v-if="totals.length" class="total-stack"><span>{{ tr('sharePublicTotal') }}</span><strong v-for="total in totals" :key="total.currency">{{ money(total.amount, total.currency) }}</strong></div></div><div class="summary-grid"><article class="summary-card"><span class="summary-mark expenses" aria-hidden="true">�</span><strong>{{ page.summary.expenseCount }}</strong><span>{{ tr('shareExpenses') }}</span></article><article class="summary-card"><span class="summary-mark subscriptions" aria-hidden="true">�</span><strong>{{ page.summary.subscriptionCount }}</strong><span>{{ tr('shareSubscriptions') }}</span></article><article v-if="page.summary.settlementCount" class="summary-card"><span class="summary-mark settlements" aria-hidden="true">�</span><strong>{{ page.summary.settlementCount }}</strong><span>{{ tr('shareSettlements') }}</span></article></div></section>

      <section v-if="balances.length" class="balance-summary card" aria-labelledby="share-balances-title"><div class="section-title"><div><p class="eyebrow">{{ tr('shareBalances') }}</p><h2 id="share-balances-title">{{ tr('shareCurrentBalances') }}</h2><p class="section-help">{{ tr('shareCurrentBalancesDesc') }}</p></div></div><div class="balance-list"><article v-for="(item, index) in balances" :key="'balance-' + index + text(item.member)" class="balance-row"><span class="balance-avatar" aria-hidden="true">{{ text(item.member).slice(0, 1).toUpperCase() || '?' }}</span><div class="balance-person"><strong>{{ item.member }}</strong><span>{{ Number(item.amountMinor) >= 0 ? tr('shareBalanceReceivable') : tr('shareBalancePayable') }}</span></div><strong class="balance-amount" :class="{ negative: Number(item.amountMinor) < 0 }">{{ money(item.amountMinor, item.currency || page.currency) }}</strong></article></div></section>

      <section class="records-section card" aria-labelledby="share-records-title"><div class="section-title"><div><p class="eyebrow">{{ tr('sharePublicRecords') }}</p><h2 id="share-records-title">{{ tr('sharePublicRecords') }}</h2><p class="section-help">{{ tr('shareRecordsHelp') }}</p></div></div><div v-if="hasRecords" class="record-toolbar"><div class="section-tabs" role="tablist" :aria-label="tr('shareRecordFilter')"><button v-for="tab in sectionTabs" :key="tab.key" type="button" class="section-tab" :class="{ active: activeSection === tab.key }" @click="changeSection(tab.key)">{{ tab.label }} <span>{{ tab.count }}</span></button></div><div class="record-tools"><input v-model="search" type="search" :placeholder="tr('shareSearchRecords')" :aria-label="tr('shareSearchRecords')" @input="scheduleFilterReload"><button type="button" class="sort-button" @click="changeSort">{{ sortNewest ? tr('shareSortNewest') : tr('shareSortOldest') }}</button></div></div><SharePagination v-if="hasRecords" :page="page.page || pageNumber" :total-pages="page.totalPages || 1" :page-size="pageSize" :page-sizes="pageSizes" :loading="loading" @page="loadPage" @update:page-size="changePageSize" /><div v-if="records.length" class="record-feed"><article v-for="(item, index) in records" :key="recordKey(item, index)" class="record"><div class="record-primary"><div class="record-heading"><span class="record-type" :class="'type-' + item.type">{{ recordTypeLabel(item.type) }}</span><strong>{{ recordTitle(item) }}</strong></div><span class="record-meta"><b v-if="item.category" class="category-chip">{{ item.category }}</b><i v-for="metaIndex in recordMeta(item).length" :key="metaIndex" aria-hidden="true"></i><span v-for="(meta, metaIndex) in recordMeta(item)" :key="'meta-' + metaIndex">{{ meta }}</span></span><small v-if="item.notes">{{ item.notes }}</small></div><strong class="record-amount">{{ money(item.amountMinor, item.currency || page.currency) }}</strong></article></div><div v-else-if="hasRecords" class="record-no-match">{{ tr('shareNoMatches') }}</div><div v-else class="record-empty"><div class="empty-mark" aria-hidden="true"></div><p>{{ tr('sharePublicNoRecords') }}</p></div><SharePagination v-if="hasRecords" :page="page.page || pageNumber" :total-pages="page.totalPages || 1" :page-size="pageSize" :page-sizes="pageSizes" :loading="loading" @page="loadPage" @update:page-size="changePageSize" /></section>
      <footer class="share-footer"><span class="share-brand"><span aria-hidden="true">SF</span>SubFlow</span><span>{{ tr('sharePublicReadOnly') }}</span></footer>
    </template>
  </main>
</template>

<style scoped>
.shared-ledger{position:relative;width:min(1060px,100%);min-height:100vh;margin:0 auto;padding:42px 22px 34px;display:grid;align-content:start;gap:20px}.shared-backdrop{position:fixed;z-index:-1;inset:0;background:radial-gradient(circle at 15% 6%,color-mix(in srgb,var(--brand) 19%,transparent),transparent 30%),radial-gradient(circle at 88% 4%,color-mix(in srgb,var(--success) 11%,transparent),transparent 24%),var(--bg)}.share-hero{position:relative;overflow:hidden;min-height:248px;padding:28px;border:1px solid color-mix(in srgb,var(--brand) 32%,var(--line));border-radius:24px;background:linear-gradient(132deg,color-mix(in srgb,var(--brand) 24%,var(--surface)),var(--surface) 56%,color-mix(in srgb,var(--success) 9%,var(--surface)));box-shadow:var(--shadow-lg)}.share-hero::after{content:"";position:absolute;right:-90px;bottom:-140px;width:360px;height:360px;border:1px solid color-mix(in srgb,var(--brand) 34%,transparent);border-radius:50%;box-shadow:0 0 0 42px color-mix(in srgb,var(--brand) 7%,transparent),0 0 0 86px color-mix(in srgb,var(--brand) 5%,transparent)}.share-brand{display:inline-flex;align-items:center;gap:8px;color:var(--ink);font-size:13px;font-weight:800}.share-brand>span{display:grid;place-items:center;width:24px;height:24px;border-radius:8px;background:var(--brand);color:#fff;font-size:8px;letter-spacing:-.04em}.hero-content{position:absolute;left:28px;right:28px;bottom:29px;z-index:1}.hero-content h1{max-width:720px;margin:5px 0 12px;font-size:clamp(32px,5vw,54px);letter-spacing:-.065em;line-height:1.05}.hero-meta{display:flex;flex-wrap:wrap;gap:8px}.hero-meta span{padding:6px 9px;border:1px solid var(--line);border-radius:999px;background:color-mix(in srgb,var(--surface) 70%,transparent);color:var(--muted);font-size:11px;font-weight:750}.hero-seal{position:absolute;z-index:1;right:34px;top:38px;display:grid;place-items:center;width:48px;height:48px;border-radius:16px;background:var(--surface);color:var(--brand);font-size:24px;box-shadow:var(--shadow)}.summary-section,.balance-summary{display:grid;gap:13px}.section-title{display:flex;align-items:end;justify-content:space-between;gap:18px}.section-title h2{margin:3px 0 0;font-size:20px;letter-spacing:-.03em}.section-help{margin:6px 0 0;color:var(--muted);font-size:14px;line-height:1.5}.total-stack{display:grid;justify-items:end;gap:2px;text-align:right}.total-stack span{color:var(--muted);font-size:10px;font-weight:800;letter-spacing:.08em;text-transform:uppercase}.total-stack strong{color:var(--brand);font-size:14px}.summary-grid{display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:12px}.summary-card{position:relative;display:grid;gap:5px;min-height:132px;padding:19px;border:1px solid var(--line);border-radius:17px;background:var(--surface);box-shadow:var(--shadow);overflow:hidden}.summary-card::after{content:"";position:absolute;right:-22px;bottom:-28px;width:88px;height:88px;border-radius:50%;background:color-mix(in srgb,var(--brand) 7%,transparent)}.summary-mark{display:grid;place-items:center;width:29px;height:29px;border-radius:9px;background:var(--brand-soft);color:var(--brand);font-weight:850}.summary-mark.expenses{background:var(--success-soft);color:var(--success)}.summary-card strong{margin-top:auto;font-size:29px;letter-spacing:-.05em}.summary-card>span:last-child{color:var(--muted);font-size:14px;font-weight:700}.balance-summary{padding:24px 26px}.balance-list{display:grid;max-height:360px;overflow:auto}.balance-row{display:flex;align-items:center;gap:12px;padding:13px 0;border-top:1px solid var(--line)}.balance-avatar{display:grid;place-items:center;flex:0 0 36px;width:36px;height:36px;border-radius:12px;background:var(--brand-soft);color:var(--brand);font-weight:800}.balance-person{display:grid;gap:3px;min-width:0}.balance-person strong{overflow:hidden;text-overflow:ellipsis;white-space:nowrap;font-size:16px}.balance-person span{color:var(--muted);font-size:14px}.balance-amount{margin-left:auto;font-size:16px;white-space:nowrap}.balance-amount.negative{color:var(--danger)}.records-section{padding:0;overflow:hidden}.records-section>.section-title{padding:23px 26px 18px;border-bottom:1px solid var(--line)}.record-toolbar{display:grid;gap:12px;padding:16px 26px;border-bottom:1px solid var(--line);background:var(--surface-soft)}.section-tabs{display:flex;gap:8px;overflow:auto}.section-tab,.sort-button{border:1px solid var(--line);border-radius:10px;background:var(--surface);color:var(--muted);padding:8px 12px;font-size:14px;white-space:nowrap;cursor:pointer}.section-tab.active{border-color:var(--brand);background:var(--brand-soft);color:var(--ink)}.section-tab span{margin-left:4px;color:var(--brand)}.record-tools{display:flex;gap:10px}.record-tools input{min-width:0;flex:1;padding:10px 12px;border:1px solid var(--line);border-radius:10px;background:var(--surface);color:var(--ink);font-size:16px}.record-feed{display:grid}.record{display:flex;align-items:flex-start;justify-content:space-between;gap:18px;padding:16px 26px;border-bottom:1px solid var(--line)}.record-primary{display:grid;gap:5px;min-width:0}.record-heading{display:flex;align-items:center;gap:9px;min-width:0}.record-heading strong{overflow:hidden;text-overflow:ellipsis;white-space:nowrap;font-size:16px}.record-type{flex:0 0 auto;padding:3px 7px;border-radius:999px;background:var(--brand-soft);color:var(--brand);font-size:12px;font-weight:750}.type-expense{background:var(--success-soft);color:var(--success)}.record-meta{display:flex;flex-wrap:wrap;align-items:center;gap:6px;color:var(--muted);font-size:14px;line-height:1.45}.record-meta i{width:3px;height:3px;border-radius:50%;background:var(--line-strong)}.record-primary small{color:color-mix(in srgb,var(--muted) 88%,var(--ink));font-size:14px;white-space:pre-wrap}.record-amount{flex:0 0 auto;padding-top:2px;font-size:16px;white-space:nowrap}.category-chip{display:inline-flex;padding:2px 7px;border-radius:999px;background:var(--brand-soft);color:var(--brand);font-size:12px;font-weight:700}.record-empty,.record-no-match{padding:47px 22px;color:var(--muted);font-size:15px;text-align:center}.record-empty .empty-mark{width:38px;height:38px;margin:0 auto 7px;background:var(--surface-soft);font-size:21px}.record-empty p{margin:0}.share-footer{display:flex;align-items:center;justify-content:space-between;padding:0 5px;color:var(--muted);font-size:14px}.share-footer .share-brand{color:var(--muted);font-size:12px}.share-footer .share-brand>span{width:20px;height:20px;border-radius:6px;font-size:7px}.share-state,.unlock-card{width:min(480px,100%);margin:12vh auto;padding:34px;text-align:center}.share-state h1,.unlock-card h1{margin:7px 0 9px;font-size:27px;letter-spacing:-.045em}.share-state p:not(.eyebrow),.unlock-card>p:not(.eyebrow){margin:0;color:var(--muted);font-size:15px;line-height:1.55}.share-state button{margin-top:18px}.state-orbit{width:42px;height:42px;margin:0 auto 18px;border:3px solid var(--brand-soft);border-top-color:var(--brand);border-radius:50%;animation:share-spin .9s linear infinite}.state-mark,.lock-mark{display:grid;place-items:center;width:46px;height:46px;margin:0 auto 16px;border-radius:15px;background:var(--brand-soft);color:var(--brand);font-size:23px;font-weight:800}.error-state .state-mark{background:var(--danger-soft);color:var(--danger)}.unlock-card{max-width:450px;text-align:left}.unlock-card .lock-mark{margin:0 0 15px}.unlock-card form{display:grid;gap:13px;margin-top:20px}.unlock-card label{display:grid;gap:7px;font-size:14px;font-weight:750}.unlock-card .primary{width:100%}@keyframes share-spin{to{transform:rotate(360deg)}}@media(max-width:680px){.shared-ledger{padding:20px 15px 28px;gap:15px}.share-hero{min-height:224px;padding:21px;border-radius:19px}.hero-content{left:21px;right:21px;bottom:22px}.hero-seal{right:20px;top:20px;width:40px;height:40px;border-radius:13px}.summary-grid{grid-template-columns:1fr}.summary-card{min-height:101px}.section-title{align-items:flex-start;flex-direction:column}.total-stack{justify-items:start;text-align:left}.records-section>.section-title,.record-toolbar{padding-left:18px;padding-right:18px}.record{gap:11px;padding-left:18px;padding-right:18px}.record-heading{align-items:flex-start;flex-direction:column;gap:5px}.record-amount{font-size:14px}.balance-summary{padding:20px 18px}.balance-person strong{font-size:15px}.balance-amount{font-size:14px}.share-state,.unlock-card{margin:8vh auto;padding:25px 20px}.share-footer{padding:0 2px}}
</style>

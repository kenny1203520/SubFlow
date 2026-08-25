<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { pb } from '../pocketbase'
import type { SharePage } from '../api/types'
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
type RecordRow = Record<string, unknown>
type RecordSection = 'all' | 'expenses' | 'subscriptions' | 'settlements' | 'balances'
const activeSection = ref<RecordSection>('all')
const search = ref("")
const sortNewest = ref(true)
const pageNumber = ref(1)
const pageSize = ref(25)
const pageSizes = [5, 10, 15, 25]
const totals = computed(() => Object.entries(page.value?.summary?.expenseTotals || {}).map(([currency, amount]) => ({ currency, amount })))
const hasRecords = computed(() => Boolean(page.value?.expenses?.length || page.value?.subscriptions?.length || page.value?.settlements?.length || page.value?.balances?.length))
const totalRows = computed(() => (page.value?.expenses?.length || 0) + (page.value?.subscriptions?.length || 0) + (page.value?.settlements?.length || 0) + (page.value?.balances?.length || 0))
const sectionTabs = computed(() => [{ key: 'all', label: tr('shareAllRecords'), count: totalRows.value }, ...(page.value?.expenses?.length ? [{ key: 'expenses', label: tr('shareExpenses'), count: page.value.expenses.length }] : []), ...(page.value?.subscriptions?.length ? [{ key: 'subscriptions', label: tr('shareSubscriptions'), count: page.value.subscriptions.length }] : []), ...(page.value?.settlements?.length ? [{ key: 'settlements', label: tr('shareSettlements'), count: page.value.settlements.length }] : []), ...(page.value?.balances?.length ? [{ key: 'balances', label: tr('shareBalances'), count: page.value.balances.length }] : [])] as Array<{key:RecordSection;label:string;count:number}>)
function rowDate(row: RecordRow, section: RecordSection) { return String(row[section === 'expenses' ? 'incurredOn' : section === 'subscriptions' ? 'nextBilling' : 'settledOn'] || '') }
function rowSearchText(row: RecordRow) { return ['title','name','category','paidBy','from','to','status','billingCycle','member'].map(key => String(row[key] || '')).join(' ').toLocaleLowerCase(locale.value) }
function filteredRows(rows: Array<RecordRow> | undefined, section: RecordSection) { const query=search.value.trim().toLocaleLowerCase(locale.value); return [...(rows || [])].filter(row => !query || rowSearchText(row).includes(query)).sort((a,b) => { const l=Date.parse(rowDate(a,section)) || 0; const r=Date.parse(rowDate(b,section)) || 0; return sortNewest.value ? r-l : l-r }) }
const visibleExpenses = computed(() => filteredRows(page.value?.expenses, 'expenses')); const visibleSubscriptions = computed(() => filteredRows(page.value?.subscriptions, 'subscriptions')); const visibleSettlements = computed(() => filteredRows(page.value?.settlements, 'settlements')); const visibleBalances = computed(() => filteredRows(page.value?.balances, 'balances')); const visibleCount = computed(() => visibleExpenses.value.length + visibleSubscriptions.value.length + visibleSettlements.value.length + visibleBalances.value.length)
function sectionVisible(section: RecordSection) { return activeSection.value === 'all' || activeSection.value === section }

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
  loading.value = true
  error.value = ''
  try {
    const query = '?page=1&perPage=' + pageSize.value
    const result = await request('/shares/' + token + query)
    if (result.requiresLogin) {
      if (!pb.authStore.isValid) { await router.push({ name: 'auth', query: { redirect: route.fullPath } }); return }
      accountAccess.value = true
      page.value = await request('/shares/' + token + '/account' + query)
    } else page.value = result
    pageNumber.value = 1
  } catch { error.value = tr('sharePublicUnavailable') }
  finally { loading.value = false }
}
async function unlock() {
  loading.value = true
  error.value = ''
  try {
    page.value = await request('/shares/' + token + '/access?page=1&perPage=' + pageSize.value, { method: 'POST', body: JSON.stringify({ password: password.value }) })
    pageNumber.value = 1
  } catch { error.value = tr('sharePublicUnavailable') }
  finally { loading.value = false }
}
async function loadPage(target: number) {
  const max = page.value?.totalPages || 1
  if (target < 1 || target > max) return
  loading.value = true
  error.value = ''
  try {
    const base = accountAccess.value ? '/shares/' + token + '/account' : '/shares/' + token
    page.value = await request(base + '?page=' + target + '&perPage=' + pageSize.value)
    pageNumber.value = target
  } catch { error.value = tr('sharePublicUnavailable') }
  finally { loading.value = false }
}
async function changePageSize() {
  pageNumber.value = 1
  await loadPage(1)
}
function money(amount: unknown, currency: unknown) { return new Intl.NumberFormat(locale.value, { style: 'currency', currency: String(currency || 'TWD') }).format(Number(amount || 0) / 100) }
function date(value: unknown) { return value ? formatDate(String(value)) : '—' }
function rangeLabel(value: unknown) { const raw=String(value || ''); if (raw === 'all') return tr('shareRangeAll'); if (raw === 'rolling') return tr('shareRangeRolling'); if (raw === 'fixed') return tr('shareRangeFixed'); return raw }
function cycleLabel(value: unknown) { const key=String(value || ''); return ({ monthly: tr('monthly'), quarterly: tr('quarterly'), yearly: tr('yearly') } as Record<string,string>)[key] || key }
function statusLabel(value: unknown) { const key=String(value || ''); return ({ active: tr('active'), paused: tr('paused'), cancelled: tr('cancelled'), ending: tr('ending'), ended: tr('ended') } as Record<string,string>)[key] || key }
onMounted(load)
</script>

<template>
  <main class="shared-ledger">
    <div class="shared-backdrop" aria-hidden="true"></div>
    <section v-if="loading && !page" class="share-state card"><div class="state-orbit" aria-hidden="true"></div><p class="eyebrow">SubFlow</p><h1>{{ tr('sharePublicLoading') }}</h1></section>
    <section v-else-if="error" class="share-state card error-state"><div class="state-mark" aria-hidden="true">!</div><p class="eyebrow">SubFlow</p><h1>{{ tr('sharePublicUnavailable') }}</h1><p>{{ tr('sharePublicUnavailableDesc') }}</p></section>
    <section v-else-if="page?.requiresPassword" class="unlock-card card"><div class="lock-mark" aria-hidden="true">⌁</div><p class="eyebrow">{{ tr('sharePublicLedger') }}</p><h1>{{ tr('sharePublicProtected') }}</h1><p>{{ tr('sharePublicProtectedDesc') }}</p><form @submit.prevent="unlock"><label><span>{{ tr('password') }}</span><input v-model="password" type="password" required autofocus autocomplete="current-password"></label><button class="primary" :disabled="loading">{{ tr('sharePublicOpen') }}</button></form></section>

    <template v-else-if="page">
      <header class="share-hero"><div class="share-brand"><span aria-hidden="true">SF</span>SubFlow</div><div class="hero-content"><p class="eyebrow">{{ tr('sharePublicLedger') }}</p><h1>{{ page.name }}</h1><div class="hero-meta"><span>{{ rangeLabel(page.rangeLabel) }}</span><span>{{ tr('sharePublicReadOnly') }}</span></div></div><div class="hero-seal" aria-hidden="true">↗</div></header>

      <section v-if="page.showSummary && page.summary" class="summary-section" aria-labelledby="share-summary-title"><div class="section-title"><div><p class="eyebrow">{{ tr('sharePublicSummary') }}</p><h2 id="share-summary-title">{{ tr('sharePublicSummary') }}</h2></div><div v-if="totals.length" class="total-stack"><span>{{ tr('sharePublicTotal') }}</span><strong v-for="total in totals" :key="total.currency">{{ money(total.amount, total.currency) }}</strong></div></div><div class="summary-grid"><article class="summary-card"><span class="summary-mark expenses" aria-hidden="true">↙</span><strong>{{ page.summary.expenseCount }}</strong><span>{{ tr('shareExpenses') }}</span></article><article class="summary-card"><span class="summary-mark subscriptions" aria-hidden="true">⟳</span><strong>{{ page.summary.subscriptionCount }}</strong><span>{{ tr('shareSubscriptions') }}</span></article><article v-if="page.summary.settlementCount" class="summary-card"><span class="summary-mark settlements" aria-hidden="true">⇄</span><strong>{{ page.summary.settlementCount }}</strong><span>{{ tr('shareSettlements') }}</span></article></div></section>

      <section class="records-section card" aria-labelledby="share-records-title"><div class="section-title"><div><p class="eyebrow">{{ tr('sharePublicRecords') }}</p><h2 id="share-records-title">{{ tr('sharePublicRecords') }}</h2></div></div><div v-if="hasRecords" class="record-toolbar"><div class="section-tabs" role="tablist" :aria-label="tr('shareRecordFilter')"><button v-for="tab in sectionTabs" :key="tab.key" type="button" class="section-tab" :class="{ active: activeSection === tab.key }" @click="activeSection = tab.key">{{ tab.label }} <span>{{ tab.count }}</span></button></div><div class="record-tools"><input v-model="search" type="search" :placeholder="tr('shareSearchRecords')" :aria-label="tr('shareSearchRecords')"><button type="button" class="sort-button" @click="sortNewest = !sortNewest">{{ sortNewest ? tr('shareSortNewest') : tr('shareSortOldest') }}</button></div></div><div v-if="hasRecords" class="record-groups"><div v-if="sectionVisible('expenses') && visibleExpenses.length" class="record-group"><h3><span aria-hidden="true">↙</span>{{ tr('shareExpenses') }}</h3><article v-for="item in visibleExpenses" :key="String(item.title)+String(item.incurredOn)" class="record"><div class="record-primary"><strong>{{ item.title }}</strong><span><b v-if="item.category" class="category-chip">{{ item.category }}</b><i v-if="item.category" aria-hidden="true"></i>{{ date(item.incurredOn) }}</span><small v-if="item.paidBy">{{ tr('sharePublicPaidBy', { value: String(item.paidBy) }) }}</small><small v-if="item.notes">{{ item.notes }}</small></div><strong class="record-amount">{{ money(item.amountMinor, item.currency) }}</strong></article></div><div v-if="sectionVisible('subscriptions') && visibleSubscriptions.length" class="record-group"><h3><span aria-hidden="true">⟳</span>{{ tr('shareSubscriptions') }}</h3><article v-for="item in visibleSubscriptions" :key="String(item.name)+String(item.nextBilling)" class="record"><div class="record-primary"><strong>{{ item.name }}</strong><span><b v-if="item.category" class="category-chip">{{ item.category }}</b><i v-if="item.category" aria-hidden="true"></i>{{ cycleLabel(item.billingCycle) }}<i aria-hidden="true"></i>{{ statusLabel(item.status) }}</span><small v-if="item.paidBy">{{ tr('sharePublicPaidBy', { value: String(item.paidBy) }) }}</small><small v-if="item.notes">{{ item.notes }}</small></div><strong class="record-amount">{{ money(item.amountMinor, item.currency) }}</strong></article></div><div v-if="sectionVisible('settlements') && visibleSettlements.length" class="record-group"><h3><span aria-hidden="true">⇄</span>{{ tr('shareSettlements') }}</h3><article v-for="item in visibleSettlements" :key="String(item.settledOn)+String(item.amountMinor)" class="record"><div class="record-primary"><strong v-if="item.from">{{ tr('sharePublicFromTo', { from: String(item.from), to: String(item.to) }) }}</strong><span>{{ date(item.settledOn) }}</span><small v-if="item.notes">{{ item.notes }}</small></div><strong class="record-amount">{{ money(item.amountMinor, item.currency) }}</strong></article></div><div v-if="sectionVisible('balances') && visibleBalances.length" class="record-group balances-group"><h3><span aria-hidden="true">⇄</span>{{ tr('shareBalances') }}</h3><article v-for="(item,index) in visibleBalances" :key="'balance-' + index + String(item.member)" class="record"><div class="record-primary"><strong>{{ item.member }}</strong><small>{{ Number(item.amountMinor) >= 0 ? tr('shareBalanceReceivable') : tr('shareBalancePayable') }}</small></div><strong class="record-amount">{{ money(item.amountMinor, item.currency) }}</strong></article></div></div><div v-if="hasRecords && !visibleCount" class="record-no-match">{{ tr('shareNoMatches') }}</div><div v-else-if="!hasRecords" class="record-empty"><div class="empty-mark" aria-hidden="true">–</div><p>{{ tr('sharePublicNoRecords') }}</p></div><div v-if="hasRecords" class="pagination-bar">
  <label class="page-size-control"><span>{{ tr('pageSize') }}</span><select v-model.number="pageSize" @change="changePageSize"><option v-for="size in pageSizes" :key="size" :value="size">{{ tr('pageSizeCount', { count: size }) }}</option></select></label>
  <nav v-if="(page.totalPages || 1) > 1" class="page-navigation" :aria-label="tr('pagination')">
    <button type="button" class="page-button" :disabled="loading || pageNumber === 1" @click="loadPage(pageNumber - 1)">{{ tr('previousPage') }}</button>
    <button v-for="number in (page.totalPages || 1)" :key="number" type="button" class="page-button page-number" :class="{ active: pageNumber === number }" :aria-current="pageNumber === number ? 'page' : undefined" :disabled="loading" @click="loadPage(number)">{{ number }}</button>
    <button type="button" class="page-button" :disabled="loading || pageNumber === (page.totalPages || 1)" @click="loadPage(pageNumber + 1)">{{ tr('nextPage') }}</button>
  </nav>
</div></section>
      <footer class="share-footer"><span class="share-brand"><span aria-hidden="true">SF</span>SubFlow</span><span>{{ tr('sharePublicReadOnly') }}</span></footer>
    </template>
  </main>
</template>

<style scoped>
.shared-ledger{position:relative;width:min(1060px,100%);min-height:100vh;margin:0 auto;padding:42px 22px 34px;display:grid;align-content:start;gap:20px}.shared-backdrop{position:fixed;z-index:-1;inset:0;background:radial-gradient(circle at 15% 6%,color-mix(in srgb,var(--brand) 19%,transparent),transparent 30%),radial-gradient(circle at 88% 4%,color-mix(in srgb,var(--success) 11%,transparent),transparent 24%),var(--bg)}.share-hero{position:relative;overflow:hidden;min-height:248px;padding:28px;border:1px solid color-mix(in srgb,var(--brand) 32%,var(--line));border-radius:24px;background:linear-gradient(132deg,color-mix(in srgb,var(--brand) 24%,var(--surface)),var(--surface) 56%,color-mix(in srgb,var(--success) 9%,var(--surface)));box-shadow:var(--shadow-lg)}.share-hero::after{content:"";position:absolute;right:-90px;bottom:-140px;width:360px;height:360px;border:1px solid color-mix(in srgb,var(--brand) 34%,transparent);border-radius:50%;box-shadow:0 0 0 42px color-mix(in srgb,var(--brand) 7%,transparent),0 0 0 86px color-mix(in srgb,var(--brand) 5%,transparent)}.share-brand{display:inline-flex;align-items:center;gap:8px;color:var(--ink);font-size:13px;font-weight:800}.share-brand>span{display:grid;place-items:center;width:24px;height:24px;border-radius:8px;background:var(--brand);color:#fff;font-size:8px;letter-spacing:-.04em}.hero-content{position:absolute;left:28px;right:28px;bottom:29px;z-index:1}.hero-content h1{max-width:720px;margin:5px 0 12px;font-size:clamp(32px,5vw,54px);letter-spacing:-.065em;line-height:1.05}.hero-meta{display:flex;flex-wrap:wrap;gap:8px}.hero-meta span{padding:6px 9px;border:1px solid var(--line);border-radius:999px;background:color-mix(in srgb,var(--surface) 70%,transparent);color:var(--muted);font-size:11px;font-weight:750}.hero-seal{position:absolute;z-index:1;right:34px;top:38px;display:grid;place-items:center;width:48px;height:48px;border-radius:16px;background:var(--surface);color:var(--brand);font-size:24px;box-shadow:var(--shadow)}.summary-section{display:grid;gap:13px}.section-title{display:flex;align-items:end;justify-content:space-between;gap:18px}.section-title h2{margin:3px 0 0;font-size:20px;letter-spacing:-.03em}.total-stack{display:grid;justify-items:end;gap:2px;text-align:right}.total-stack span{color:var(--muted);font-size:10px;font-weight:800;letter-spacing:.08em;text-transform:uppercase}.total-stack strong{color:var(--brand);font-size:14px}.summary-grid{display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:12px}.summary-card{position:relative;display:grid;gap:5px;min-height:132px;padding:19px;border:1px solid var(--line);border-radius:17px;background:var(--surface);box-shadow:var(--shadow);overflow:hidden}.summary-card::after{content:"";position:absolute;right:-22px;bottom:-28px;width:88px;height:88px;border-radius:50%;background:color-mix(in srgb,var(--brand) 7%,transparent)}.summary-mark{display:grid;place-items:center;width:29px;height:29px;border-radius:9px;background:var(--brand-soft);color:var(--brand);font-weight:850}.summary-mark.expenses{background:var(--success-soft);color:var(--success)}.summary-mark.settlements{background:color-mix(in srgb,var(--brand) 14%,var(--surface));color:var(--brand)}.summary-card strong{margin-top:auto;font-size:29px;letter-spacing:-.05em}.summary-card>span:last-child{color:var(--muted);font-size:12px;font-weight:700}.records-section{padding:0;overflow:hidden}.records-section>.section-title{padding:23px 26px 18px;border-bottom:1px solid var(--line)}.record-groups{display:grid}.record-group{padding:19px 26px 4px;border-bottom:1px solid var(--line)}.record-group:last-child{border-bottom:0}.record-group h3{display:flex;align-items:center;gap:7px;margin:0 0 4px;color:var(--ink);font-size:13px}.record-group h3 span{display:grid;place-items:center;width:23px;height:23px;border-radius:7px;background:var(--brand-soft);color:var(--brand);font-size:13px}.record{display:flex;align-items:flex-start;justify-content:space-between;gap:18px;padding:14px 0;border-top:1px solid var(--line)}.record-primary{display:grid;gap:3px;min-width:0}.record-primary strong{overflow:hidden;text-overflow:ellipsis;white-space:nowrap;font-size:14px}.record-primary span,.record-primary small{color:var(--muted);font-size:11px;line-height:1.45}.record-primary small{color:color-mix(in srgb,var(--muted) 88%,var(--ink));white-space:pre-wrap}.record-primary i{display:inline-block;width:3px;height:3px;margin:0 7px 2px;border-radius:50%;background:var(--line-strong)}.record-amount{flex:0 0 auto;padding-top:2px;font-size:13px;white-space:nowrap}.record-empty{display:grid;justify-items:center;gap:6px;padding:47px 22px;color:var(--muted);font-size:13px;text-align:center}.record-empty .empty-mark{width:38px;height:38px;margin:0;background:var(--surface-soft);font-size:21px}.record-empty p{margin:0}.load-more{display:block;width:100%;padding:15px;border-top:1px solid var(--line);border-radius:0}.share-footer{display:flex;align-items:center;justify-content:space-between;padding:0 5px;color:var(--muted);font-size:11px}.share-footer .share-brand{color:var(--muted);font-size:11px}.share-footer .share-brand>span{width:20px;height:20px;border-radius:6px;font-size:7px}.share-state,.unlock-card{width:min(480px,100%);margin:12vh auto;padding:34px;text-align:center}.share-state h1,.unlock-card h1{margin:7px 0 9px;font-size:27px;letter-spacing:-.045em}.share-state p:not(.eyebrow),.unlock-card>p:not(.eyebrow){margin:0;color:var(--muted);font-size:13px;line-height:1.55}.state-orbit{width:42px;height:42px;margin:0 auto 18px;border:3px solid var(--brand-soft);border-top-color:var(--brand);border-radius:50%;animation:share-spin .9s linear infinite}.state-mark,.lock-mark{display:grid;place-items:center;width:46px;height:46px;margin:0 auto 16px;border-radius:15px;background:var(--brand-soft);color:var(--brand);font-size:23px;font-weight:800}.error-state .state-mark{background:var(--danger-soft);color:var(--danger)}.unlock-card{max-width:450px;text-align:left}.unlock-card .lock-mark{margin:0 0 15px}.unlock-card form{display:grid;gap:13px;margin-top:20px}.unlock-card label{display:grid;gap:7px;font-size:12px;font-weight:750}.unlock-card .primary{width:100%}@keyframes share-spin{to{transform:rotate(360deg)}}@media(max-width:680px){.shared-ledger{padding:20px 15px 28px;gap:15px}.share-hero{min-height:224px;padding:21px;border-radius:19px}.hero-content{left:21px;right:21px;bottom:22px}.hero-seal{right:20px;top:20px;width:40px;height:40px;border-radius:13px}.summary-grid{grid-template-columns:1fr}.summary-card{min-height:101px}.section-title{align-items:flex-start;flex-direction:column}.total-stack{justify-items:start;text-align:left}.records-section>.section-title,.record-group{padding-left:18px;padding-right:18px}.record{gap:11px}.record-amount{font-size:12px}.share-state,.unlock-card{margin:8vh auto;padding:25px 20px}.share-footer{padding:0 2px}}
.record-toolbar{display:grid;gap:12px;padding:16px 26px;border-bottom:1px solid var(--line);background:var(--surface-soft)}.section-tabs{display:flex;gap:8px;overflow:auto}.section-tab,.sort-button{border:1px solid var(--line);border-radius:10px;background:var(--surface);color:var(--muted);padding:8px 12px;font-size:14px;white-space:nowrap;cursor:pointer}.section-tab.active{border-color:var(--brand);background:var(--brand-soft);color:var(--ink)}.section-tab span{margin-left:4px;color:var(--brand)}.record-tools{display:flex;gap:10px}.record-tools input{min-width:0;flex:1;padding:10px 12px;border:1px solid var(--line);border-radius:10px;background:var(--surface);color:var(--ink);font-size:16px}.record-primary span{display:flex;flex-wrap:wrap;align-items:center;gap:6px;font-size:14px}.record-primary small{font-size:13px}.record-primary strong{font-size:16px}.record-amount{font-size:15px}.category-chip{display:inline-flex;padding:2px 7px;border-radius:999px;background:var(--brand-soft);color:var(--brand);font-size:12px;font-weight:700}.record-no-match{padding:24px;text-align:center;color:var(--muted);font-size:15px}@media(max-width:680px){.record-toolbar{padding:14px 18px}.record-tools{flex-direction:column}.record-tools input,.sort-button{width:100%}.section-tab{font-size:14px}.record-primary strong{font-size:15px}.record-primary span,.record-primary small{font-size:14px}.record-amount{font-size:14px}}.pagination-bar{display:flex;align-items:center;justify-content:space-between;gap:14px;padding:14px 20px;border-top:1px solid var(--line);background:var(--surface-soft)}.page-size-control{display:flex;align-items:center;gap:8px;color:var(--muted);font-size:14px}.page-size-control select{padding:8px 28px 8px 10px;border:1px solid var(--line);border-radius:9px;background:var(--surface);color:var(--ink);font-size:14px}.page-navigation{display:flex;align-items:center;gap:6px;flex-wrap:wrap;justify-content:flex-end}.page-button{min-width:34px;padding:8px 10px;border:1px solid var(--line);border-radius:9px;background:var(--surface);color:var(--muted);font-size:14px;cursor:pointer}.page-button:hover:not(:disabled),.page-button.active{border-color:var(--brand);background:var(--brand-soft);color:var(--ink)}.page-button:disabled{cursor:not-allowed;opacity:.5}.page-number{min-width:34px}@media(max-width:680px){.pagination-bar{align-items:stretch;flex-direction:column;padding:14px 18px}.page-navigation{justify-content:space-between}.page-button{flex:1}.page-number{flex:0 0 34px}}</style>

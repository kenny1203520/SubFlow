<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { pb } from '../pocketbase'
import type { SharePage } from '../api/types'

const route = useRoute()
const router = useRouter()
const token = String(route.params.token || '')
const page = ref<SharePage>()
const password = ref('')
const loading = ref(true)
const error = ref('')
const accountAccess = ref(false)

async function request(path: string, options: RequestInit = {}) {
  const headers = new Headers(options.headers)
  headers.set('Accept', 'application/json')
  if (pb.authStore.token) headers.set('Authorization', `Bearer ${pb.authStore.token}`)
  if (options.body) headers.set('Content-Type', 'application/json')
  const response = await fetch(`/api/subflow/v1${path}`, { ...options, headers })
  const body = await response.json()
  if (!response.ok) throw new Error(body?.error?.message || 'This share page is unavailable')
  return body.data as SharePage
}

async function load() {
  loading.value = true; error.value = ''
  try {
    const result = await request(`/shares/${token}`)
    if (result.requiresLogin) {
      if (!pb.authStore.isValid) { await router.push({ name: 'auth', query: { redirect: route.fullPath } }); return }
      accountAccess.value = true
      page.value = await request(`/shares/${token}/account`)
    } else page.value = result
  } catch (reason) { error.value = reason instanceof Error ? reason.message : 'This share page is unavailable' } finally { loading.value = false }
}
async function unlock() {
  loading.value = true; error.value = ''
  try { page.value = await request(`/shares/${token}/access`, { method: 'POST', body: JSON.stringify({ password: password.value }) }) }
  catch { error.value = 'This share page is unavailable' } finally { loading.value = false }
}
async function loadMore() {
  if (!page.value?.nextPage) return
  loading.value = true
  try {
    const nextNumber = page.value.nextPage
    const next = await request(accountAccess.value ? `/shares/${token}/account?page=${nextNumber}` : `/shares/${token}?page=${nextNumber}`)
    page.value = { ...next, expenses: [...(page.value.expenses || []), ...(next.expenses || [])], subscriptions: [...(page.value.subscriptions || []), ...(next.subscriptions || [])], settlements: [...(page.value.settlements || []), ...(next.settlements || [])] }
  } catch (reason) { error.value = reason instanceof Error ? reason.message : 'This share page is unavailable' } finally { loading.value = false }
}
function money(amount: unknown, currency: unknown) { return new Intl.NumberFormat(undefined, { style: 'currency', currency: String(currency || 'TWD') }).format(Number(amount || 0) / 100) }
onMounted(load)
</script>

<template>
  <main class="share-page">
    <section v-if="loading && !page" class="card">Loading shared ledger…</section>
    <section v-else-if="error" class="card error-state"><h1>Share page unavailable</h1><p>{{ error }}</p></section>
    <section v-else-if="page?.requiresPassword" class="card unlock"><h1>Protected share</h1><p>Enter the password supplied by the owner to view this ledger.</p><form @submit.prevent="unlock"><input v-model="password" type="password" required autofocus><button class="primary">View ledger</button></form></section>
    <template v-else-if="page">
      <header><small>SubFlow · shared ledger</small><h1>{{ page.name }}</h1><p>{{ page.rangeLabel }}</p></header>
      <section v-if="page.showSummary && page.summary" class="summary"><article class="card"><strong>{{ page.summary.expenseCount }}</strong><span>Expenses</span></article><article class="card"><strong>{{ page.summary.subscriptionCount }}</strong><span>Subscriptions</span></article><article class="card"><strong>{{ page.summary.settlementCount }}</strong><span>Settlements</span></article></section>
      <section v-if="page.expenses?.length" class="card"><h2>Expenses</h2><article v-for="item in page.expenses" :key="String(item.title)+String(item.incurredOn)" class="record"><div><strong>{{ item.title }}</strong><small>{{ item.category }} · {{ new Date(String(item.incurredOn)).toLocaleDateString() }}</small><small v-if="item.paidBy">Paid by {{ item.paidBy }}</small><small v-if="item.notes">{{ item.notes }}</small></div><strong>{{ money(item.amountMinor, item.currency) }}</strong></article></section>
      <section v-if="page.subscriptions?.length" class="card"><h2>Subscriptions</h2><article v-for="item in page.subscriptions" :key="String(item.name)+String(item.nextBilling)" class="record"><div><strong>{{ item.name }}</strong><small>{{ item.category }} · {{ item.billingCycle }} · {{ item.status }}</small><small v-if="item.paidBy">Paid by {{ item.paidBy }}</small><small v-if="item.notes">{{ item.notes }}</small></div><strong>{{ money(item.amountMinor, item.currency) }}</strong></article></section>
      <section v-if="page.settlements?.length" class="card"><h2>Settlements</h2><article v-for="item in page.settlements" :key="String(item.settledOn)+String(item.amountMinor)" class="record"><div><strong v-if="item.from">{{ item.from }} → {{ item.to }}</strong><small>{{ new Date(String(item.settledOn)).toLocaleDateString() }}</small><small v-if="item.notes">{{ item.notes }}</small></div><strong>{{ money(item.amountMinor, item.currency) }}</strong></article></section>
      <button v-if="page.nextPage" class="ghost wide" :disabled="loading" @click="loadMore">Load more</button>
    </template>
  </main>
</template>

<style scoped>
.share-page{width:min(980px,100%);margin:0 auto;padding:48px 20px;display:grid;gap:20px}.share-page header{padding:16px 4px}.share-page h1{margin:6px 0;font-size:38px}.share-page header p,.record small{display:block;color:var(--muted)}.summary{display:grid;grid-template-columns:repeat(3,1fr);gap:14px}.summary article{display:grid;gap:5px}.summary strong{font-size:28px}.record{display:flex;justify-content:space-between;gap:18px;padding:14px 0;border-top:1px solid var(--line)}.unlock{max-width:460px;margin:12vh auto}.unlock form{display:flex;gap:8px;margin-top:18px}@media(max-width:600px){.summary{grid-template-columns:1fr}.unlock form{flex-direction:column}}
</style>

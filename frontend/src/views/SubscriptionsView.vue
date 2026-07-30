<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useWorkspaceStore } from '../stores/workspace'
import MoneyValue from '../components/MoneyValue.vue'
import EmptyState from '../components/EmptyState.vue'
import type { BillingCycle, Subscription, SubscriptionStatus } from '../api/types'
import { majorToMinor, minorToInput } from '../api/money'

const workspace = useWorkspaceStore()
const editingId = ref('')
const form = reactive({ name: '', category: '影音', amount: '', billingCycle: 'monthly' as BillingCycle, nextBilling: new Date().toISOString().slice(0, 10), status: 'active' as SubscriptionStatus, notes: '' })

function reset() {
  editingId.value = ''
  Object.assign(form, { name: '', category: '影音', amount: '', billingCycle: 'monthly', nextBilling: new Date().toISOString().slice(0, 10), status: 'active', notes: '' })
}

function edit(item: Subscription) {
  editingId.value = item.id
  Object.assign(form, { name: item.name, category: item.category, amount: minorToInput(item.amountMinor, item.currency), billingCycle: item.billingCycle, nextBilling: item.nextBilling.slice(0, 10), status: item.status, notes: item.notes })
}

async function submit() {
  const group = workspace.currentGroup
  if (!group) return
  const input = { name: form.name, category: form.category, amountMinor: majorToMinor(form.amount, group.currency), currency: group.currency, billingCycle: form.billingCycle, nextBilling: new Date(form.nextBilling).toISOString(), status: form.status, notes: form.notes }
  if (editingId.value) await workspace.updateSubscription(editingId.value, input)
  else await workspace.addSubscription(input)
  reset()
}

async function remove(item: Subscription) {
  if (!confirm(`確定刪除「${item.name}」？`)) return
  await workspace.deleteSubscription(item.id)
  if (editingId.value === item.id) reset()
}
</script>

<template>
  <section class="page">
    <div class="page-heading"><div><p class="eyebrow">RECURRING</p><h1>訂閱管理</h1><p>管理固定扣款的金額、週期、狀態與下次扣款日。</p></div></div>
    <div class="two-column content-heavy">
      <div class="card">
        <div class="card-title"><h2>所有訂閱</h2><span>{{ workspace.subscriptions.length }} 筆</span></div>
        <div v-if="workspace.subscriptions.length" class="rows">
          <div v-for="item in workspace.subscriptions" :key="item.id" class="row">
            <div class="service-icon">{{ item.name.slice(0, 1) }}</div>
            <div class="grow"><strong>{{ item.name }}</strong><small>{{ item.category || '未分類' }} · 下次 {{ new Date(item.nextBilling).toLocaleDateString('zh-TW') }} · {{ item.status }}</small></div>
            <div class="money"><MoneyValue :amount="item.amountMinor" :currency="item.currency"/><small>/ {{ item.billingCycle === 'monthly' ? '月' : item.billingCycle === 'quarterly' ? '季' : '年' }}</small></div>
            <button class="icon-button edit-button" aria-label="編輯" @click="edit(item)">✎</button>
            <button class="icon-button" aria-label="刪除" @click="remove(item)">×</button>
          </div>
        </div>
        <EmptyState v-else title="還沒有訂閱" description="加入第一筆固定扣款。" />
      </div>
      <form class="card form-card sticky" @submit.prevent="submit">
        <h2>{{ editingId ? '編輯訂閱' : '新增訂閱' }}</h2>
        <label>名稱<input v-model="form.name" required placeholder="例如 Netflix"></label>
        <div class="form-row"><label>分類<input v-model="form.category"></label><label>金額<input v-model="form.amount" type="number" min="0" step="0.01" required></label></div>
        <div class="form-row"><label>週期<select v-model="form.billingCycle"><option value="monthly">每月</option><option value="quarterly">每季</option><option value="yearly">每年</option></select></label><label>狀態<select v-model="form.status"><option value="active">啟用</option><option value="paused">暫停</option><option value="cancelled">取消</option></select></label></div>
        <label>下次扣款<input v-model="form.nextBilling" type="date" required></label>
        <label>備註<textarea v-model="form.notes" rows="2"></textarea></label>
        <div class="form-actions"><button class="primary" :disabled="workspace.loading">{{ editingId ? '儲存變更' : '新增訂閱' }}</button><button v-if="editingId" type="button" class="ghost" @click="reset">取消</button></div>
      </form>
    </div>
  </section>
</template>

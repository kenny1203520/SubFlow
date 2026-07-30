<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useAuthStore } from '../stores/auth'
import { useWorkspaceStore } from '../stores/workspace'
import MoneyValue from '../components/MoneyValue.vue'
import EmptyState from '../components/EmptyState.vue'
import type { Expense } from '../api/types'
import { majorToMinor, minorToInput } from '../api/money'

const workspace = useWorkspaceStore()
const auth = useAuthStore()
const editingId = ref('')
const form = reactive({ title: '', category: '餐飲', amount: '', paidBy: '', incurredOn: new Date().toISOString().slice(0, 10), notes: '' })

function reset() {
  editingId.value = ''
  Object.assign(form, { title: '', category: '餐飲', amount: '', paidBy: '', incurredOn: new Date().toISOString().slice(0, 10), notes: '' })
}

function edit(item: Expense) {
  editingId.value = item.id
  Object.assign(form, { title: item.title, category: item.category, amount: minorToInput(item.amountMinor, workspace.currentGroup?.currency), paidBy: item.paidBy, incurredOn: item.incurredOn.slice(0, 10), notes: item.notes })
}

async function submit() {
  const input = { title: form.title, category: form.category, amountMinor: majorToMinor(form.amount, workspace.currentGroup?.currency), paidBy: form.paidBy || auth.record?.id || '', incurredOn: new Date(form.incurredOn).toISOString(), notes: form.notes }
  if (editingId.value) await workspace.updateExpense(editingId.value, input)
  else await workspace.addExpense(input)
  reset()
}

async function remove(item: Expense) {
  if (!confirm(`確定刪除「${item.title}」？`)) return
  await workspace.deleteExpense(item.id)
  if (editingId.value === item.id) reset()
}
</script>

<template>
  <section class="page">
    <div class="page-heading"><div><p class="eyebrow">SPENDING</p><h1>共同支出</h1><p>記錄誰先付款，讓群組的每筆花費保持透明。</p></div></div>
    <div class="two-column content-heavy">
      <div class="card">
        <div class="card-title"><h2>近期支出</h2><span>{{ workspace.expenses.length }} 筆</span></div>
        <div v-if="workspace.expenses.length" class="rows">
          <div v-for="item in workspace.expenses" :key="item.id" class="row">
            <div class="service-icon expense">{{ item.title.slice(0, 1) }}</div>
            <div class="grow"><strong>{{ item.title }}</strong><small>{{ item.category || '未分類' }} · {{ new Date(item.incurredOn).toLocaleDateString('zh-TW') }}</small></div>
            <MoneyValue :amount="item.amountMinor" :currency="workspace.currentGroup?.currency"/>
            <button class="icon-button edit-button" aria-label="編輯" @click="edit(item)">✎</button>
            <button class="icon-button" aria-label="刪除" @click="remove(item)">×</button>
          </div>
        </div>
        <EmptyState v-else title="尚無共同支出" description="新增一筆餐費、交通或採買紀錄。" />
      </div>
      <form class="card form-card sticky" @submit.prevent="submit">
        <h2>{{ editingId ? '編輯支出' : '新增支出' }}</h2>
        <label>項目<input v-model="form.title" required placeholder="例如 週末晚餐"></label>
        <div class="form-row"><label>分類<input v-model="form.category"></label><label>金額<input v-model="form.amount" type="number" min="0" step="0.01" required></label></div>
        <div class="form-row"><label>付款人<select v-model="form.paidBy"><option value="">我自己</option><option v-for="member in workspace.members" :key="member.userId" :value="member.userId">{{ member.user?.name || member.user?.email }}</option></select></label><label>日期<input v-model="form.incurredOn" type="date" required></label></div>
        <label>備註<textarea v-model="form.notes" rows="2"></textarea></label>
        <div class="form-actions"><button class="primary" :disabled="workspace.loading">{{ editingId ? '儲存變更' : '新增支出' }}</button><button v-if="editingId" type="button" class="ghost" @click="reset">取消</button></div>
      </form>
    </div>
  </section>
</template>

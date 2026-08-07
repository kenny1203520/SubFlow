<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useWorkspaceStore } from '../stores/workspace'
import MoneyValue from '../components/MoneyValue.vue'
import EmptyState from '../components/EmptyState.vue'

const workspace = useWorkspaceStore()
const scope = ref<'personal'|'group'|'all'>('personal')
const summary = computed(() => scope.value === 'group' ? workspace.summary : workspace.personalSummary)
async function change(value: 'personal'|'group'|'all') {
  scope.value = value
  if (value === 'group') { if (workspace.currentGroupId) await workspace.refreshGroup() }
  else await workspace.refreshPersonal(value)
}
onMounted(() => void change('personal'))
</script>

<template>
  <section class="page">
    <div class="page-heading"><div><p class="eyebrow">OVERVIEW</p><h1>{{ scope === 'personal' ? '個人財務總覽' : scope === 'all' ? '所有群組總覽' : '群組總覽' }}</h1><p>依個人、目前群組或所有群組查看財務資訊。</p></div><RouterLink class="primary" :to="scope === 'group' ? '/expenses' : '/personal/expenses'">新增支出</RouterLink></div>
    <div class="segmented scope-switch"><button :class="{active:scope==='personal'}" @click="change('personal')">個人</button><button :class="{active:scope==='group'}" :disabled="!workspace.currentGroupId" @click="change('group')">目前群組</button><button :class="{active:scope==='all'}" @click="change('all')">All groups</button></div>
    <div class="metrics"><article v-for="item in (summary?.currencies?.length ? summary.currencies : [{currency:workspace.currentGroup?.currency||'TWD',monthlySubscriptionMinor:summary?.monthlySubscriptionMinor||0,cashOutflowMinor:summary?.monthExpenseMinor||0,activeSubscriptions:summary?.activeSubscriptions||0}])" :key="item.currency"><small>{{ item.currency }} 每月訂閱</small><strong><MoneyValue :amount="item.monthlySubscriptionMinor" :currency="item.currency"/></strong><span>{{ item.activeSubscriptions }} 個啟用中</span></article><article class="accent"><small>本月支出</small><strong>{{ summary?.monthExpenseMinor ?? 0 }}</strong><span>依範圍同步更新</span></article></div>
    <div class="card"><div class="card-title"><h2>即將扣款</h2><RouterLink :to="scope === 'group' ? '/subscriptions' : '/personal/subscriptions'">管理訂閱 →</RouterLink></div><div v-if="summary?.upcoming.length" class="rows"><div v-for="item in summary.upcoming" :key="item.id" class="row"><div class="grow"><strong>{{ item.name }}</strong><small>{{ new Date(item.nextBilling).toLocaleDateString('zh-TW') }} · {{ item.lifecycleStatus || item.status }}</small></div><MoneyValue :amount="item.amountMinor" :currency="item.currency"/></div></div><EmptyState v-else title="近期沒有扣款" description="新增訂閱後，未來 30 天的扣款會出現在這裡。"/></div>
  </section>
</template>

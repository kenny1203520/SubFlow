<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { useWorkspaceStore } from '../stores/workspace'
import MoneyValue from '../components/MoneyValue.vue'
import EmptyState from '../components/EmptyState.vue'
import { useI18n } from '../i18n'

const workspace = useWorkspaceStore()
const { tr, locale } = useI18n()
const route = useRoute()
const scope = ref<'personal'|'group'|'all'>('personal')
const summary = computed(() => scope.value === 'group' ? workspace.summary : workspace.personalSummary)
async function change(value: 'personal'|'group'|'all') {
  scope.value = value
  if (value === 'group') { if (workspace.currentGroupId) await workspace.refreshGroup() }
  else await workspace.refreshPersonal(value)
}
async function initialize(){if(route.params.groupId){scope.value='group';await workspace.selectGroup(String(route.params.groupId))}else await change('personal')}
onMounted(()=>void initialize())
watch(()=>route.params.groupId,()=>void initialize())
</script>

<template>
  <section class="page">
    <div class="page-heading"><div><p class="eyebrow">OVERVIEW</p><h1>{{ scope === 'personal' ? tr('dashboardPersonal') : scope === 'all' ? tr('dashboardAll') : tr('dashboardGroup') }}</h1><p>{{tr('dashboardDesc')}}</p></div><RouterLink class="primary" :to="scope === 'group' ? '/expenses' : '/personal/expenses'">{{tr('addExpense')}}</RouterLink></div>
    <div v-if="!route.params.groupId" class="segmented scope-switch"><button :class="{active:scope==='personal'}" @click="change('personal')">{{tr('personal')}}</button><button :class="{active:scope==='group'}" :disabled="!workspace.currentGroupId" @click="change('group')">{{tr('currentGroup')}}</button><button :class="{active:scope==='all'}" @click="change('all')">{{tr('allGroups')}}</button></div>
    <div class="metrics"><article v-for="item in (summary?.currencies?.length ? summary.currencies : [{currency:workspace.currentGroup?.currency||'TWD',monthlySubscriptionMinor:summary?.monthlySubscriptionMinor||0,cashOutflowMinor:summary?.monthExpenseMinor||0,activeSubscriptions:summary?.activeSubscriptions||0}])" :key="item.currency"><small>{{ item.currency }} 每月訂閱</small><strong><MoneyValue :amount="item.monthlySubscriptionMinor" :currency="item.currency"/></strong><span>{{ item.activeSubscriptions }} 個啟用中</span></article><article class="accent"><small>本月支出</small><strong>{{ summary?.monthExpenseMinor ?? 0 }}</strong><span>依範圍同步更新</span></article></div>
    <div class="card"><div class="card-title"><h2>{{tr('upcoming')}}</h2><RouterLink :to="scope === 'group' ? '/subscriptions' : '/personal/subscriptions'">{{tr('manageSubscriptions')}}</RouterLink></div><div v-if="summary?.upcoming.length" class="rows"><div v-for="item in summary.upcoming" :key="item.id" class="row"><div class="grow"><strong>{{ item.name }}</strong><small>{{ new Date(item.nextBilling).toLocaleDateString(locale) }} · {{ item.lifecycleStatus || item.status }}</small></div><MoneyValue :amount="item.amountMinor" :currency="item.currency"/></div></div><EmptyState v-else :title="tr('noUpcoming')" :description="tr('noUpcomingDesc')"/></div>
  </section>
</template>

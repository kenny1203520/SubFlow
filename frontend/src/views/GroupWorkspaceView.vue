<script setup lang="ts">
import { computed, onMounted, watch } from 'vue'
import { useRoute } from 'vue-router'
import { useWorkspaceStore } from '../stores/workspace'
const route=useRoute(),workspace=useWorkspaceStore()
const groupId=computed(()=>String(route.params.groupId||''))
const group=computed(()=>workspace.groups.find(value=>value.id===groupId.value))
async function activate(){if(groupId.value&&workspace.currentGroupId!==groupId.value)await workspace.selectGroup(groupId.value)}
onMounted(()=>void activate());watch(groupId,()=>void activate())
</script>
<template><section class="group-workspace"><div class="group-workspace-head"><RouterLink class="back-link" to="/groups">← 所有群組</RouterLink><div><p class="eyebrow">GROUP WORKSPACE</p><h1>{{group?.name||'群組工作區'}}</h1><p>{{group?.description||'管理共同支出、訂閱、成員與群組設定。'}}</p></div></div><nav class="group-tabs" aria-label="群組功能"><RouterLink :to="`/groups/${groupId}/overview`">總覽</RouterLink><RouterLink :to="`/groups/${groupId}/expenses`">支出與分帳</RouterLink><RouterLink :to="`/groups/${groupId}/subscriptions`">訂閱</RouterLink><RouterLink :to="`/groups/${groupId}/members`">成員</RouterLink><RouterLink :to="`/groups/${groupId}/settings`">設定</RouterLink></nav><RouterView v-slot="{ Component, route }"><component :is="Component" :key="route.fullPath"/></RouterView></section></template>

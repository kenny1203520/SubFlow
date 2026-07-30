<script setup lang="ts">
import { onMounted,watch } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from './stores/auth'
import { useWorkspaceStore } from './stores/workspace'
const auth=useAuthStore(),workspace=useWorkspaceStore(),router=useRouter()
onMounted(async()=>{await auth.initialize();if(auth.authenticated){await workspace.loadGroups();if(workspace.currentGroupId)await workspace.selectGroup(workspace.currentGroupId)}})
watch(()=>auth.authenticated,value=>{if(!value){workspace.clear();void router.push('/auth')}})
function logout(){auth.logout()}
</script>

<template>
  <div v-if="!auth.ready" class="splash">SubFlow 正在準備你的工作區…</div>
  <RouterView v-else-if="!auth.authenticated" />
  <div v-else class="shell">
    <aside class="sidebar">
      <RouterLink class="brand" to="/"><span>SF</span><strong>SubFlow</strong></RouterLink>
      <nav><RouterLink to="/">總覽</RouterLink><RouterLink to="/groups">群組</RouterLink><RouterLink to="/members">成員與邀請</RouterLink><RouterLink to="/subscriptions">訂閱</RouterLink><RouterLink to="/expenses">共同支出</RouterLink></nav>
      <div class="sidebar-bottom"><RouterLink to="/profile">{{ auth.name }}</RouterLink><button class="ghost" @click="logout">登出</button></div>
    </aside>
    <main class="main"><header class="topbar"><div><small>目前群組</small><select :value="workspace.currentGroupId" @change="workspace.selectGroup(($event.target as HTMLSelectElement).value)"><option value="">請選擇群組</option><option v-for="group in workspace.groups" :key="group.id" :value="group.id">{{ group.name }}</option></select></div><span v-if="workspace.loading" class="sync">同步中</span></header><div v-if="workspace.permissionDenied" class="notice danger">你沒有權限查看或修改這個群組。</div><div v-else-if="workspace.error" class="notice">{{ workspace.error }} <button @click="workspace.refreshGroup">重試</button></div><RouterView /></main>
  </div>
</template>

<script setup lang="ts">
import { onErrorCaptured, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from './stores/auth'
import { useWorkspaceStore } from './stores/workspace'
import { useTheme } from './theme'
import { useI18n } from './i18n'
import LanguageSwitcher from './components/LanguageSwitcher.vue'
import ThemeSwitcher from './components/ThemeSwitcher.vue'

const auth = useAuthStore()
const workspace = useWorkspaceStore()
const router = useRouter()
const route = useRoute()
const { tr } = useI18n()
const routeError = ref<unknown>()
const routeRetry = ref(0)
let hydratedUser = ''
let hydration: Promise<void> | undefined
useTheme()

async function hydrateWorkspace() {
  const userId = String(auth.record?.id || '')
  if (!userId || hydratedUser === userId) return
  if (!hydration) hydration = workspace.loadGroups().then(() => { hydratedUser = userId }).finally(() => { hydration = undefined })
  await hydration
}

async function bootstrap() {
  await auth.initialize()
  if (auth.authenticated) await hydrateWorkspace()
}

function retryRoute() {
  routeError.value = undefined
  routeRetry.value++
}

onErrorCaptured(error => {
  routeError.value = error
  return false
})
onMounted(() => void bootstrap())
watch(() => auth.authenticated, authenticated => {
  if (authenticated) { void hydrateWorkspace(); return }
  hydratedUser = ''
  workspace.clear()
  if (auth.ready && route.name !== 'auth') void router.replace({ name: 'auth' })
})
watch(() => route.fullPath, () => { routeError.value = undefined })
</script>

<template>
  <div v-if="!auth.ready" class="splash"><div class="splash-mark">SF</div><strong>SubFlow</strong></div>
  <RouterView v-else-if="!auth.authenticated" />
  <div v-else class="shell">
    <aside class="sidebar">
      <RouterLink class="brand" :to="{ name: 'dashboard' }"><span>SF</span><strong>SubFlow</strong></RouterLink>
      <p class="sidebar-label">{{ tr('workspace') }}</p>
      <nav>
        <RouterLink :to="{ name: 'dashboard' }"><svg viewBox="0 0 24 24"><path d="M4 13h6V4H4v9Zm0 7h6v-4H4v4Zm10 0h6v-9h-6v9Zm0-16v4h6V4h-6Z" /></svg><span>{{ tr('overview') }}</span></RouterLink>
        <RouterLink :to="{ name: 'personal-expenses' }"><svg viewBox="0 0 24 24"><path d="M4 5h16v14H4zM8 9h8M8 13h5" /></svg><span>{{ tr('personalLedger') }}</span></RouterLink>
        <RouterLink :to="{ name: 'groups' }"><svg viewBox="0 0 24 24"><path d="M4 6.5 12 3l8 3.5-8 3-8-3Zm0 5 8 3 8-3M4 16.5l8 3 8-3" /></svg><span>{{ tr('groups') }}</span></RouterLink>
        <RouterLink v-if="auth.record?.systemRoleId" :to="{ name: 'admin' }"><svg viewBox="0 0 24 24"><path d="M12 3 4 7v5c0 5 3.4 8.6 8 9 4.6-.4 8-4 8-9V7l-8-4Z" /></svg><span>{{ tr('settings') }}</span></RouterLink>
      </nav>
      <div class="sidebar-bottom">
        <RouterLink class="profile-link" :to="{ name: 'profile' }"><span class="user-avatar">{{ auth.name.slice(0, 1).toUpperCase() }}</span><span><small>{{ tr('accountSettings') }}</small>{{ auth.name }}</span></RouterLink>
        <button class="logout-button" @click="auth.logout">{{ tr('logout') }}</button>
      </div>
    </aside>
    <main class="main">
      <header class="topbar">
        <div><small class="topbar-label">SubFlow</small><strong>{{ tr('personalGroupFinance') }}</strong></div>
        <div class="topbar-actions"><span v-if="workspace.loading" class="sync"><i></i>{{ tr('syncing') }}</span><ThemeSwitcher /><LanguageSwitcher /></div>
      </header>
      <div v-if="workspace.permissionDenied" class="notice danger">{{ tr('forbidden') }}</div>
      <div v-else-if="workspace.error" class="notice">{{ workspace.localizedError }} <button @click="workspace.retryLast">{{ tr('retry') }}</button></div>
      <section v-if="routeError" class="page narrow"><div class="card error-state"><h1>{{ tr('routeError') }}</h1><p>{{ tr('routeErrorDesc') }}</p><div class="form-actions"><button class="primary" @click="retryRoute">{{ tr('retry') }}</button><RouterLink class="ghost" :to="{ name: 'dashboard' }">{{ tr('backToDashboard') }}</RouterLink></div></div></section>
      <RouterView v-else v-slot="{ Component }"><component :is="Component" :key="`${route.matched[0]?.path || route.path}-${routeRetry}`" /></RouterView>
    </main>
  </div>
</template>

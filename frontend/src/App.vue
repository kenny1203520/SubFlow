<script setup lang="ts">
import { onErrorCaptured, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from './stores/auth'
import { useWorkspaceStore } from './stores/workspace'
import { useSetupStore } from './stores/setup'
import { useTheme } from './theme'
import { useI18n } from './i18n'
import LanguageSwitcher from './components/LanguageSwitcher.vue'
import NotificationBell from './components/NotificationBell.vue'
import ThemeSwitcher from './components/ThemeSwitcher.vue'
import PwaUpdatePrompt from './components/PwaUpdatePrompt.vue'
import ToastContainer from './components/ToastContainer.vue'
import TimezoneMismatchDialog from './components/TimezoneMismatchDialog.vue'
import { browserTimezone } from './timezone'

const auth = useAuthStore()
const workspace = useWorkspaceStore()
const setup = useSetupStore()
const router = useRouter()
const route = useRoute()
const { tr } = useI18n()
const routeError = ref<unknown>()
const routeRetry = ref(0)
let hydratedUser = ''
let hydration: Promise<void> | undefined
useTheme()

// Checked once per login/app-open (per the user's chosen cadence), not
// persisted anywhere — a fresh mismatch check runs again next time they sign
// in, even if they dismissed the last one.
const timezoneChecked = ref(false)
const timezoneMismatch = ref<{ saved: string; current: string } | undefined>()
function checkTimezoneMismatch() {
  if (timezoneChecked.value) return
  timezoneChecked.value = true
  const saved = auth.record?.timezone
  const current = browserTimezone()
  if (saved && current && saved !== current) timezoneMismatch.value = { saved, current }
}
async function applyDetectedTimezone() {
  if (!timezoneMismatch.value || !auth.record) return
  await auth.updateProfile({ name: auth.record.name, timezone: timezoneMismatch.value.current, default_currency: auth.record.defaultCurrency })
  timezoneMismatch.value = undefined
}

async function hydrateWorkspace() {
  const userId = String(auth.record?.id || '')
  if (!userId || hydratedUser === userId) return
  if (!hydration) hydration = workspace.loadGroups().then(() => { hydratedUser = userId }).finally(() => { hydration = undefined })
  await hydration
  checkTimezoneMismatch()
}

async function bootstrap() {
	try { await setup.refresh() } catch { setup.status.initialized = true; setup.ready = true }
	if (!setup.initialized) { if (route.name !== 'setup') await router.replace({ name: 'setup' }); return }
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
  timezoneChecked.value = false
  timezoneMismatch.value = undefined
  workspace.clear()
  if (auth.ready && setup.initialized && route.name !== 'auth') void router.replace({ name: 'auth' })
})
watch(() => setup.initialized, initialized => { if (!initialized && route.name !== 'setup') void router.replace({ name: 'setup' }) })
watch(() => route.fullPath, () => { routeError.value = undefined })
</script>

<template>
  <PwaUpdatePrompt />
  <ToastContainer />
  <TimezoneMismatchDialog :open="!!timezoneMismatch" :saved-timezone="timezoneMismatch?.saved||''" :current-timezone="timezoneMismatch?.current||''" @update="applyDetectedTimezone" @later="timezoneMismatch=undefined" />
  <div v-if="!setup.ready || (setup.initialized && !auth.ready)" class="splash"><div class="splash-mark">SF</div><strong>SubFlow</strong></div>
  <RouterView v-else-if="!setup.initialized" />
  <RouterView v-else-if="!auth.authenticated" />
  <div v-else class="shell">
    <aside class="sidebar">
      <RouterLink class="brand" :to="{ name: 'dashboard' }"><span>SF</span><strong>SubFlow</strong></RouterLink>
      <p class="sidebar-label">{{ tr('workspace') }}</p>
      <nav>
        <RouterLink :to="{ name: 'dashboard' }"><svg viewBox="0 0 24 24"><path d="M4 13h6V4H4v9Zm0 7h6v-4H4v4Zm10 0h6v-9h-6v9Zm0-16v4h6V4h-6Z" /></svg><span>{{ tr('overview') }}</span></RouterLink>
        <RouterLink :to="{ name: 'personal-expenses' }"><svg viewBox="0 0 24 24"><path d="M4 5h16v14H4zM8 9h8M8 13h5" /></svg><span>{{ tr('personalLedger') }}</span></RouterLink>
        <RouterLink :to="{ name: 'groups' }"><svg viewBox="0 0 24 24"><path d="M4 6.5 12 3l8 3.5-8 3-8-3Zm0 5 8 3 8-3M4 16.5l8 3 8-3" /></svg><span>{{ tr('groups') }}</span></RouterLink>
        <RouterLink :to="{ name: 'about' }"><svg viewBox="0 0 24 24"><circle cx="12" cy="12" r="9" /><path d="M12 16v-4M12 8h.01" /></svg><span>{{ tr('about') }}</span></RouterLink>
        <RouterLink class="mobile-only-link" :to="{ name: 'profile' }"><svg viewBox="0 0 24 24"><circle cx="12" cy="8" r="4" /><path d="M4 20c0-4 3.6-7 8-7s8 3 8 7" /></svg><span>{{ tr('profile') }}</span></RouterLink>
      </nav>
      <div class="sidebar-bottom">
        <RouterLink v-if="auth.canAdminister" class="admin-link" :to="{ name: 'admin' }"><svg viewBox="0 0 24 24"><path d="M12 3 4 7v5c0 5 3.4 8.6 8 9 4.6-.4 8-4 8-9V7l-8-4Z" /></svg><span>{{ tr('systemAdministration') }}</span></RouterLink>
        <RouterLink class="profile-link" :to="{ name: 'profile' }"><span class="user-avatar">{{ auth.name.slice(0, 1).toUpperCase() }}</span><span><small>{{ tr('accountSettings') }}</small>{{ auth.name }}</span></RouterLink>
        <button class="logout-button" @click="auth.logout">{{ tr('logout') }}</button>
      </div>
    </aside>
    <main class="main">
      <header class="topbar">
        <div><small class="topbar-label">SubFlow</small><strong>{{ tr('personalGroupFinance') }}</strong></div>
        <div class="topbar-actions"><span v-if="workspace.loading" class="sync"><i></i>{{ tr('syncing') }}</span><NotificationBell /><ThemeSwitcher /><LanguageSwitcher /></div>
      </header>
      <div v-if="workspace.permissionDenied" class="notice danger">{{ tr('forbidden') }}</div>
      <div v-else-if="workspace.error" class="notice">{{ workspace.localizedError }} <button @click="workspace.retryLast">{{ tr('retry') }}</button></div>
      <section v-if="routeError" class="page narrow"><div class="card error-state"><h1>{{ tr('routeError') }}</h1><p>{{ tr('routeErrorDesc') }}</p><div class="form-actions"><button class="primary" @click="retryRoute">{{ tr('retry') }}</button><RouterLink class="ghost" :to="{ name: 'dashboard' }">{{ tr('backToDashboard') }}</RouterLink></div></div></section>
      <RouterView v-else v-slot="{ Component }"><component :is="Component" :key="`${route.matched[0]?.path || route.path}-${routeRetry}`" /></RouterView>
    </main>
  </div>
</template>

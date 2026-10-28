<script setup lang="ts">
import { computed, onErrorCaptured, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from './stores/auth'
import { useWorkspaceStore } from './stores/workspace'
import { useSetupStore } from './stores/setup'
import { useTheme } from './theme'
import { useI18n } from './i18n'
import LanguageSwitcher from './components/LanguageSwitcher.vue'
import Logo from './components/Logo.vue'
import NotificationBell from './components/NotificationBell.vue'
import ThemeSwitcher from './components/ThemeSwitcher.vue'
import PwaUpdatePrompt from './components/PwaUpdatePrompt.vue'
import ToastContainer from './components/ToastContainer.vue'
import TimezoneMismatchDialog from './components/TimezoneMismatchDialog.vue'
import UpgradeMaintenance from './components/UpgradeMaintenance.vue'
import { browserTimezone } from './timezone'
import { useUpgradeStore } from './stores/upgrade'

const auth = useAuthStore()
const workspace = useWorkspaceStore()
const setup = useSetupStore()
const upgrade = useUpgradeStore()
const router = useRouter()
const route = useRoute()
const { tr } = useI18n()
const publicRoute = computed(() => route.meta.public === true)
const routeError = ref<unknown>()
const routeRetry = ref(0)
let hydratedUser = ''
let hydration: Promise<void> | undefined
let bootstrapRunning = false
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
	if (bootstrapRunning) return
	bootstrapRunning = true
	try {
	await upgrade.start()
	if (upgrade.blocked) return
	try { await setup.refresh() } catch { if (upgrade.blocked) return; setup.status.initialized = true; setup.ready = true }
	if (!setup.initialized) { if (route.name !== 'setup') await router.replace({ name: 'setup' }); return }
  await auth.initialize()
  if (auth.authenticated) await hydrateWorkspace()
	} finally {
		bootstrapRunning = false
	}
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
watch(() => upgrade.blocked, (blocked, wasBlocked) => {
  if (blocked) { workspace.pauseForMaintenance(); return }
  if (wasBlocked) { void bootstrap(); void workspace.resumeAfterMaintenance() }
})
watch(() => auth.authenticated, authenticated => {
  if (authenticated) { void hydrateWorkspace(); return }
  hydratedUser = ''
  timezoneChecked.value = false
  timezoneMismatch.value = undefined
  workspace.clear()
  if (auth.ready && setup.initialized && route.name !== 'auth' && !publicRoute.value) void router.replace({ name: 'auth' })
})
watch(() => setup.initialized, initialized => { if (!initialized && route.name !== 'setup') void router.replace({ name: 'setup' }) })
watch(() => route.fullPath, () => { routeError.value = undefined })
</script>

<template>
  <div class="app-frame">
    <div class="app-content">
      <PwaUpdatePrompt v-if="!upgrade.blocked" />
      <ToastContainer />
      <TimezoneMismatchDialog :open="!!timezoneMismatch" :saved-timezone="timezoneMismatch?.saved||''" :current-timezone="timezoneMismatch?.current||''" @update="applyDetectedTimezone" @later="timezoneMismatch=undefined" />
      <div v-if="!setup.ready || (setup.initialized && !auth.ready)" class="splash"><Logo variant="lockup" size="lg" /></div>
      <RouterView v-else-if="!setup.initialized" />
      <RouterView v-else-if="!auth.authenticated || publicRoute" />
      <div v-else class="shell">
    <aside class="sidebar">
      <RouterLink class="brand" :to="{ name: 'home' }"><Logo variant="lockup" size="md" /></RouterLink>
      <p class="sidebar-label">{{ tr('workspace') }}</p>
      <nav :aria-label="tr('primaryNavigation')">
        <RouterLink :to="{ name: 'home' }"><svg viewBox="0 0 24 24"><path d="M3 10.5 12 3l9 7.5v9a1.5 1.5 0 0 1-1.5 1.5h-15A1.5 1.5 0 0 1 3 19.5v-9ZM9 21v-7h6v7" /></svg><span>{{ tr('home') }}</span></RouterLink>
        <RouterLink :to="{ name: 'personal-ledger' }" :class="{ 'router-link-active': route.path.startsWith('/personal') }"><svg viewBox="0 0 24 24"><path d="M4 5h16v14H4zM8 9h8M8 13h5" /></svg><span>{{ tr('personalLedger') }}</span></RouterLink>
        <RouterLink :to="{ name: 'groups' }" :class="{ 'router-link-active': route.path.startsWith('/groups') }"><svg viewBox="0 0 24 24"><path d="M4 6.5 12 3l8 3.5-8 3-8-3Zm0 5 8 3 8-3M4 16.5l8 3 8-3" /></svg><span>{{ tr('groups') }}</span></RouterLink>
        <RouterLink :to="{ name: 'more' }"><svg viewBox="0 0 24 24"><circle cx="5" cy="12" r="1"/><circle cx="12" cy="12" r="1"/><circle cx="19" cy="12" r="1"/></svg><span>{{ tr('more') }}</span></RouterLink>
      </nav>
      <div class="sidebar-bottom">
        <RouterLink v-if="auth.canAdminister" class="admin-link" :to="{ name: 'admin' }"><svg viewBox="0 0 24 24"><path d="M12 3 4 7v5c0 5 3.4 8.6 8 9 4.6-.4 8-4 8-9V7l-8-4Z" /></svg><span>{{ tr('systemAdministration') }}</span></RouterLink>
        <RouterLink class="profile-link" :to="{ name: 'profile' }"><span class="user-avatar">{{ auth.name.slice(0, 1).toUpperCase() }}</span><span><small>{{ tr('accountSettings') }}</small>{{ auth.name }}</span></RouterLink>
        <button class="logout-button" :disabled="auth.signingOut" @click="auth.signOut" type="button">{{ tr('logout') }}</button>
      </div>
    </aside>
    <main class="main">
      <header class="topbar">
        <div class="topbar-brand"><Logo variant="mark" size="sm" /><div><small class="topbar-label">SubFlow</small><strong>{{ tr('personalGroupFinance') }}</strong></div></div>
        <div class="topbar-actions">
          <span v-if="workspace.loading" class="sync"><i></i>{{ tr('syncing') }}</span>
          <span v-if="!workspace.online" class="offline-indicator" :title="tr('offlineIndicatorHelp')">⚠ {{ tr('offlineMode') }}</span>
          <button v-else-if="workspace.outboxPending>0" type="button" class="sync-pending" :class="{'has-error':workspace.hasSyncErrors}" :title="tr('pendingSyncHelp')" @click="workspace.syncOutbox()">↻ {{ tr('pendingSyncCount',{count:workspace.outboxPending}) }}</button>
          <NotificationBell /><ThemeSwitcher /><LanguageSwitcher />
        </div>
      </header>
      <div v-if="workspace.permissionDenied" class="notice danger">{{ tr('forbidden') }}</div>
      <div v-else-if="workspace.error" class="notice">{{ workspace.localizedError }} <button @click="workspace.retryLast" type="button">{{ tr('retry') }}</button></div>
      <section v-if="routeError" class="page narrow"><div class="card error-state"><h1>{{ tr('routeError') }}</h1><p>{{ tr('routeErrorDesc') }}</p><div class="form-actions"><button class="primary" @click="retryRoute" type="button">{{ tr('retry') }}</button><RouterLink class="ghost" :to="{ name: 'home' }">{{ tr('backToDashboard') }}</RouterLink></div></div></section>
      <RouterView v-else v-slot="{ Component }"><component :is="Component" :key="`${route.matched[0]?.path || route.path}-${routeRetry}`" /></RouterView>
    </main>
      </div>
    </div>
    <UpgradeMaintenance />
  </div>
</template>

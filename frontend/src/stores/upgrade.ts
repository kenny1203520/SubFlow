import { computed, onScopeDispose, ref } from 'vue'
import { defineStore } from 'pinia'
import { isClientUpdateRequired, setMaintenanceBlocked, signalMaintenance } from '../runtime/maintenance'

export type UpgradePhase = 'checking' | 'backing_up' | 'migrating' | 'verifying' | 'ready' | 'failed'
type RuntimeStatus = { state: UpgradePhase; phase: UpgradePhase; ready: boolean; problemCode?: string; appVersion?: string; pocketBaseVersion?: string }

const initialStatus: RuntimeStatus = { state: 'checking', phase: 'checking', ready: false }

export const useUpgradeStore = defineStore('upgrade', () => {
  const status = ref<RuntimeStatus>({ ...initialStatus })
  const checked = ref(false)
  const clientUpdateRequired = ref(isClientUpdateRequired())
  const blocked = computed(() => !status.value.ready || clientUpdateRequired.value)
  let timer: ReturnType<typeof setTimeout> | undefined
  let initialCheck: Promise<void> | undefined

  function apply(value: RuntimeStatus) {
    const known = ['checking', 'backing_up', 'migrating', 'verifying', 'ready', 'failed'].includes(value.state)
    status.value = known ? value : { ...initialStatus }
    checked.value = true
    if (status.value.ready && status.value.state === 'ready') {
      try { localStorage.setItem('subflow-runtime-last-ready', 'true') } catch { /* Storage may be disabled. */ }
      setMaintenanceBlocked(clientUpdateRequired.value)
    } else {
      setMaintenanceBlocked(true)
      try { localStorage.removeItem('subflow-runtime-last-ready') } catch { /* Storage may be disabled. */ }
    }
  }

  async function refresh() {
    try {
      const response = await fetch('/api/subflow/v1/runtime/status', { headers: { Accept: 'application/json', 'X-SubFlow-Protocol': '1' }, cache: 'no-store' })
      if (!response.ok) throw new Error('runtime status unavailable')
      const body = await response.json() as { data?: RuntimeStatus }
      if (!body.data) throw new Error('runtime status invalid')
      apply(body.data)
    } catch {
      checked.value = true
      if (typeof navigator !== 'undefined' && !navigator.onLine) {
        let wasReady = false
        try { wasReady = localStorage.getItem('subflow-runtime-last-ready') === 'true' } catch { /* Storage may be disabled. */ }
        if (wasReady && !clientUpdateRequired.value) {
          status.value = { state: 'ready', phase: 'ready', ready: true }
          setMaintenanceBlocked(false)
        }
      } else {
        status.value = { ...initialStatus }
        setMaintenanceBlocked(true)
      }
    }
    schedule()
  }

  function schedule() {
    if (timer) clearTimeout(timer)
    timer = setTimeout(() => { void refresh() }, blocked.value ? 2500 : 12000)
  }

  function start() {
    if (!initialCheck) {
      setMaintenanceBlocked(true)
      initialCheck = refresh()
    }
    return initialCheck
  }

  function handleMaintenance(event: Event) {
    const detail = (event as CustomEvent<{ state?: UpgradePhase; problemCode?: string }>).detail
    if (detail?.state) status.value = { ...status.value, state: detail.state, phase: detail.state, ready: false, problemCode: detail.problemCode }
    else status.value = { ...status.value, state: 'checking', phase: 'checking', ready: false }
    setMaintenanceBlocked(true)
    try { localStorage.removeItem('subflow-runtime-last-ready') } catch { /* Storage may be disabled. */ }
    schedule()
  }

  function handleClientUpdate() {
    clientUpdateRequired.value = true
    setMaintenanceBlocked(true)
  }

  const handleOnline = () => { void refresh() }
  if (typeof window !== 'undefined') {
    window.addEventListener('subflow:maintenance', handleMaintenance)
    window.addEventListener('subflow:client-update-required', handleClientUpdate)
    window.addEventListener('online', handleOnline)
    onScopeDispose(() => {
      if (timer) clearTimeout(timer)
      window.removeEventListener('subflow:maintenance', handleMaintenance)
      window.removeEventListener('subflow:client-update-required', handleClientUpdate)
      window.removeEventListener('online', handleOnline)
    })
  }

  function reportMaintenance() { signalMaintenance(); handleMaintenance(new CustomEvent('subflow:maintenance')) }
  function reloadAfterSaving() { window.location.reload() }

  return { status, checked, blocked, clientUpdateRequired, start, refresh, reportMaintenance, reloadAfterSaving }
})

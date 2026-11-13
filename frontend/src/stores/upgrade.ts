import { computed, onScopeDispose, ref } from 'vue'
import { defineStore } from 'pinia'
import { isClientUpdateRequired, setMaintenanceBlocked, signalMaintenance } from '../runtime/maintenance'

export type UpgradePhase = 'checking' | 'backing_up' | 'migrating' | 'verifying' | 'ready' | 'failed'
type RuntimeStatus = { state: UpgradePhase; phase: UpgradePhase; ready: boolean; problemCode?: string; appVersion?: string; pocketBaseVersion?: string }

const initialStatus: RuntimeStatus = { state: 'checking', phase: 'checking', ready: false }

export const useUpgradeStore = defineStore('upgrade', () => {
  const status = ref<RuntimeStatus>({ ...initialStatus })
  const checked = ref(false)
  const connectionIssue = ref<'timeout' | 'invalid' | 'unavailable'>()
  const clientUpdateRequired = ref(isClientUpdateRequired())
  const blocked = computed(() => !status.value.ready || !!connectionIssue.value || clientUpdateRequired.value)
  const checking = ref(false)
  let timer: ReturnType<typeof setTimeout> | undefined
  let initialCheck: Promise<void> | undefined
  let inFlight: Promise<void> | undefined
  let activeRequest: AbortController | undefined
  let failures = 0
  let disposed = false

  function apply(value: RuntimeStatus) {
    status.value = value
    connectionIssue.value = undefined
    failures = 0
    checked.value = true
    if (status.value.ready && status.value.state === 'ready') {
      try { localStorage.setItem('subflow-runtime-last-ready', 'true') } catch { /* Storage may be disabled. */ }
      setMaintenanceBlocked(clientUpdateRequired.value)
    } else {
      setMaintenanceBlocked(true)
      try { localStorage.removeItem('subflow-runtime-last-ready') } catch { /* Storage may be disabled. */ }
    }
  }

  function refresh() {
    if (inFlight) return inFlight
    clearTimeout(timer)
    checking.value = true
    inFlight = requestStatus().finally(() => {
      inFlight = undefined
      checking.value = false
      schedule()
    })
    return inFlight
  }

  async function requestStatus() {
    const controller = new AbortController()
    activeRequest = controller
    const timeout = setTimeout(() => controller.abort(), 5000)
    let issue: typeof connectionIssue.value = 'unavailable'
    try {
      const response = await fetch('/api/subflow/v1/runtime/status', { headers: { Accept: 'application/json', 'X-SubFlow-Protocol': '1' }, cache: 'no-store', signal: controller.signal })
      if (!response.ok) throw new Error('runtime status unavailable')
      issue = 'invalid'
      const body = await response.json() as { data?: RuntimeStatus }
      const value = body?.data
      if (!value || !['checking', 'backing_up', 'migrating', 'verifying', 'ready', 'failed'].includes(value.state)
        || value.phase !== value.state || value.ready !== (value.state === 'ready')
        || (value.problemCode !== undefined && (typeof value.problemCode !== 'string' || !/^[a-z0-9_-]{1,64}$/.test(value.problemCode)))) {
        throw new Error('runtime status invalid')
      }
      if (!disposed) apply(value)
    } catch {
      if (disposed) return
      checked.value = true
      failures++
      if (typeof navigator !== 'undefined' && !navigator.onLine) {
        let wasReady = false
        try { wasReady = localStorage.getItem('subflow-runtime-last-ready') === 'true' } catch { /* Storage may be disabled. */ }
        if (wasReady && !clientUpdateRequired.value) {
          status.value = { state: 'ready', phase: 'ready', ready: true }
          connectionIssue.value = undefined
          setMaintenanceBlocked(false)
          return
        }
      }
      connectionIssue.value = controller.signal.aborted ? 'timeout' : issue
      setMaintenanceBlocked(true)
    } finally {
      clearTimeout(timeout)
      activeRequest = undefined
    }
  }

  function schedule() {
    if (timer) clearTimeout(timer)
    if (disposed || inFlight) return
    const delay = failures ? Math.min(3000 * 2 ** (failures - 1), 30000) : blocked.value ? 2500 : 12000
    timer = setTimeout(() => { void refresh() }, delay)
  }

  function start() {
    if (!initialCheck) {
      setMaintenanceBlocked(true)
      initialCheck = refresh()
    }
    return initialCheck
  }

  function handleMaintenance(event: Event) {
    connectionIssue.value = undefined
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
      disposed = true
      activeRequest?.abort()
      if (timer) clearTimeout(timer)
      window.removeEventListener('subflow:maintenance', handleMaintenance)
      window.removeEventListener('subflow:client-update-required', handleClientUpdate)
      window.removeEventListener('online', handleOnline)
    })
  }

  function reportMaintenance() { signalMaintenance(); handleMaintenance(new CustomEvent('subflow:maintenance')) }
  function reloadAfterSaving() { window.location.reload() }

  return { status, checked, checking, connectionIssue, blocked, clientUpdateRequired, start, refresh, reportMaintenance, reloadAfterSaving }
})

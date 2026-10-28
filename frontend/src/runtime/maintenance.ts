let blocked = false
let clientUpdateRequired = false

export function isMaintenanceBlocked(): boolean { return blocked || clientUpdateRequired }
export function isClientUpdateRequired(): boolean { return clientUpdateRequired }

export function setMaintenanceBlocked(value: boolean) {
  blocked = value
  if (typeof window !== 'undefined') window.dispatchEvent(new CustomEvent('subflow:maintenance-state', { detail: { blocked: isMaintenanceBlocked() } }))
}

export function signalMaintenance() {
  blocked = true
  if (typeof window !== 'undefined') {
    window.dispatchEvent(new CustomEvent('subflow:maintenance', { detail: { state: 'checking' } }))
    window.dispatchEvent(new CustomEvent('subflow:maintenance-state', { detail: { blocked: true } }))
  }
}

export function signalClientUpdateRequired() {
  clientUpdateRequired = true
  blocked = true
  if (typeof window !== 'undefined') {
    window.dispatchEvent(new CustomEvent('subflow:client-update-required'))
    window.dispatchEvent(new CustomEvent('subflow:maintenance-state', { detail: { blocked: true } }))
  }
}

// @vitest-environment jsdom
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, disposePinia, setActivePinia } from 'pinia'
import { isMaintenanceBlocked, setMaintenanceBlocked } from '../runtime/maintenance'
import { useUpgradeStore } from './upgrade'

let pinia: ReturnType<typeof createPinia>

beforeEach(() => {
  vi.useFakeTimers()
  localStorage.clear()
  setMaintenanceBlocked(false)
  pinia = createPinia()
  setActivePinia(pinia)
  Object.defineProperty(navigator, 'onLine', { value: true, configurable: true })
})

afterEach(() => {
  disposePinia(pinia)
  vi.unstubAllGlobals()
  vi.useRealTimers()
  setMaintenanceBlocked(false)
  localStorage.clear()
})

describe('upgrade store', () => {
  it('keeps writes blocked after maintenance was observed, even if the device goes offline', async () => {
    localStorage.setItem('subflow-runtime-last-ready', 'true')
    const fetchMock = vi.fn().mockResolvedValue(new Response(JSON.stringify({ data: { state: 'migrating', phase: 'migrating', ready: false } }), { status: 200 }))
    vi.stubGlobal('fetch', fetchMock)
    const upgrade = useUpgradeStore()

    await upgrade.refresh()
    expect(upgrade.blocked).toBe(true)
    expect(localStorage.getItem('subflow-runtime-last-ready')).toBeNull()

    Object.defineProperty(navigator, 'onLine', { value: false, configurable: true })
    fetchMock.mockRejectedValue(new Error('offline'))
    await upgrade.refresh()

    expect(upgrade.blocked).toBe(true)
    expect(isMaintenanceBlocked()).toBe(true)
  })

  it('unblocks synchronization after the backend confirms readiness', async () => {
    const fetchMock = vi.fn()
      .mockResolvedValueOnce(new Response(JSON.stringify({ data: { state: 'backing_up', phase: 'backing_up', ready: false } }), { status: 200 }))
      .mockResolvedValueOnce(new Response(JSON.stringify({ data: { state: 'ready', phase: 'ready', ready: true } }), { status: 200 }))
    vi.stubGlobal('fetch', fetchMock)
    const upgrade = useUpgradeStore()

    await upgrade.refresh()
    expect(upgrade.blocked).toBe(true)
    await upgrade.refresh()
    expect(upgrade.blocked).toBe(false)
    expect(isMaintenanceBlocked()).toBe(false)
  })
})

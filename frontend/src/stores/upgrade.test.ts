// @vitest-environment jsdom
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, disposePinia, setActivePinia } from 'pinia'
import { isMaintenanceBlocked, setMaintenanceBlocked } from '../runtime/maintenance'
import { useUpgradeStore } from './upgrade'

let pinia: ReturnType<typeof createPinia>
const storage = new Map<string, string>()
const storageMock = {
  clear: () => storage.clear(),
  getItem: (key: string) => storage.get(key) ?? null,
  removeItem: (key: string) => { storage.delete(key) },
  setItem: (key: string, value: string) => { storage.set(key, String(value)) },
}

beforeEach(() => {
  vi.useFakeTimers()
  storage.clear()
  vi.stubGlobal('localStorage', storageMock)
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
  storage.clear()
})

describe('upgrade store', () => {
  it('times out a pending response body and shares the in-flight request', async () => {
    const fetchMock = vi.fn((_url, options: RequestInit) => Promise.resolve({
      ok: true,
      json: () => new Promise((_resolve, reject) => options.signal?.addEventListener('abort', () => reject(new Error('aborted')))),
    }))
    vi.stubGlobal('fetch', fetchMock)
    const upgrade = useUpgradeStore()
    const request = upgrade.refresh()
    void upgrade.refresh()
    await vi.advanceTimersByTimeAsync(5000)
    await request
    expect(fetchMock).toHaveBeenCalledOnce()
    expect(upgrade.connectionIssue).toBe('timeout')
    expect(upgrade.blocked).toBe(true)
  })

  it('distinguishes a proxy error from malformed status and recovers', async () => {
    vi.stubGlobal('fetch', vi.fn()
      .mockResolvedValueOnce(new Response('<html>Proxy unavailable</html>', { status: 503 }))
      .mockResolvedValueOnce(new Response('{invalid'))
      .mockResolvedValueOnce(new Response(JSON.stringify({ data: { state: 'ready', phase: 'ready', ready: true } }))))
    const upgrade = useUpgradeStore()
    await upgrade.refresh()
    expect(upgrade.connectionIssue).toBe('unavailable')
    await upgrade.refresh()
    expect(upgrade.connectionIssue).toBe('invalid')
    await upgrade.refresh()
    expect(upgrade.blocked).toBe(false)
  })
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

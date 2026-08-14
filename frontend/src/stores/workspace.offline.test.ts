// @vitest-environment jsdom
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { pb } from '../pocketbase'
import { useWorkspaceStore } from './workspace'

vi.mock('../offline/db', () => ({
  get: vi.fn(async () => undefined),
  set: vi.fn(async () => undefined),
  del: vi.fn(async () => undefined),
  getAll: vi.fn(async () => []),
}))

const snapshots = new Map<string, unknown>()
vi.mock('../offline/snapshot', () => ({
  // Cloning on write matters here, not just in the real db.ts: the store
  // passes its live reactive arrays straight in, and without a clone this
  // mock would keep a reference to them — a later in-place mutation of the
  // store's array (as the read-fallback test does to simulate "gone after
  // going offline") would silently mutate the "saved" snapshot too.
  saveSnapshot: vi.fn(async (userId: string, scope: string, groupId: string, bundle: unknown) => { snapshots.set(`${userId}:${scope}:${groupId}`, JSON.parse(JSON.stringify(bundle))) }),
  loadSnapshot: vi.fn(async (userId: string, scope: string, groupId: string) => snapshots.get(`${userId}:${scope}:${groupId}`)),
  saveGroups: vi.fn(async () => {}),
  loadGroups: vi.fn(async () => undefined),
  saveCurrencies: vi.fn(async () => {}),
  loadCurrencies: vi.fn(async () => undefined),
}))

let outboxEntries: Array<{ id: string; userId: string; kind: string; op: string; scope: string; groupId: string; targetId: string; payload?: unknown; createdAt: string; status: string; error?: string }> = []
vi.mock('../offline/outbox', () => ({
  localId: () => `local-${Math.random().toString(36).slice(2)}`,
  isLocalId: (id: string) => id.startsWith('local-'),
  enqueue: vi.fn(async (entry: { userId: string }) => {
    const full = { ...entry, id: `outbox-${outboxEntries.length}`, createdAt: new Date().toISOString(), status: 'pending' }
    outboxEntries.push(full as never)
    return full
  }),
  listForUser: vi.fn(async (userId: string) => outboxEntries.filter(e => e.userId === userId)),
  remove: vi.fn(async (id: string) => { outboxEntries = outboxEntries.filter(e => e.id !== id) }),
  markFailed: vi.fn(async (id: string, error: string) => { const e = outboxEntries.find(v => v.id === id); if (e) { e.status = 'failed'; e.error = error } }),
  updateTargetId: vi.fn(async (id: string, targetId: string) => { const e = outboxEntries.find(v => v.id === id); if (e) e.targetId = targetId }),
}))

function validToken() {
  const payload = btoa(JSON.stringify({ exp: Math.floor(Date.now() / 1000) + 3600 })).replace(/=/g, '').replace(/\+/g, '-').replace(/\//g, '_')
  return `e30.${payload}.signature`
}
function envelope(data: unknown) { return new Response(JSON.stringify({ data }), { status: 200, headers: { 'Content-Type': 'application/json' } }) }

beforeEach(() => {
  pb.authStore.clear()
  pb.authStore.save(validToken(), { id: 'u1', email: 'u1@example.com', collectionId: 'users', collectionName: 'users' })
  setActivePinia(createPinia())
  snapshots.clear()
  outboxEntries = []
})
afterEach(() => { vi.unstubAllGlobals() })

describe('workspace store offline behavior', () => {
  it('falls back to a saved snapshot when refreshGroup runs offline', async () => {
    const fetchMock = vi.fn().mockImplementation((url: string) => {
      if (url.includes('/groups?perPage')) return Promise.resolve(envelope([{ id: 'g1', name: 'Home', description: '', currency: 'TWD', timezone: 'UTC', color: '#000', ownerId: 'u1', createdAt: '', updatedAt: '' }]))
      if (url.includes('/currencies')) return Promise.resolve(envelope([]))
      if (url.includes('/invitations/pending')) return Promise.resolve(envelope([]))
      if (url.includes('/notifications')) return Promise.resolve(envelope([]))
      if (url.includes('/groups/g1/members')) return Promise.resolve(envelope([]))
      if (url.includes('/groups/g1/subscriptions')) return Promise.resolve(envelope([]))
      if (url.includes('/groups/g1/expenses')) return Promise.resolve(envelope([{ id: 'e1', groupId: 'g1', title: 'Lunch', category: '', amountMinor: 100, currency: 'TWD', baseCurrency: 'TWD', baseAmountMinor: 100, exchangeRate: '1', exchangeRateDate: '', rateMode: 'automatic', paidBy: 'u1', incurredOn: '2026-01-01', notes: '', createdAt: '', updatedAt: '' }]))
      if (url.includes('/groups/g1/settlements')) return Promise.resolve(envelope([]))
      if (url.includes('/groups/g1/summary')) return Promise.resolve(envelope({}))
      if (url.includes('/groups/g1/access')) return Promise.resolve(envelope({ permissions: [] }))
      if (url.includes('/events')) return Promise.reject(new Error('no sse in test'))
      return Promise.reject(new Error(`unexpected fetch: ${url}`))
    })
    vi.stubGlobal('fetch', fetchMock)

    const workspace = useWorkspaceStore()
    await workspace.selectGroup('g1')
    expect(workspace.expenses).toHaveLength(1)
    expect(workspace.expenses[0].title).toBe('Lunch')

    // Now go offline and force a fresh refreshGroup — the live fetch must
    // not even be attempted, and the previously-saved snapshot should be
    // what repopulates the arrays.
    workspace.expenses.length = 0
    Object.defineProperty(navigator, 'onLine', { value: false, configurable: true })
    window.dispatchEvent(new Event('offline'))
    fetchMock.mockClear()
    await workspace.refreshGroup()

    expect(workspace.expenses).toHaveLength(1)
    expect(workspace.expenses[0].title).toBe('Lunch')
  })

  it('queues a create locally when offline and replays it once back online', async () => {
    const fetchMock = vi.fn().mockImplementation((url: string, init?: RequestInit) => {
      if (url.includes('/groups?perPage')) return Promise.resolve(envelope([{ id: 'g1', name: 'Home', description: '', currency: 'TWD', timezone: 'UTC', color: '#000', ownerId: 'u1', createdAt: '', updatedAt: '' }]))
      if (url.includes('/currencies')) return Promise.resolve(envelope([]))
      if (url.includes('/invitations/pending')) return Promise.resolve(envelope([]))
      if (url.includes('/notifications')) return Promise.resolve(envelope([]))
      if (url.includes('/groups/g1/members')) return Promise.resolve(envelope([]))
      if (url.includes('/groups/g1/subscriptions')) return Promise.resolve(envelope([]))
      if (url.includes('/groups/g1/expenses') && init?.method === 'POST') return Promise.resolve(envelope({ id: 'real-e1' }))
      if (url.includes('/groups/g1/expenses')) return Promise.resolve(envelope([]))
      if (url.includes('/groups/g1/settlements')) return Promise.resolve(envelope([]))
      if (url.includes('/groups/g1/summary')) return Promise.resolve(envelope({}))
      if (url.includes('/groups/g1/access')) return Promise.resolve(envelope({ permissions: [] }))
      if (url.includes('/events')) return Promise.reject(new Error('no sse in test'))
      return Promise.reject(new Error(`unexpected fetch: ${url}`))
    })
    vi.stubGlobal('fetch', fetchMock)

    const workspace = useWorkspaceStore()
    await workspace.selectGroup('g1')

    Object.defineProperty(navigator, 'onLine', { value: false, configurable: true })
    window.dispatchEvent(new Event('offline'))

    const ok = await workspace.addExpense({ title: 'Offline Snack', category: '', amountMinor: 50, currency: 'TWD', paidBy: 'u1', notes: '' })
    expect(ok).toBe(true)
    expect(workspace.expenses.some(item => item.title === 'Offline Snack' && item.pendingSync)).toBe(true)
    expect(outboxEntries).toHaveLength(1)
    expect(fetchMock.mock.calls.some(call => String(call[0]).includes('/groups/g1/expenses') && (call[1] as RequestInit)?.method === 'POST')).toBe(false)

    Object.defineProperty(navigator, 'onLine', { value: true, configurable: true })
    window.dispatchEvent(new Event('online'))
    await new Promise(resolve => setTimeout(resolve, 0))
    await workspace.syncOutbox()

    expect(outboxEntries).toHaveLength(0)
  })
})

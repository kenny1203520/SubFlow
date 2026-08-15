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
vi.mock('../offline/snapshot', () => ({
  saveSnapshot: vi.fn(async () => {}),
  loadSnapshot: vi.fn(async () => undefined),
  saveGroups: vi.fn(async () => {}),
  loadGroups: vi.fn(async () => undefined),
  saveCurrencies: vi.fn(async () => {}),
  loadCurrencies: vi.fn(async () => undefined),
}))
vi.mock('../offline/outbox', () => ({
  localId: () => `local-${Math.random().toString(36).slice(2)}`,
  isLocalId: (id: string) => id.startsWith('local-'),
  enqueue: vi.fn(async () => ({})),
  listForUser: vi.fn(async () => []),
  remove: vi.fn(async () => {}),
  markFailed: vi.fn(async () => {}),
  updateTargetId: vi.fn(async () => {}),
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
})
afterEach(() => { vi.unstubAllGlobals(); vi.useRealTimers() })

describe('workspace store SSE refresh debounce', () => {
  it('coalesces a burst of onEvent calls into a single refresh pass', async () => {
    const fetchMock = vi.fn().mockImplementation((url: string) => {
      if (url.includes('/groups?perPage')) return Promise.resolve(envelope([{ id: 'g1', name: 'Home', description: '', currency: 'TWD', timezone: 'UTC', color: '#000', ownerId: 'u1', createdAt: '', updatedAt: '' }]))
      if (url.includes('/currencies')) return Promise.resolve(envelope([]))
      if (url.includes('/invitations/pending')) return Promise.resolve(envelope([]))
      if (url.includes('/notifications')) return Promise.resolve(envelope([]))
      if (url.includes('/groups/g1/members')) return Promise.resolve(envelope([]))
      if (url.includes('/groups/g1/subscriptions')) return Promise.resolve(envelope([]))
      if (url.includes('/groups/g1/expenses')) return Promise.resolve(envelope([]))
      if (url.includes('/groups/g1/settlements')) return Promise.resolve(envelope([]))
      if (url.includes('/groups/g1/summary')) return Promise.resolve(envelope({}))
      if (url.includes('/groups/g1/access')) return Promise.resolve(envelope({ permissions: [] }))
      if (url.includes('/subscriptions?perPage')) return Promise.resolve(envelope([]))
      if (url.includes('/expenses?perPage')) return Promise.resolve(envelope([]))
      if (url.includes('/dashboard')) return Promise.resolve(envelope({}))
      if (url.includes('/events')) return Promise.reject(new Error('no sse in test'))
      return Promise.reject(new Error(`unexpected fetch: ${url}`))
    })
    vi.stubGlobal('fetch', fetchMock)

    const workspace = useWorkspaceStore()
    await workspace.selectGroup('g1')

    vi.useFakeTimers()
    fetchMock.mockClear()

    // A burst of 5 events fired back-to-back within the debounce window,
    // mixing resources so the flush must still honor every one of them
    // (loadGroups from the 'groups' event, refreshGroup from the g1-scoped
    // ones) rather than only whatever the last event asked for.
    workspace.onEvent({ type: 'created', groupId: '', resource: 'groups', resourceId: 'g1', occurredAt: '' })
    workspace.onEvent({ type: 'created', groupId: 'g1', resource: 'expenses', resourceId: 'e1', occurredAt: '' })
    workspace.onEvent({ type: 'updated', groupId: 'g1', resource: 'expenses', resourceId: 'e1', occurredAt: '' })
    workspace.onEvent({ type: 'created', groupId: 'g1', resource: 'subscriptions', resourceId: 's1', occurredAt: '' })
    workspace.onEvent({ type: 'created', groupId: 'g1', resource: 'subscriptions', resourceId: 's2', occurredAt: '' })

    // Mid-burst: nothing should have refreshed yet.
    expect(fetchMock).not.toHaveBeenCalled()

    await vi.advanceTimersByTimeAsync(500)

    const dashboardCalls = fetchMock.mock.calls.filter(call => String(call[0]).includes('/dashboard')).length
    const groupsCalls = fetchMock.mock.calls.filter(call => String(call[0]).includes('/groups?perPage')).length
    const summaryCalls = fetchMock.mock.calls.filter(call => String(call[0]).includes('/groups/g1/summary')).length
    expect(dashboardCalls).toBe(1)
    expect(groupsCalls).toBe(1)
    expect(summaryCalls).toBe(1)
  })

  it('does not refresh the current group for an event scoped to a different group', async () => {
    const fetchMock = vi.fn().mockImplementation((url: string) => {
      if (url.includes('/groups?perPage')) return Promise.resolve(envelope([{ id: 'g1', name: 'Home', description: '', currency: 'TWD', timezone: 'UTC', color: '#000', ownerId: 'u1', createdAt: '', updatedAt: '' }]))
      if (url.includes('/currencies')) return Promise.resolve(envelope([]))
      if (url.includes('/invitations/pending')) return Promise.resolve(envelope([]))
      if (url.includes('/notifications')) return Promise.resolve(envelope([]))
      if (url.includes('/groups/g1/members')) return Promise.resolve(envelope([]))
      if (url.includes('/groups/g1/subscriptions')) return Promise.resolve(envelope([]))
      if (url.includes('/groups/g1/expenses')) return Promise.resolve(envelope([]))
      if (url.includes('/groups/g1/settlements')) return Promise.resolve(envelope([]))
      if (url.includes('/groups/g1/summary')) return Promise.resolve(envelope({}))
      if (url.includes('/groups/g1/access')) return Promise.resolve(envelope({ permissions: [] }))
      if (url.includes('/subscriptions?perPage')) return Promise.resolve(envelope([]))
      if (url.includes('/expenses?perPage')) return Promise.resolve(envelope([]))
      if (url.includes('/dashboard')) return Promise.resolve(envelope({}))
      if (url.includes('/events')) return Promise.reject(new Error('no sse in test'))
      return Promise.reject(new Error(`unexpected fetch: ${url}`))
    })
    vi.stubGlobal('fetch', fetchMock)

    const workspace = useWorkspaceStore()
    await workspace.selectGroup('g1')

    vi.useFakeTimers()
    fetchMock.mockClear()

    workspace.onEvent({ type: 'created', groupId: 'g2', resource: 'expenses', resourceId: 'e9', occurredAt: '' })
    await vi.advanceTimersByTimeAsync(500)

    const summaryCalls = fetchMock.mock.calls.filter(call => String(call[0]).includes('/groups/g1/summary')).length
    expect(summaryCalls).toBe(0)
  })
})

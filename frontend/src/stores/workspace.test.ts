// @vitest-environment jsdom
// Online-path happy-path coverage for the store's core actions.
// workspace.offline.test.ts stays focused on the offline-fallback path;
// workspace.sse.test.ts covers the SSE refresh debounce.
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
function failure(status: number, code: string) { return new Response(JSON.stringify({ error: { code, message: code } }), { status }) }

const group = { id: 'g1', name: 'Home', description: '', currency: 'TWD', timezone: 'UTC', color: '#000', ownerId: 'u1', createdAt: '', updatedAt: '' }

// Covers the full fan-out selectGroup/refreshGroup trigger for group "g1".
// Individual tests layer a specific url(+method) => Response override on top
// (checked first) for the one request they're actually exercising. Method is
// optional but matters for e.g. POST /groups vs. the GET /groups/g1/* fan-out
// that a chained selectGroup() triggers afterwards -- both contain the same
// "/groups" substring.
function mockFetch(overrides: Array<[string, () => Response, string?]> = []) {
  const fetchMock = vi.fn().mockImplementation((url: string, init?: RequestInit) => {
    const method = init?.method ?? 'GET'
    for (const [key, handler, wantMethod] of overrides) if (url.includes(key) && (!wantMethod || wantMethod === method)) return Promise.resolve(handler())
    if (url.includes('/groups?perPage')) return Promise.resolve(envelope([group]))
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
  return fetchMock
}

beforeEach(() => {
  pb.authStore.clear()
  pb.authStore.save(validToken(), { id: 'u1', email: 'u1@example.com', collectionId: 'users', collectionName: 'users' })
  setActivePinia(createPinia())
})
afterEach(() => vi.unstubAllGlobals())

describe('workspace store group actions', () => {
  it('createGroup posts the input, prepends the result, and selects it', async () => {
    mockFetch([['/api/subflow/v1/groups', () => envelope(group)]])
    const workspace = useWorkspaceStore()
    const ok = await workspace.createGroup({ name: 'Home', description: '', currency: 'TWD', timezone: 'UTC', color: '#000' })
    expect(ok).toBeUndefined() // createGroup's own run() result isn't returned by the wrapper
    expect(workspace.groups.map(v => v.id)).toContain('g1')
    expect(workspace.currentGroupId).toBe('g1')
  })

  it('updateGroup patches the current group and replaces it in the list', async () => {
    const fetchMock = mockFetch()
    const workspace = useWorkspaceStore()
    await workspace.loadGroups()
    await workspace.selectGroup('g1')
    fetchMock.mockImplementation((url: string) => url.endsWith('/groups/g1') ? Promise.resolve(envelope({ ...group, name: 'Renamed' })) : Promise.resolve(envelope([]))) // simplistic after selection, only updateGroup's own call matters here
    await workspace.updateGroup({ name: 'Renamed', description: '', currency: 'TWD', timezone: 'UTC', color: '#000' })
    expect(workspace.groups.find(v => v.id === 'g1')?.name).toBe('Renamed')
  })

  it('deleteGroup removes the group and clears the current selection', async () => {
    const fetchMock = mockFetch()
    const workspace = useWorkspaceStore()
    await workspace.loadGroups()
    await workspace.selectGroup('g1')
    fetchMock.mockImplementation((url: string) => url.endsWith('/groups/g1') ? Promise.resolve(envelope({ deleted: true })) : Promise.resolve(envelope([])))
    await workspace.deleteGroup()
    expect(workspace.groups.find(v => v.id === 'g1')).toBeUndefined()
    expect(workspace.currentGroupId).toBe('')
  })
})

describe('workspace store expense actions', () => {
  it('addExpense posts to the group and refreshes it', async () => {
    const fetchMock = mockFetch([['/groups/g1/expenses', () => envelope({ id: 'e1' })]])
    const workspace = useWorkspaceStore()
    await workspace.selectGroup('g1')
    fetchMock.mockClear()
    const ok = await workspace.addExpense({ title: 'Lunch', category: '', amountMinor: 500, currency: 'TWD', paidBy: 'u1', notes: '' })
    expect(ok).toBe(true)
    const posted = fetchMock.mock.calls.find(call => String(call[0]).includes('/groups/g1/expenses') && (call[1] as RequestInit)?.method === 'POST')
    expect(posted).toBeTruthy()
  })

  it('updateExpense patches the expense and refreshes the group', async () => {
    const fetchMock = mockFetch([['/expenses/e1', () => envelope({ id: 'e1', title: 'Updated' })]])
    const workspace = useWorkspaceStore()
    await workspace.selectGroup('g1')
    const ok = await workspace.updateExpense('e1', { title: 'Updated', category: '', amountMinor: 500, currency: 'TWD', paidBy: 'u1', notes: '' })
    expect(ok).toBe(true)
    const patched = fetchMock.mock.calls.find(call => String(call[0]).endsWith('/expenses/e1') && (call[1] as RequestInit)?.method === 'PATCH')
    expect(patched).toBeTruthy()
  })

  it('deleteExpense deletes the expense and refreshes the group', async () => {
    const fetchMock = mockFetch([['/expenses/e1', () => envelope({ deleted: true })]])
    const workspace = useWorkspaceStore()
    await workspace.selectGroup('g1')
    await workspace.deleteExpense('e1')
    const deleted = fetchMock.mock.calls.find(call => String(call[0]).endsWith('/expenses/e1') && (call[1] as RequestInit)?.method === 'DELETE')
    expect(deleted).toBeTruthy()
  })
})

describe('workspace store subscription actions', () => {
  it('addSubscription posts to the group and refreshes it', async () => {
    const fetchMock = mockFetch([['/groups/g1/subscriptions', (): Response => envelope({ id: 's1' })]])
    const workspace = useWorkspaceStore()
    await workspace.selectGroup('g1')
    const ok = await workspace.addSubscription({ name: 'Netflix', category: '', amountMinor: 39000, currency: 'TWD', billingCycle: 'monthly', status: 'active', notes: '' })
    expect(ok).toBe(true)
    const posted = fetchMock.mock.calls.find(call => String(call[0]).includes('/groups/g1/subscriptions') && (call[1] as RequestInit)?.method === 'POST')
    expect(posted).toBeTruthy()
  })

  it('updateSubscription patches the subscription', async () => {
    const fetchMock = mockFetch([['/subscriptions/s1', () => envelope({ id: 's1', name: 'Renamed' })]])
    const workspace = useWorkspaceStore()
    await workspace.selectGroup('g1')
    const ok = await workspace.updateSubscription('s1', { name: 'Renamed', category: '', amountMinor: 39000, currency: 'TWD', billingCycle: 'monthly', status: 'active', notes: '' })
    expect(ok).toBe(true)
    const patched = fetchMock.mock.calls.find(call => String(call[0]).endsWith('/subscriptions/s1') && (call[1] as RequestInit)?.method === 'PATCH')
    expect(patched).toBeTruthy()
  })

  it('deleteSubscription deletes the subscription', async () => {
    const fetchMock = mockFetch([['/subscriptions/s1', () => envelope({ deleted: true })]])
    const workspace = useWorkspaceStore()
    await workspace.selectGroup('g1')
    await workspace.deleteSubscription('s1')
    const deleted = fetchMock.mock.calls.find(call => String(call[0]).endsWith('/subscriptions/s1') && (call[1] as RequestInit)?.method === 'DELETE')
    expect(deleted).toBeTruthy()
  })

  it('stopSubscription posts the end date and refreshes personal + group scopes', async () => {
    const fetchMock = mockFetch([['/subscriptions/s1/stop', () => envelope({ id: 's1', status: 'active' })]])
    const workspace = useWorkspaceStore()
    await workspace.selectGroup('g1')
    await workspace.stopSubscription('s1', '2026-12-31')
    const posted = fetchMock.mock.calls.find(call => String(call[0]).endsWith('/subscriptions/s1/stop') && (call[1] as RequestInit)?.method === 'POST')
    expect(posted).toBeTruthy()
    expect(JSON.parse(String((posted?.[1] as RequestInit)?.body))).toEqual({ endsOn: '2026-12-31' })
  })
})

describe('workspace store category and currency actions', () => {
  it('createCategory posts and appends the created category', async () => {
    mockFetch([['/categories', () => envelope({ id: 'c1', scope: 'personal', customName: 'Snacks' })]])
    const workspace = useWorkspaceStore()
    const created = await workspace.createCategory('personal', 'Snacks')
    expect(created.id).toBe('c1')
    expect(workspace.categories.map(v => v.id)).toContain('c1')
  })

  it('updateCategory patches and replaces the category in place', async () => {
    mockFetch([['/categories', () => envelope({ id: 'c1', scope: 'personal', customName: 'Snacks' })]])
    const workspace = useWorkspaceStore()
    await workspace.createCategory('personal', 'Snacks')
    mockFetch([['/categories/c1', () => envelope({ id: 'c1', scope: 'personal', customName: 'Treats' })]])
    const updated = await workspace.updateCategory('c1', { customName: 'Treats', iconKey: 'tag' })
    expect(updated.customName).toBe('Treats')
    expect(workspace.categories.find(v => v.id === 'c1')?.customName).toBe('Treats')
  })

  it('archiveCategory deletes it and removes it from the list', async () => {
    mockFetch([['/categories', () => envelope({ id: 'c1', scope: 'personal', customName: 'Snacks' })]])
    const workspace = useWorkspaceStore()
    await workspace.createCategory('personal', 'Snacks')
    mockFetch([['/categories/c1', () => envelope({ deleted: true })]])
    await workspace.archiveCategory('c1')
    expect(workspace.categories.find(v => v.id === 'c1')).toBeUndefined()
  })

  it('quoteRate returns the quoted rate for the given currency pair and date', async () => {
    const rate = { baseCurrency: 'USD', quoteCurrency: 'TWD', rate: '31.5', effectiveDate: '2026-08-01', provider: 'test', fetchedAt: '2026-08-01T00:00:00Z' }
    mockFetch([['/exchange-rates/quote', () => envelope(rate)]])
    const workspace = useWorkspaceStore()
    const quote = await workspace.quoteRate('USD', 'TWD', '2026-08-01')
    expect(quote).toEqual(rate)
  })

  it('previewGroupCurrency and changeGroupCurrency both scope to the current group', async () => {
    const expectedPreview = { from: 'TWD', to: 'USD', affected: 0, missing: [] }
    const fetchMock = mockFetch([
      ['/currency-change/preview', () => envelope(expectedPreview)],
      ['/currency-change', () => envelope({ ...group, currency: 'USD' })],
    ])
    const workspace = useWorkspaceStore()
    await workspace.loadGroups()
    await workspace.selectGroup('g1')
    const preview = await workspace.previewGroupCurrency('USD')
    expect(preview).toEqual(expectedPreview)
    const changed = await workspace.changeGroupCurrency('USD')
    expect(changed.currency).toBe('USD')
    expect(workspace.groups.find(v => v.id === 'g1')?.currency).toBe('USD')
    const previewCall = fetchMock.mock.calls.find(call => String(call[0]).includes('/groups/g1/currency-change/preview'))
    const changeCall = fetchMock.mock.calls.find(call => String(call[0]).includes('/groups/g1/currency-change') && !String(call[0]).includes('preview'))
    expect(previewCall).toBeTruthy()
    expect(changeCall).toBeTruthy()
  })
})

describe('workspace store run()/retryLast() error wrapper', () => {
  it('a failed action surfaces localizedError and leaves the group list unchanged', async () => {
    mockFetch([['/api/subflow/v1/groups', () => failure(409, 'conflict'), 'POST']])
    const workspace = useWorkspaceStore()
    await workspace.createGroup({ name: 'Home', description: '', currency: 'TWD', timezone: 'UTC', color: '#000' })
    expect(workspace.error).toBe('conflict')
    expect(workspace.groups).toHaveLength(0)
  })

  it('retryLast re-invokes the same failed action and can succeed once the backend recovers', async () => {
    let attempt = 0
    const fetchMock = mockFetch([['/api/subflow/v1/groups', () => attempt++ === 0 ? failure(409, 'conflict') : envelope(group), 'POST']])
    const workspace = useWorkspaceStore()
    await workspace.createGroup({ name: 'Home', description: '', currency: 'TWD', timezone: 'UTC', color: '#000' })
    expect(workspace.error).toBe('conflict')

    await workspace.retryLast()
    expect(workspace.error).toBe('')
    expect(workspace.groups.map(v => v.id)).toContain('g1')
    const postCalls = fetchMock.mock.calls.filter(call => String(call[0]) === '/api/subflow/v1/groups' && (call[1] as RequestInit)?.method === 'POST')
    expect(postCalls).toHaveLength(2)
  })
})

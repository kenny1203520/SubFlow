import { beforeEach, describe, expect, it, vi } from 'vitest'

const memory = new Map<string, Map<string, unknown>>()
vi.mock('./db', () => ({
  get: vi.fn(async (store: string, key: string) => memory.get(store)?.get(key)),
  set: vi.fn(async (store: string, key: string, value: unknown) => { if (!memory.has(store)) memory.set(store, new Map()); memory.get(store)!.set(key, value) }),
  del: vi.fn(async (store: string, key: string) => { memory.get(store)?.delete(key) }),
  getAll: vi.fn(async (store: string) => Array.from(memory.get(store)?.values() ?? [])),
}))

import { loadCurrencies, loadGroups, loadSnapshot, saveCurrencies, saveGroups, saveSnapshot } from './snapshot'

beforeEach(() => memory.clear())

describe('snapshot', () => {
  it('round-trips a bundle for a scope', async () => {
    await saveSnapshot('u1', 'personal', '', { expenses: [{ id: 'e1' } as never], subscriptions: [] })
    const loaded = await loadSnapshot('u1', 'personal', '')
    expect(loaded?.expenses).toHaveLength(1)
    expect(loaded?.expenses[0].id).toBe('e1')
  })

  it('keeps personal and group snapshots separate', async () => {
    await saveSnapshot('u1', 'personal', '', { expenses: [], subscriptions: [] })
    await saveSnapshot('u1', 'group', 'g1', { expenses: [{ id: 'g-e1' } as never], subscriptions: [], settlements: [], members: [] })
    expect((await loadSnapshot('u1', 'personal', ''))?.expenses).toHaveLength(0)
    expect((await loadSnapshot('u1', 'group', 'g1'))?.expenses).toHaveLength(1)
  })

  it('keeps different users\' snapshots separate for the same scope', async () => {
    await saveSnapshot('u1', 'group', 'g1', { expenses: [{ id: 'u1-e' } as never], subscriptions: [] })
    await saveSnapshot('u2', 'group', 'g1', { expenses: [{ id: 'u2-e' } as never], subscriptions: [] })
    expect((await loadSnapshot('u1', 'group', 'g1'))?.expenses[0].id).toBe('u1-e')
    expect((await loadSnapshot('u2', 'group', 'g1'))?.expenses[0].id).toBe('u2-e')
  })

  it('returns undefined when nothing was ever saved', async () => {
    expect(await loadSnapshot('u1', 'group', 'never-saved')).toBeUndefined()
  })

  it('round-trips the group list separately from scope bundles', async () => {
    await saveGroups('u1', [{ id: 'g1', name: 'Home', description: '', currency: 'TWD', timezone: 'UTC', color: '#000', ownerId: 'u1', createdAt: '', updatedAt: '' }])
    expect((await loadGroups('u1'))?.[0].name).toBe('Home')
    expect(await loadGroups('u2')).toBeUndefined()
  })

  it('round-trips the currency list', async () => {
    await saveCurrencies('u1', [{ code: 'TWD', digits: 2 }])
    expect((await loadCurrencies('u1'))?.[0].code).toBe('TWD')
  })
})

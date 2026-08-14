import { beforeEach, describe, expect, it, vi } from 'vitest'

const memory = new Map<string, Map<string, unknown>>()
vi.mock('./db', () => ({
  get: vi.fn(async (store: string, key: string) => memory.get(store)?.get(key)),
  set: vi.fn(async (store: string, key: string, value: unknown) => { if (!memory.has(store)) memory.set(store, new Map()); memory.get(store)!.set(key, value) }),
  del: vi.fn(async (store: string, key: string) => { memory.get(store)?.delete(key) }),
  getAll: vi.fn(async (store: string) => Array.from(memory.get(store)?.values() ?? [])),
}))

import { enqueue, isLocalId, listForUser, localId, markFailed, remove, updateTargetId } from './outbox'

beforeEach(() => memory.clear())

describe('outbox', () => {
  it('generates local ids that are recognizable and unique', () => {
    const a = localId(), b = localId()
    expect(isLocalId(a)).toBe(true)
    expect(isLocalId('real-record-id')).toBe(false)
    expect(a).not.toBe(b)
  })

  it('lists only the current user\'s entries, oldest first', async () => {
    const target = localId()
    await enqueue({ userId: 'u1', kind: 'expense', op: 'create', scope: 'personal', groupId: '', targetId: target, payload: { title: 'first' } })
    await enqueue({ userId: 'u2', kind: 'expense', op: 'create', scope: 'personal', groupId: '', targetId: localId(), payload: {} })
    await enqueue({ userId: 'u1', kind: 'expense', op: 'update', scope: 'personal', groupId: '', targetId: target, payload: { title: 'second' } })
    const entries = await listForUser('u1')
    expect(entries).toHaveLength(2)
    expect(entries[0].op).toBe('create')
    expect(entries[1].op).toBe('update')
  })

  it('marks an entry failed without dropping it', async () => {
    const entry = await enqueue({ userId: 'u1', kind: 'settlement', op: 'delete', scope: 'group', groupId: 'g1', targetId: 'set-1' })
    await markFailed(entry.id, 'conflict')
    const [reloaded] = await listForUser('u1')
    expect(reloaded.status).toBe('failed')
    expect(reloaded.error).toBe('conflict')
  })

  it('removes an entry once synced', async () => {
    const entry = await enqueue({ userId: 'u1', kind: 'expense', op: 'create', scope: 'personal', groupId: '', targetId: localId(), payload: {} })
    await remove(entry.id)
    expect(await listForUser('u1')).toHaveLength(0)
  })

  it('repoints a queued entry\'s targetId once its local id resolves to a real one', async () => {
    const target = localId()
    const entry = await enqueue({ userId: 'u1', kind: 'expense', op: 'update', scope: 'personal', groupId: '', targetId: target, payload: {} })
    await updateTargetId(entry.id, 'real-id-123')
    const [reloaded] = await listForUser('u1')
    expect(reloaded.targetId).toBe('real-id-123')
  })
})

import * as db from './db'

export type OutboxKind = 'expense' | 'subscription' | 'settlement'
export type OutboxOp = 'create' | 'update' | 'delete'
export type OutboxScope = 'personal' | 'group'

// One not-yet-synced mutation. targetId is the record it acts on — a
// local-prefixed id for a create that hasn't round-tripped to the server
// yet, or the real id for an update/delete on an already-synced record.
// payload is the create/update request body; omitted for delete.
export interface OutboxEntry {
  id: string
  userId: string
  kind: OutboxKind
  op: OutboxOp
  scope: OutboxScope
  groupId: string
  targetId: string
  payload?: unknown
  createdAt: string
  status: 'pending' | 'failed'
  error?: string
}

// Local ids stand in for a server id between an offline create and its sync.
// Prefixed so the rest of the app can tell "this row isn't real yet" apart
// from a real PocketBase id at a glance, without a separate flag everywhere.
export function localId(): string { return `local-${crypto.randomUUID()}` }
export function isLocalId(id: string): boolean { return id.startsWith('local-') }

export async function enqueue(entry: Omit<OutboxEntry, 'id' | 'createdAt' | 'status'>): Promise<OutboxEntry> {
  const full: OutboxEntry = { ...entry, id: crypto.randomUUID(), createdAt: new Date().toISOString(), status: 'pending' }
  await db.set('outbox', full.id, full)
  return full
}

// FIFO by creation time so a later update to a record replays after the
// create that produced it, and so on — outbox entries for one record must
// stay in the order the user made them.
export async function listForUser(userId: string): Promise<OutboxEntry[]> {
  const all = await db.getAll<OutboxEntry>('outbox')
  return all.filter(entry => entry.userId === userId).sort((a, b) => a.createdAt.localeCompare(b.createdAt))
}

export async function remove(id: string): Promise<void> {
  await db.del('outbox', id)
}

export async function markFailed(id: string, error: string): Promise<void> {
  const entry = await db.get<OutboxEntry>('outbox', id)
  if (!entry) return
  await db.set('outbox', id, { ...entry, status: 'failed', error })
}

export async function updateTargetId(id: string, targetId: string): Promise<void> {
  const entry = await db.get<OutboxEntry>('outbox', id)
  if (!entry) return
  await db.set('outbox', id, { ...entry, targetId })
}

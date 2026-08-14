import type { CurrencyInfo, Expense, Group, Membership, Settlement, Subscription } from '../api/types'
import * as db from './db'

export type SnapshotScope = 'personal' | 'group'

// What one scope (personal, or a specific group) needs to render its
// expense/subscription/settlement views while offline. Settlements and
// members only exist for a group scope; the store never populates them for
// 'personal'.
export interface SnapshotBundle {
  expenses: Expense[]
  subscriptions: Subscription[]
  settlements?: Settlement[]
  members?: Membership[]
}

// Snapshots are keyed by user so that switching accounts on the same device
// (or logging out and a different person logging in) never shows the
// previous person's cached data before a real refresh completes.
function key(userId: string, scope: SnapshotScope, groupId: string): string {
  return scope === 'personal' ? `${userId}:personal` : `${userId}:group:${groupId}`
}

export async function saveSnapshot(userId: string, scope: SnapshotScope, groupId: string, bundle: SnapshotBundle): Promise<void> {
  await db.set('snapshots', key(userId, scope, groupId), bundle)
}

export async function loadSnapshot(userId: string, scope: SnapshotScope, groupId: string): Promise<SnapshotBundle | undefined> {
  return db.get<SnapshotBundle>('snapshots', key(userId, scope, groupId))
}

// The group list itself (name/currency/timezone/color) is cached separately
// from the per-scope bundle above — group views need it to render at all
// (currency formatting, timezone labels), independent of which group's
// expenses/subscriptions are being shown.
export async function saveGroups(userId: string, groups: Group[]): Promise<void> {
  await db.set('snapshots', `${userId}:groups`, groups)
}

export async function loadGroups(userId: string): Promise<Group[] | undefined> {
  return db.get<Group[]>('snapshots', `${userId}:groups`)
}

// Currency list rarely changes but is needed to render the amount/currency
// picker on the create-expense/subscription forms — without it, those forms
// would have nothing to offer for currency while offline.
export async function saveCurrencies(userId: string, currencies: CurrencyInfo[]): Promise<void> {
  await db.set('snapshots', `${userId}:currencies`, currencies)
}

export async function loadCurrencies(userId: string): Promise<CurrencyInfo[] | undefined> {
  return db.get<CurrencyInfo[]>('snapshots', `${userId}:currencies`)
}

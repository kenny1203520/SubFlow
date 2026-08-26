import type { CurrencyInfo, DailyLedger, Expense, Group, Income, Membership, Settlement, Subscription } from '../api/types'
import * as db from './db'

export type SnapshotScope = 'personal' | 'group'

export interface SnapshotBundle {
  expenses: Expense[]
  incomes?: Income[]
  subscriptions: Subscription[]
  settlements?: Settlement[]
  members?: Membership[]
}

function key(userId: string, scope: SnapshotScope, groupId: string): string {
  return scope === 'personal' ? `${userId}:personal` : `${userId}:group:${groupId}`
}

export async function saveSnapshot(userId: string, scope: SnapshotScope, groupId: string, bundle: SnapshotBundle): Promise<void> {
  await db.set('snapshots', key(userId, scope, groupId), bundle)
}
export async function loadSnapshot(userId: string, scope: SnapshotScope, groupId: string): Promise<SnapshotBundle | undefined> {
  return db.get<SnapshotBundle>('snapshots', key(userId, scope, groupId))
}
export async function saveGroups(userId: string, groups: Group[]): Promise<void> {
  await db.set('snapshots', `${userId}:groups`, groups)
}
export async function loadGroups(userId: string): Promise<Group[] | undefined> {
  return db.get<Group[]>('snapshots', `${userId}:groups`)
}
export async function saveCurrencies(userId: string, currencies: CurrencyInfo[]): Promise<void> {
  await db.set('snapshots', `${userId}:currencies`, currencies)
}
export async function loadCurrencies(userId: string): Promise<CurrencyInfo[] | undefined> {
  return db.get<CurrencyInfo[]>('snapshots', `${userId}:currencies`)
}
export async function saveLedger(userId: string, ledger: DailyLedger): Promise<void> {
  await db.set('snapshots', `${userId}:ledger:${ledger.date}`, ledger)
}
export async function loadLedger(userId: string, date: string): Promise<DailyLedger | undefined> {
  return db.get<DailyLedger>('snapshots', `${userId}:ledger:${date}`)
}

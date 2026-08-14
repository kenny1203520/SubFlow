// Minimal IndexedDB wrapper — just enough to store keyed JSON values in a
// couple of object stores, with no schema beyond that. Deliberately not a
// general-purpose library: offline/snapshot.ts and offline/outbox.ts are the
// only callers, and both just need get/set/del/getAll on a named store.
const DB_NAME = 'subflow-offline'
const DB_VERSION = 1
const STORES = ['snapshots', 'outbox'] as const
export type Store = typeof STORES[number]

let dbPromise: Promise<IDBDatabase> | undefined

function openDb(): Promise<IDBDatabase> {
  if (!dbPromise) dbPromise = new Promise((resolve, reject) => {
    const request = indexedDB.open(DB_NAME, DB_VERSION)
    request.onupgradeneeded = () => { const db = request.result; for (const store of STORES) if (!db.objectStoreNames.contains(store)) db.createObjectStore(store) }
    request.onsuccess = () => resolve(request.result)
    request.onerror = () => reject(request.error)
  })
  return dbPromise
}

async function run<T>(store: Store, mode: IDBTransactionMode, fn: (objectStore: IDBObjectStore) => IDBRequest<T>): Promise<T> {
  const db = await openDb()
  return new Promise((resolve, reject) => {
    const request = fn(db.transaction(store, mode).objectStore(store))
    request.onsuccess = () => resolve(request.result)
    request.onerror = () => reject(request.error)
  })
}

export const get = <T>(store: Store, key: string): Promise<T | undefined> => run<T | undefined>(store, 'readonly', s => s.get(key))
// Callers pass Vue reactive arrays/objects straight from the store — those
// are Proxy-wrapped, and IndexedDB's structured-clone algorithm throws
// DataCloneError on a Proxy ("[object Array] could not be cloned") rather
// than silently unwrapping it. A JSON round-trip is a cheap, universal fix
// since everything ever stored here (Expense/Subscription/Settlement/Group/
// CurrencyInfo/OutboxEntry) is already plain JSON-shaped data.
export const set = <T>(store: Store, key: string, value: T): Promise<void> => run(store, 'readwrite', s => s.put(JSON.parse(JSON.stringify(value)), key)).then(() => undefined)
export const del = (store: Store, key: string): Promise<void> => run(store, 'readwrite', s => s.delete(key)).then(() => undefined)
export const getAll = <T>(store: Store): Promise<T[]> => run<T[]>(store, 'readonly', s => s.getAll())

import { computed, reactive, ref } from 'vue'
import { defineStore } from 'pinia'
import { ApiClient, ApiError } from '../api/client'
import { SSEClient } from '../api/sse'
import type { AccessRole, AuditLog, BillingDates, Category, Currency, CurrencyChangePreview, CurrencyInfo, DashboardSummary, Envelope, ExchangeRate, Expense, Group, GroupAccess, Invitation, Membership, Meta, Notification, OwnershipTransfer, Settlement, Subscription, SubscriptionPeriods, SubFlowEvent } from '../api/types'
import { useAuthStore } from './auth'
import { useToastStore } from './toast'
import { useI18n } from '../i18n'
import * as outbox from '../offline/outbox'
import type { OutboxEntry, OutboxKind, OutboxScope } from '../offline/outbox'
import * as snapshotStore from '../offline/snapshot'

type GroupInput = Pick<Group, 'name' | 'description' | 'currency' | 'timezone' | 'color'>
// startsOn is optional so an update can omit it when the user did not touch
// the date input: the backend then keeps the exact stored instant instead of
// a same-day reconstruction from the viewer's timezone, which would shift it
// by up to 24h relative to the original (see dateInput.ts).
type SubscriptionInput = Pick<Subscription, 'name'|'category'|'amountMinor'|'currency'|'billingCycle'|'status'|'notes'> & Partial<Pick<Subscription,'paidBy'|'endsOn'|'nextBilling'|'categoryId'|'rateMode'|'exchangeRate'|'billingInterval'|'startsOn'|'splitMode'|'splits'|'revisionScope'|'effectiveBillingAt'|'endBillingAt'>>
// incurredOn is optional for the same reason startsOn is on SubscriptionInput.
type ExpenseInput = Pick<Expense, 'title'|'category'|'amountMinor'|'currency'|'paidBy'|'notes'> & Partial<Pick<Expense,'splitMode'|'splits'|'categoryId'|'rateMode'|'exchangeRate'|'incurredOn'>>

export const useWorkspaceStore = defineStore('workspace', () => {
  const auth = useAuthStore()
  const toast = useToastStore()
  const { tr, locale } = useI18n()
  const api = new ApiClient(() => auth.token, auth.logout)
  const groups = ref<Group[]>([])
  const currencies = ref<CurrencyInfo[]>([])
  const categories = ref<Category[]>([])
  const currentGroupId = ref('')
  const members = ref<Membership[]>([])
  const invitations = ref<Invitation[]>([])
  const invitationsMeta = ref<Meta>({ page:1, perPage:25, totalItems:0, totalPages:0 })
  const pendingInvitations = ref<Invitation[]>([])
  const notifications = ref<Notification[]>([])
  const subscriptions = ref<Subscription[]>([])
  const expenses = ref<Expense[]>([])
  const settlements = ref<Settlement[]>([])
  const groupRoles = ref<AccessRole[]>([])
  const ownershipTransfer = ref<OwnershipTransfer>()
  const groupAuditLogs = ref<AuditLog[]>([])
  const groupAuditMeta = ref<Meta>({ page:1, perPage:25, totalItems:0, totalPages:0 })
  const groupPermissions = ref<string[]>([])
  const groupErrors = reactive<Record<'access'|'members'|'subscriptions'|'expenses'|'settlements'|'summary', string>>({ access:'', members:'', subscriptions:'', expenses:'', settlements:'', summary:'' })
  const groupBusy = reactive<Record<'access'|'members'|'subscriptions'|'expenses'|'settlements'|'summary', number>>({ access:0, members:0, subscriptions:0, expenses:0, settlements:0, summary:0 })
  const personalSubscriptions = ref<Subscription[]>([])
  const personalExpenses = ref<Expense[]>([])
  const personalSummary = ref<DashboardSummary | null>(null)
  const summary = ref<DashboardSummary | null>(null)
  const busy = reactive<Record<string, number>>({})
  const loading = computed(() => Object.values(busy).some(value => value > 0))
  const error = ref('')
  const errorCode = ref('')
  const permissionDenied = ref(false)
  const localizedError = computed(() => {
    const keys: Record<string, Parameters<typeof tr>[0]> = { network_error: 'networkError', invalid_response: 'invalidResponse', request_failed: 'requestFailed', invalid_request: 'invalidRequest', conflict: 'conflict', not_found: 'notFound', internal_error: 'internalError', forbidden: 'forbiddenError', rate_unavailable:'rateUnavailable' }
    return errorCode.value && keys[errorCode.value] ? tr(keys[errorCode.value]) : error.value || tr('unexpectedError')
  })
  const currentGroup = computed(() => groups.value.find(value => value.id === currentGroupId.value))
  const currentMembership = computed(() => members.value.find(value => value.userId === auth.record?.id))
  const isOwner = computed(() => currentMembership.value?.role === 'owner')
  // Drives the topbar sync button's danger styling — a record stuck with a
  // failed sync needs the same visibility across any scope the user happens
  // to be viewing, not just the one it lives in.
  const hasSyncErrors = computed(() => [expenses, subscriptions, settlements, personalExpenses, personalSubscriptions].some(list => list.value.some(item => item.syncError)))
  let sse: SSEClient | undefined
  let sseStarted = false
  let loadedGroupId = ''
  let groupRequest = 0
  let hydratingGroupId = ''
  let groupHydration: Promise<void> | undefined
  let lastRetry: (() => Promise<void>) | undefined
  // navigator.onLine is a best-effort signal (it can say "online" on a dead
  // network), but it's what lets a mutation skip straight to the offline
  // path instead of waiting out a fetch timeout first.
  const online = ref(typeof navigator === 'undefined' || navigator.onLine)
  const outboxPending = ref(0)
  let syncing = false
  async function refreshOutboxPending() { outboxPending.value = auth.record ? (await outbox.listForUser(auth.record.id)).length : 0 }
  if (typeof window !== 'undefined') {
    window.addEventListener('online', () => { online.value = true; void syncOutbox() })
    window.addEventListener('offline', () => { online.value = false })
  }

  function resourceError(reason: unknown) {
    const code = reason instanceof ApiError ? reason.code : 'internal_error'
    const keys: Record<string, Parameters<typeof tr>[0]> = { network_error:'networkError', invalid_response:'invalidResponse', request_failed:'requestFailed', invalid_request:'invalidRequest', conflict:'conflict', not_found:'notFound', internal_error:'internalError', forbidden:'forbiddenError', rate_unavailable:'rateUnavailable' }
    return keys[code] ? tr(keys[code]) : tr('requestFailed')
  }

  // Returns whether the task succeeded, so a caller that needs to react to a
  // failure (e.g. keep a form drawer open and show the error) can check it
  // instead of assuming success like a bare `await run(...)` always did.
  // `notify` toasts the outcome; background reads (loadGroups, refreshPersonal,
  // refreshDashboard) pass false since the user did not just trigger them.
  async function run(task: () => Promise<void>, key = 'general', notify = true): Promise<boolean> {
    busy[key] = (busy[key] || 0) + 1
    lastRetry = task
    error.value = ''
    errorCode.value = ''
    permissionDenied.value = false
    try {
      await task()
      if (notify) toast.push('success', tr('actionSucceeded'))
      return true
    } catch (reason) {
      const value = reason as { status?: number; message?: string }
      permissionDenied.value = value.status === 403
      error.value = value.message ?? ''
      errorCode.value = reason instanceof ApiError ? reason.code : 'internal_error'
      if (notify) toast.push('error', tr('actionFailed', { reason: localizedError.value }))
      return false
    } finally {
      busy[key] = Math.max(0, (busy[key] || 1) - 1)
    }
  }

  async function retryLast() { if (lastRetry) await run(lastRetry, 'retry') }

  // Tries the real mutation first; only falls back to the local/outbox path
  // when the browser is known offline or the request specifically failed to
  // reach the server at all (ApiError code network_error) — any other
  // failure (validation, permission, conflict) is a real error and must
  // still surface normally through run()'s catch, not be treated as "queue
  // it for later".
  async function withOfflineFallback(attempt: () => Promise<void>, applyOffline: () => Promise<void>): Promise<void> {
    if (online.value) {
      try { await attempt(); return } catch (reason) { if (!(reason instanceof ApiError) || reason.code !== 'network_error') throw reason }
    }
    await applyOffline()
    await refreshOutboxPending()
  }

  // Server-computed fields (base currency conversion, resolved splits, …)
  // aren't knowable client-side, so the optimistic record just mirrors the
  // input verbatim for those — pendingSync is what tells the UI this is a
  // provisional value, not the final one syncOutbox() will replace it with.
  function localExpense(id: string, input: ExpenseInput, groupId?: string): Expense {
    const now = new Date().toISOString()
    return { id, groupId, ownerId: groupId ? undefined : auth.record?.id, title: input.title, category: input.category, categoryId: input.categoryId, amountMinor: input.amountMinor, currency: input.currency, baseCurrency: input.currency, baseAmountMinor: input.amountMinor, exchangeRate: '1', exchangeRateDate: now, rateMode: 'automatic', paidBy: input.paidBy, incurredOn: input.incurredOn || now, notes: input.notes, splitMode: input.splitMode, splits: input.splits, createdAt: now, updatedAt: now, pendingSync: true }
  }
  function localSubscription(id: string, input: SubscriptionInput, groupId?: string): Subscription {
    const now = new Date().toISOString()
    const startsOn = input.startsOn || now
    return { id, groupId, ownerId: groupId ? undefined : auth.record?.id, paidBy: input.paidBy || auth.record?.id || '', name: input.name, category: input.category, categoryId: input.categoryId, amountMinor: input.amountMinor, currency: input.currency, baseCurrency: input.currency, baseAmountMinor: input.amountMinor, exchangeRate: '1', exchangeRateDate: now, rateMode: 'automatic', billingCycle: input.billingCycle, billingInterval: input.billingInterval, startsOn, nextBilling: input.nextBilling || startsOn, status: input.status, notes: input.notes, splitMode: input.splitMode, splits: input.splits, createdAt: now, updatedAt: now, pendingSync: true }
  }
  // Every offline mutation to personalExpenses/personalSubscriptions must
  // persist the result to the snapshot cache immediately, not just update
  // the in-memory ref: submit() handlers unconditionally call
  // refreshPersonal() right after a successful create/update/delete, and
  // while offline that reloads straight from the cached snapshot (see
  // refreshPersonal's catch branch) — a stale cache would silently revert
  // the optimistic change the user just made one line earlier.
  async function savePersonalSnapshot() {
    const userId = auth.record?.id
    if (userId) await snapshotStore.saveSnapshot(userId, 'personal', '', { expenses: personalExpenses.value, subscriptions: personalSubscriptions.value })
  }
  function localSettlement(id: string, input: Pick<Settlement,'fromUserId'|'toUserId'|'amountMinor'|'settledOn'|'notes'>, groupId: string): Settlement {
    const now = new Date().toISOString()
    const currency = (currentGroup.value?.currency || 'TWD') as Currency
    return { id, groupId, fromUserId: input.fromUserId, toUserId: input.toUserId, createdBy: auth.record?.id || '', amountMinor: input.amountMinor, currency, baseCurrency: currency, baseAmountMinor: input.amountMinor, exchangeRate: '1', exchangeRateDate: now, settledOn: input.settledOn || now, notes: input.notes, createdAt: now, updatedAt: now, pendingSync: true }
  }

  // Deleting a record that was itself never synced (still a local- id) has
  // nothing to tell the server — cancel whatever create/update chain is
  // still queued for it instead of queuing a delete for a record the
  // backend has never heard of.
  async function localDelete(kind: OutboxKind, scope: OutboxScope, groupId: string, id: string): Promise<void> {
    const userId = auth.record?.id
    if (!userId) return
    if (outbox.isLocalId(id)) {
      const entries = await outbox.listForUser(userId)
      await Promise.all(entries.filter(entry => entry.targetId === id).map(entry => outbox.remove(entry.id)))
    } else {
      await outbox.enqueue({ userId, kind, op: 'delete', scope, groupId, targetId: id })
    }
  }

  async function loadGroups() {
    const userId = auth.record?.id
    await run(async () => {
      try {
        if (!online.value) throw new ApiError(0, 'network_error', 'offline')
        const [groupResult,currencyResult]=await Promise.all([api.get<Group[]>('/groups?perPage=100'),api.get<CurrencyInfo[]>('/currencies')])
        groups.value = groupResult.data
        currencies.value = currencyResult.data
        if (userId) { await snapshotStore.saveGroups(userId, groups.value); await snapshotStore.saveCurrencies(userId, currencies.value) }
        try { await loadInvitationInbox() } catch { pendingInvitations.value=[]; notifications.value=[] }
      } catch (reason) {
        if (!(reason instanceof ApiError) || reason.code !== 'network_error') throw reason
        const cachedGroups = userId ? await snapshotStore.loadGroups(userId) : undefined
        if (!cachedGroups) throw reason
        groups.value = cachedGroups
        currencies.value = (userId ? await snapshotStore.loadCurrencies(userId) : undefined) || currencies.value
      }
      if (currentGroupId.value && !groups.value.some(group => group.id === currentGroupId.value)) currentGroupId.value = ''
      if (!sseStarted) { sse = new SSEClient(() => auth.token, onEvent, auth.logout, () => { online.value = true; void syncOutbox() }); sseStarted = true; void sse.start() }
      if (userId) { await refreshOutboxPending(); if (online.value) void syncOutbox() }
    }, 'groups', false)
  }

  async function loadCategories(scope:'personal'|'group',groupId='') { categories.value=(await api.get<Category[]>(`/categories?scope=${scope}${groupId?`&groupId=${encodeURIComponent(groupId)}`:''}`)).data }
  async function createCategory(scope:'personal'|'group',customName:string,groupId='',iconKey='tag'){const value=(await api.post<Category>('/categories',{scope,customName,groupId,iconKey})).data;categories.value.push(value);return value}
  async function updateCategory(id:string, value:Pick<Category,'customName'|'iconKey'>){const updated=(await api.patch<Category>(`/categories/${id}`,value)).data;categories.value=categories.value.map(item=>item.id===id?updated:item);return updated}
  async function archiveCategory(id:string){await api.delete(`/categories/${id}`);categories.value=categories.value.filter(v=>v.id!==id)}
  async function quoteRate(from:Currency,to:Currency,date:string){return (await api.get<ExchangeRate>(`/exchange-rates/quote?from=${encodeURIComponent(from)}&to=${encodeURIComponent(to)}&date=${encodeURIComponent(date)}`)).data}
  async function previewGroupCurrency(currency:Currency){return (await api.post<CurrencyChangePreview>(`/groups/${currentGroupId.value}/currency-change/preview`,{currency})).data}
  async function changeGroupCurrency(currency:Currency){const group=(await api.post<Group>(`/groups/${currentGroupId.value}/currency-change`,{currency})).data;groups.value=groups.value.map(v=>v.id===group.id?group:v);await refreshGroup();return group}

  async function selectGroup(id: string) {
    if (id && loadedGroupId === id) return
    if (id && hydratingGroupId === id && groupHydration) return groupHydration
    currentGroupId.value = id
    loadedGroupId = ''
    const request = ++groupRequest
    if (!id) {
      members.value = []
      invitations.value = []
      invitationsMeta.value = { page:1, perPage:25, totalItems:0, totalPages:0 }
      subscriptions.value = []
      expenses.value = []
      summary.value = null
      groupPermissions.value = []
      groupAuditLogs.value = []
      groupAuditMeta.value = { page:1, perPage:25, totalItems:0, totalPages:0 }
      return
    }
    members.value = []
    invitations.value = []
    invitationsMeta.value = { page:1, perPage:25, totalItems:0, totalPages:0 }
    subscriptions.value = []
    expenses.value = []
    settlements.value = []
    summary.value = null
    groupPermissions.value = []
    groupAuditLogs.value = []
    groupAuditMeta.value = { page:1, perPage:25, totalItems:0, totalPages:0 }
    for (const key of Object.keys(groupErrors) as Array<keyof typeof groupErrors>) groupErrors[key] = ''
    hydratingGroupId = id
    groupHydration = refreshGroup(request).finally(() => { if (hydratingGroupId === id) { hydratingGroupId='';groupHydration=undefined } })
    await groupHydration
  }

  async function refreshGroup(expectedRequest = ++groupRequest) {
    if (!currentGroupId.value) return
    const id = currentGroupId.value
    const userId = auth.record?.id
    // Read once up front rather than per-resource: it's the same bundle
    // either way, and reading it once keeps a burst of concurrent tab
    // switches from hammering IndexedDB for no reason.
    const cached = userId ? await snapshotStore.loadSnapshot(userId, 'group', id) : undefined
    const load = async <T>(key: keyof typeof groupErrors, request: () => Promise<T>, apply: (value: T) => void, cachedValue?: T) => {
      groupBusy[key]++
      try {
        if (!online.value) throw new ApiError(0, 'network_error', 'offline')
        const value = await request()
        if (expectedRequest === groupRequest && id === currentGroupId.value) { apply(value); groupErrors[key] = '' }
      } catch (reason) {
        const isNetwork = reason instanceof ApiError && reason.code === 'network_error'
        if (expectedRequest === groupRequest && id === currentGroupId.value) {
          if (isNetwork && cachedValue !== undefined) { apply(cachedValue); groupErrors[key] = '' }
          else if (isNetwork && !online.value) groupErrors[key] = '' // known offline with nothing cached for this resource — stay stale, don't alarm
          else groupErrors[key] = resourceError(reason)
        }
      } finally { groupBusy[key] = Math.max(0, groupBusy[key] - 1) }
    }
    await Promise.all([
      load('members', () => api.get<Membership[]>(`/groups/${id}/members?perPage=100`).then(value => value.data), value => { members.value = value }, cached?.members),
      load('subscriptions', () => api.get<Subscription[]>(`/groups/${id}/subscriptions?perPage=100`).then(value => value.data), value => { subscriptions.value = value }, cached?.subscriptions),
      load('expenses', () => api.get<Expense[]>(`/groups/${id}/expenses?perPage=100`).then(value => value.data), value => { expenses.value = value }, cached?.expenses),
      load('settlements', () => api.get<Settlement[]>(`/groups/${id}/settlements?perPage=100`).then(value => value.data), value => { settlements.value = value }, cached?.settlements),
      load('summary', () => api.get<DashboardSummary>(`/groups/${id}/summary`).then(value => value.data), value => { summary.value = value }),
			load('access', () => api.get<GroupAccess>(`/groups/${id}/access`).then(value => value.data), value => { groupPermissions.value = value.permissions }),
    ])
    if (expectedRequest === groupRequest && id === currentGroupId.value) {
      if (groupPermissions.value.includes('group.members.manage') && online.value) {
        try { await loadInvitations() } catch { invitations.value = [] }
      } else if (online.value) invitations.value = []
      loadedGroupId = id
      if (userId && online.value) await snapshotStore.saveSnapshot(userId, 'group', id, { expenses: expenses.value, subscriptions: subscriptions.value, settlements: settlements.value, members: members.value })
    }
  }

  async function loadInvitations(page = 1, perPage = invitationsMeta.value.perPage || 25) {
    if (!currentGroupId.value) return
    const result = await api.get<Invitation[]>(`/groups/${currentGroupId.value}/invitations?${new URLSearchParams({ page:String(page), perPage:String(perPage) }).toString()}`)
    invitations.value = result.data
    invitationsMeta.value = result.meta || { page:1, perPage, totalItems:result.data.length, totalPages:1 }
    return result
  }

  async function refreshPersonal(scope: 'personal'|'all' = 'personal', month = '') {
    const userId = auth.record?.id
    await run(async () => {
      try {
        if (!online.value) throw new ApiError(0, 'network_error', 'offline')
        const [subscriptionPage, expensePage, dashboard] = await Promise.all([
          api.get<Subscription[]>('/subscriptions?perPage=100'),
          api.get<Expense[]>('/expenses?perPage=100'),
          api.get<DashboardSummary>(`/dashboard?scope=${scope}${month ? `&month=${encodeURIComponent(month)}` : ''}`),
        ])
        personalSubscriptions.value = subscriptionPage.data
        personalExpenses.value = expensePage.data
        personalSummary.value = dashboard.data
        if (userId) await snapshotStore.saveSnapshot(userId, 'personal', '', { expenses: personalExpenses.value, subscriptions: personalSubscriptions.value })
      } catch (reason) {
        if (!(reason instanceof ApiError) || reason.code !== 'network_error') throw reason
        const cached = userId ? await snapshotStore.loadSnapshot(userId, 'personal', '') : undefined
        if (!cached) throw reason
        personalSubscriptions.value = cached.subscriptions
        personalExpenses.value = cached.expenses
      }
    }, 'personal', false)
  }

  async function refreshDashboard(scope: 'personal'|'group'|'all', groupId: string, month: string) {
    if (scope === 'group') {
      if (groupId) await selectGroup(groupId)
      await run(async () => { summary.value = (await api.get<DashboardSummary>(`/dashboard?scope=group&groupId=${encodeURIComponent(groupId)}&month=${encodeURIComponent(month)}`)).data }, 'dashboard', false)
      return
    }
    await refreshPersonal(scope, month)
  }

  // SSE events arrive individually, but a single mutation on the server can
  // fan out into several of them in quick succession (e.g. a settlement
  // touching both the settlement and the group summary). Refreshing on every
  // one of them fired a burst of redundant, back-to-back requests instead of
  // one. Coalesce a burst into a single refresh pass after a short quiet
  // window, tracking (across the whole burst, not just the last event)
  // whether groups and/or the current group need refreshing so nothing an
  // earlier event in the burst asked for gets dropped by a later one.
  const sseRefreshDebounceMs = 300
  let sseFlushTimer: ReturnType<typeof setTimeout> | undefined
  let pendingLoadGroups = false
  let pendingGroupRefresh = false
  async function flushSseRefresh() {
    sseFlushTimer = undefined
    const shouldLoadGroups = pendingLoadGroups
    const shouldRefreshGroup = pendingGroupRefresh
    pendingLoadGroups = false
    pendingGroupRefresh = false
    if (shouldLoadGroups) await loadGroups()
    await refreshPersonal()
    if (shouldRefreshGroup) await refreshGroup()
  }
  function onEvent(event: SubFlowEvent) {
    if (event.resource === 'groups' || event.resource === 'group_members') pendingLoadGroups = true
    if (event.groupId && event.groupId === currentGroupId.value) pendingGroupRefresh = true
    if (sseFlushTimer) clearTimeout(sseFlushTimer)
    sseFlushTimer = setTimeout(() => { void flushSseRefresh() }, sseRefreshDebounceMs)
  }

  // Marks the corresponding local record with the failure so a badge can
  // show it. Branches explicitly per kind rather than returning a shared
  // ref, since Expense/Subscription/Settlement are different shapes and a
  // ref typed as their union can't be assigned a mapped array back.
  function markOfflineSyncError(entry: OutboxEntry, message: string) {
    if (entry.kind === 'expense') { const list = entry.scope === 'group' ? expenses : personalExpenses; list.value = list.value.map(item => item.id === entry.targetId ? { ...item, syncError: message } : item) }
    else if (entry.kind === 'subscription') { const list = entry.scope === 'group' ? subscriptions : personalSubscriptions; list.value = list.value.map(item => item.id === entry.targetId ? { ...item, syncError: message } : item) }
    else settlements.value = settlements.value.map(item => item.id === entry.targetId ? { ...item, syncError: message } : item)
  }
  function offlinePath(kind: OutboxKind, id: string) {
    return kind === 'expense' ? `/expenses/${id}` : kind === 'subscription' ? `/subscriptions/${id}` : `/settlements/${id}`
  }
  function offlineCreatePath(entry: OutboxEntry) {
    if (entry.kind === 'settlement') return `/groups/${entry.groupId}/settlements`
    const base = entry.kind === 'expense' ? 'expenses' : 'subscriptions'
    return entry.scope === 'group' ? `/groups/${entry.groupId}/${base}` : `/${base}`
  }

  // Replays queued mutations in the order they were made. Stops at the first
  // network failure (we're still offline; the rest stay queued for next
  // time) but keeps going past any other failure — one bad entry (say, a
  // record deleted by someone else in the meantime) shouldn't block every
  // later entry behind it in the queue. A failed entry is kept, marked, and
  // left for the user to see rather than silently dropped.
  async function syncOutbox() {
    if (syncing || !online.value || !auth.record?.id) return
    const userId = auth.record.id
    syncing = true
    try {
      const entries = await outbox.listForUser(userId)
      if (!entries.length) { outboxPending.value = 0; return }
      let touchedGroupId = ''
      let touchedPersonal = false
      for (const entry of entries) {
        try {
          if (entry.op === 'create') {
            const created = await api.post<{ id: string }>(offlineCreatePath(entry), entry.payload)
            const realId = created.data.id
            for (const later of entries) if (later.targetId === entry.targetId && later.id !== entry.id) { later.targetId = realId; await outbox.updateTargetId(later.id, realId) }
          } else if (entry.op === 'update') {
            await api.patch(offlinePath(entry.kind, entry.targetId), entry.payload)
          } else {
            await api.delete(offlinePath(entry.kind, entry.targetId))
          }
          await outbox.remove(entry.id)
          if (entry.scope === 'group') touchedGroupId = entry.groupId
          else touchedPersonal = true
        } catch (reason) {
          if (reason instanceof ApiError && reason.code === 'network_error') { online.value = false; break }
          const message = reason instanceof ApiError ? reason.message : String(reason)
          await outbox.markFailed(entry.id, message)
          markOfflineSyncError(entry, message)
        }
      }
      if (touchedPersonal) await refreshPersonal()
      if (touchedGroupId && touchedGroupId === currentGroupId.value) await refreshGroup()
      await refreshOutboxPending()
    } finally {
      syncing = false
    }
  }

  async function createGroup(input: GroupInput) {
    await run(async () => {
      const group = (await api.post<Group>('/groups', input)).data
      groups.value.unshift(group)
      await selectGroup(group.id)
    })
  }

  async function updateGroup(input: GroupInput) {
    if (!currentGroupId.value) return
    await run(async () => {
      const updated = (await api.patch<Group>(`/groups/${currentGroupId.value}`, input)).data
      groups.value = groups.value.map(group => group.id === updated.id ? updated : group)
    })
  }

  async function deleteGroup() {
    if (!currentGroupId.value) return
    await run(async () => {
      const deletedId = currentGroupId.value
      await api.delete(`/groups/${deletedId}`)
      sse?.stop()
      groups.value = groups.value.filter(group => group.id !== deletedId)
      currentGroupId.value = groups.value[0]?.id ?? ''
      members.value = []
      invitations.value = []
      subscriptions.value = []
      expenses.value = []
      summary.value = null
      if (currentGroupId.value) await selectGroup(currentGroupId.value)
    })
  }

  async function removeMember(userId: string) {
    if (!currentGroupId.value) return
    await run(async () => {
      await api.delete(`/groups/${currentGroupId.value}/members/${userId}`)
      await refreshGroup()
    })
  }
  async function loadGroupRoles() { if (!currentGroupId.value) return; groupRoles.value=(await api.get<AccessRole[]>(`/groups/${currentGroupId.value}/roles`)).data }
  async function createGroupRole(input: Pick<AccessRole,'name'|'category'|'permissions'>) { if (!currentGroupId.value) return; const role=(await api.post<AccessRole>(`/groups/${currentGroupId.value}/roles`,input)).data;groupRoles.value.push(role);return role }
  async function updateGroupRole(id:string,input:Pick<AccessRole,'name'|'category'|'permissions'>) { if (!currentGroupId.value) return; const role=(await api.patch<AccessRole>(`/groups/${currentGroupId.value}/roles/${id}`,input)).data;groupRoles.value=groupRoles.value.map(value=>value.id===id?role:value);return role }
  async function deleteGroupRole(id:string) { if (!currentGroupId.value) return; await api.delete(`/groups/${currentGroupId.value}/roles/${id}`); groupRoles.value=groupRoles.value.filter(value=>value.id!==id) }
  async function assignGroupRole(userId:string,roleId:string) { if (!currentGroupId.value) return;await api.request(`/groups/${currentGroupId.value}/members/${userId}/role`,{method:'PUT',body:JSON.stringify({roleId})});await refreshGroup() }
  // A group has at most one pending transfer at a time; 404 just means none
  // exists right now, which is the normal case and not worth surfacing as an
  // error toast.
  async function loadOwnershipTransfer() {
    if (!currentGroupId.value) return
    try { ownershipTransfer.value = (await api.get<OwnershipTransfer>(`/groups/${currentGroupId.value}/ownership-transfer`)).data }
    catch (reason) { ownershipTransfer.value = undefined; if (!(reason instanceof ApiError) || reason.code !== 'not_found') throw reason }
  }
  async function createOwnershipTransfer(toUserId: string) {
    if (!currentGroupId.value) return
    return run(async () => { ownershipTransfer.value = (await api.post<OwnershipTransfer>(`/groups/${currentGroupId.value}/ownership-transfer`, { toUserId })).data })
  }
  async function respondOwnershipTransfer(id: string, accept: boolean) {
    return run(async () => { await api.post<OwnershipTransfer>(`/ownership-transfers/${id}/respond`, { accept }); ownershipTransfer.value = undefined; await refreshGroup() })
  }
  async function cancelOwnershipTransfer(id: string) {
    return run(async () => { await api.delete(`/ownership-transfers/${id}`); ownershipTransfer.value = undefined })
  }
  async function loadGroupAuditLogs(query = ''): Promise<Envelope<AuditLog[]> | undefined> {
    if (!currentGroupId.value) return
    const params = new URLSearchParams(query)
    if (!params.has('perPage')) params.set('perPage', '25')
    const result = await api.get<AuditLog[]>(`/groups/${currentGroupId.value}/audit-logs?${params.toString()}`)
    groupAuditLogs.value = result.data
    groupAuditMeta.value = result.meta || { page:1, perPage:25, totalItems:result.data.length, totalPages:1 }
    return result
  }

  async function invite(email: string, targetPlaceholderId?: string) {
    if (!currentGroupId.value) return
    await run(async () => {
      const invitation = (await api.post<Invitation>(`/groups/${currentGroupId.value}/invitations`, { email, targetPlaceholderId })).data
      invitations.value.unshift(invitation)
    })
  }

  async function createTempMember(name: string) {
    if (!currentGroupId.value) return false
    return run(async () => {
      await api.post<Membership>(`/groups/${currentGroupId.value}/temp-members`, { name })
      await refreshGroup()
    })
  }

  async function resendInvitation(id: string) {
    await run(async () => {
      const invitation = (await api.post<Invitation>(`/invitations/${id}/resend`)).data
      invitations.value = invitations.value.map(value => value.id === id ? invitation : value)
    })
  }

  async function revokeInvitation(id: string) {
    await run(async () => {
      await api.post(`/invitations/${id}/revoke`)
      await refreshGroup()
    })
  }

  async function acceptInvitation(token: string) {
    await run(async () => {
      const invitation = (await api.post<Invitation>('/invitations/accept', { token })).data
      await loadGroups()
      await selectGroup(invitation.groupId)
    })
  }
  async function loadInvitationInbox(){const [invites, notes]=await Promise.all([api.get<Invitation[]>('/invitations/pending?perPage=100'),api.get<Notification[]>('/notifications?perPage=100')]);pendingInvitations.value=invites.data;notifications.value=notes.data}
  async function acceptPendingInvitation(id:string){await run(async()=>{const invitation=(await api.post<Invitation>(`/invitations/${id}/accept`)).data;pendingInvitations.value=pendingInvitations.value.filter(item=>item.id!==id);notifications.value=notifications.value.map(item=>item.resourceId===id?{...item,readAt:new Date().toISOString()}:item);await loadGroups();await selectGroup(invitation.groupId)})}
  async function declinePendingInvitation(id:string){await run(async()=>{await api.post(`/invitations/${id}/decline`);pendingInvitations.value=pendingInvitations.value.filter(item=>item.id!==id);notifications.value=notifications.value.map(item=>item.resourceId===id?{...item,readAt:new Date().toISOString()}:item)})}
  async function markNotificationRead(id:string){await api.post(`/notifications/${id}/read`);notifications.value=notifications.value.map(item=>item.id===id?{...item,readAt:new Date().toISOString()}:item)}

  async function addSubscription(input: SubscriptionInput, backfill = false) {
    if (!currentGroupId.value) return false
    let createdId = ''
    const ok = await run(async () => {
      await withOfflineFallback(
        async () => { const response = await api.post<Subscription>(`/groups/${currentGroupId.value}/subscriptions`, input); createdId = response.data.id; await refreshGroup() },
        async () => {
          const id = outbox.localId()
          subscriptions.value = [localSubscription(id, input, currentGroupId.value), ...subscriptions.value]
          const userId = auth.record?.id
          if (userId) await outbox.enqueue({ userId, kind: 'subscription', op: 'create', scope: 'group', groupId: currentGroupId.value, targetId: id, payload: input })
        },
      )
    })
    // Offline creates never populate createdId, so a checked backfill option
    // is silently skipped rather than backfilling a record that doesn't
    // exist on the server yet — it can be run manually after syncing.
    if (ok && backfill && createdId) await backfillSubscription(createdId)
    return ok
  }

  // Posts real Expense/occurrence records for the historical periods between
  // a group subscription's StartsOn and today, closing the gap left by
  // CreateSubscription always starting NextBilling from "now" (see the
  // backend's Service.BackfillSubscriptionPeriods). Reports its own success
  // toast with the actual count instead of the generic one from run(), since
  // "backfilled 0 periods" vs "backfilled 23 periods" is the whole point of
  // calling this.
  async function backfillSubscription(id: string) {
    let created = 0
    const ok = await run(async () => {
      const response = await api.post<{ created: number }>(`/subscriptions/${id}/backfill`)
      created = response.data.created
      if (currentGroupId.value) await refreshGroup()
    }, 'general', false)
    toast.push(ok ? 'success' : 'error', ok ? tr('subscriptionBackfillDone', { count: created }) : tr('actionFailed', { reason: localizedError.value }))
    return ok ? created : -1
  }

  async function updateSubscription(id: string, input: SubscriptionInput) {
    return run(async () => {
      await withOfflineFallback(
        async () => { await api.patch<Subscription>(`/subscriptions/${id}`, input); if (currentGroupId.value) await refreshGroup(); await refreshPersonal() },
        async () => {
          const inGroup = subscriptions.value.some(item => item.id === id)
          const list = inGroup ? subscriptions : personalSubscriptions
          list.value = list.value.map(item => item.id === id ? { ...item, ...input, pendingSync: true } : item)
          const userId = auth.record?.id
          if (userId) await outbox.enqueue({ userId, kind: 'subscription', op: 'update', scope: inGroup ? 'group' : 'personal', groupId: inGroup ? currentGroupId.value : '', targetId: id, payload: input })
          if (!inGroup) await savePersonalSnapshot()
        },
      )
    })
  }

  async function deleteSubscription(id: string) {
    await run(async () => {
      await withOfflineFallback(
        async () => { await api.delete(`/subscriptions/${id}`); if (currentGroupId.value) await refreshGroup(); await refreshPersonal() },
        async () => {
          const inGroup = subscriptions.value.some(item => item.id === id)
          await localDelete('subscription', inGroup ? 'group' : 'personal', inGroup ? currentGroupId.value : '', id)
          if (inGroup) subscriptions.value = subscriptions.value.filter(item => item.id !== id)
          else { personalSubscriptions.value = personalSubscriptions.value.filter(item => item.id !== id); await savePersonalSnapshot() }
        },
      )
    })
  }

  async function addExpense(input: ExpenseInput) {
    if (!currentGroupId.value) return false
    return run(async () => {
      await withOfflineFallback(
        async () => { await api.post<Expense>(`/groups/${currentGroupId.value}/expenses`, input); await refreshGroup() },
        async () => {
          const id = outbox.localId()
          expenses.value = [localExpense(id, input, currentGroupId.value), ...expenses.value]
          const userId = auth.record?.id
          if (userId) await outbox.enqueue({ userId, kind: 'expense', op: 'create', scope: 'group', groupId: currentGroupId.value, targetId: id, payload: input })
        },
      )
    })
  }

  async function addPersonalExpense(input: ExpenseInput) {
    return run(async () => {
      await withOfflineFallback(
        async () => { await api.post<Expense>('/expenses', input); await refreshPersonal() },
        async () => {
          const id = outbox.localId()
          personalExpenses.value = [localExpense(id, input), ...personalExpenses.value]
          const userId = auth.record?.id
          if (userId) await outbox.enqueue({ userId, kind: 'expense', op: 'create', scope: 'personal', groupId: '', targetId: id, payload: input })
          await savePersonalSnapshot()
        },
      )
    })
  }
  async function addPersonalSubscription(input: SubscriptionInput) {
    return run(async () => {
      await withOfflineFallback(
        async () => { await api.post<Subscription>('/subscriptions', input); await refreshPersonal() },
        async () => {
          const id = outbox.localId()
          personalSubscriptions.value = [localSubscription(id, input), ...personalSubscriptions.value]
          const userId = auth.record?.id
          if (userId) await outbox.enqueue({ userId, kind: 'subscription', op: 'create', scope: 'personal', groupId: '', targetId: id, payload: input })
          await savePersonalSnapshot()
        },
      )
    })
  }
  async function stopSubscription(id: string, endsOn: string) {
    await run(async () => { await api.post<Subscription>(`/subscriptions/${id}/stop`, { endsOn }); await refreshPersonal(); if (currentGroupId.value) await refreshGroup() })
  }
  async function cancelSubscriptionStop(id: string) { await run(async () => { await api.delete<Subscription>(`/subscriptions/${id}/stop`); await refreshPersonal(); if (currentGroupId.value) await refreshGroup() }) }
  async function billingDates(id: string, cursor = '', includePast = false) { return (await api.get<BillingDates>(`/subscriptions/${id}/billing-dates?limit=12${cursor ? `&cursor=${encodeURIComponent(cursor)}` : ''}${includePast ? '&includePast=true' : ''}`)).data }
  async function subscriptionPeriods(id: string, cursor = '', limit = 24) { return (await api.get<SubscriptionPeriods>(`/subscriptions/${id}/periods?limit=${limit}${cursor ? `&cursor=${encodeURIComponent(cursor)}` : ''}`)).data }

  async function addSettlement(input: Pick<Settlement,'fromUserId'|'toUserId'|'amountMinor'|'settledOn'|'notes'>) {
    if (!currentGroupId.value) return false
    return run(async () => {
      await withOfflineFallback(
        async () => { await api.post<Settlement>(`/groups/${currentGroupId.value}/settlements`, input); await refreshGroup() },
        async () => {
          const id = outbox.localId()
          settlements.value = [localSettlement(id, input, currentGroupId.value), ...settlements.value]
          const userId = auth.record?.id
          if (userId) await outbox.enqueue({ userId, kind: 'settlement', op: 'create', scope: 'group', groupId: currentGroupId.value, targetId: id, payload: input })
        },
      )
    }, 'settlements')
  }
  async function deleteSettlement(id: string) {
    await run(async () => {
      await withOfflineFallback(
        async () => { await api.delete(`/settlements/${id}`); await refreshGroup() },
        async () => { await localDelete('settlement', 'group', currentGroupId.value, id); settlements.value = settlements.value.filter(item => item.id !== id) },
      )
    }, 'settlements')
  }

  async function updateExpense(id: string, input: ExpenseInput) {
    return run(async () => {
      await withOfflineFallback(
        async () => { await api.patch<Expense>(`/expenses/${id}`, input); if (currentGroupId.value) await refreshGroup(); await refreshPersonal() },
        async () => {
          const inGroup = expenses.value.some(item => item.id === id)
          const list = inGroup ? expenses : personalExpenses
          list.value = list.value.map(item => item.id === id ? { ...item, ...input, pendingSync: true } : item)
          const userId = auth.record?.id
          if (userId) await outbox.enqueue({ userId, kind: 'expense', op: 'update', scope: inGroup ? 'group' : 'personal', groupId: inGroup ? currentGroupId.value : '', targetId: id, payload: input })
          if (!inGroup) await savePersonalSnapshot()
        },
      )
    })
  }

  async function deleteExpense(id: string) {
    await run(async () => {
      await withOfflineFallback(
        async () => { await api.delete(`/expenses/${id}`); if (currentGroupId.value) await refreshGroup(); await refreshPersonal() },
        async () => {
          const inGroup = expenses.value.some(item => item.id === id)
          await localDelete('expense', inGroup ? 'group' : 'personal', inGroup ? currentGroupId.value : '', id)
          if (inGroup) expenses.value = expenses.value.filter(item => item.id !== id)
          else { personalExpenses.value = personalExpenses.value.filter(item => item.id !== id); await savePersonalSnapshot() }
        },
      )
    })
  }

  async function exportLedger(groupId?: string) {
    const path = groupId ? `/groups/${groupId}/export` : '/export/personal'
    const { blob, filename } = await api.getBlob(`${path}?locale=${encodeURIComponent(locale.value)}`)
    const url = URL.createObjectURL(blob)
    const anchor = document.createElement('a')
    anchor.href = url
    anchor.download = filename
    document.body.appendChild(anchor)
    anchor.click()
    anchor.remove()
    URL.revokeObjectURL(url)
  }

  function clear() {
    sse?.stop()
    sseStarted = false
    groups.value = []
    currentGroupId.value = ''
    loadedGroupId = ''
    hydratingGroupId = ''
    groupHydration = undefined
    groupRequest++
    members.value = []
    invitations.value = []
    pendingInvitations.value=[]; notifications.value=[]
    subscriptions.value = []
    expenses.value = []
    settlements.value = []
    for (const key of Object.keys(groupErrors) as Array<keyof typeof groupErrors>) groupErrors[key] = ''
    personalSubscriptions.value = []
    personalExpenses.value = []
    personalSummary.value = null
    summary.value = null
    error.value = ''
    errorCode.value = ''
    permissionDenied.value = false
    outboxPending.value = 0
  }

  function isForbidden(value: unknown) {
    return value instanceof ApiError && value.status === 403
  }

  return {
    groups, currencies, categories, currentGroupId, currentGroup, currentMembership, isOwner, members, invitations, invitationsMeta, loadInvitations, pendingInvitations, notifications,
    subscriptions, expenses, settlements, groupRoles, ownershipTransfer, groupAuditLogs, groupAuditMeta, groupPermissions, groupErrors, groupBusy, personalSubscriptions, personalExpenses, personalSummary, summary, loading, busy, error, localizedError, permissionDenied, loadGroups, selectGroup,
    refreshGroup, createGroup, updateGroup, deleteGroup, removeMember, invite, createTempMember, resendInvitation,
    revokeInvitation, acceptInvitation, loadInvitationInbox, acceptPendingInvitation, declinePendingInvitation, markNotificationRead, loadGroupRoles, createGroupRole, updateGroupRole, deleteGroupRole, assignGroupRole, loadOwnershipTransfer, createOwnershipTransfer, respondOwnershipTransfer, cancelOwnershipTransfer, loadGroupAuditLogs, addSubscription, backfillSubscription, updateSubscription, deleteSubscription,
    addExpense, addPersonalExpense, updateExpense, deleteExpense, addPersonalSubscription, stopSubscription, cancelSubscriptionStop, billingDates, subscriptionPeriods, addSettlement, deleteSettlement, refreshPersonal, refreshDashboard, loadCategories, createCategory, updateCategory, archiveCategory, quoteRate, previewGroupCurrency, changeGroupCurrency, retryLast, clear, isForbidden, exportLedger,
    online, outboxPending, syncOutbox, hasSyncErrors,
    onEvent,
  }
})

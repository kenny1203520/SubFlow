import { computed, reactive, ref } from 'vue'
import { defineStore } from 'pinia'
import { ApiClient, ApiError } from '../api/client'
import { SSEClient } from '../api/sse'
import type { AccessRole, AuditLog, BillingDates, Category, Currency, CurrencyChangePreview, CurrencyInfo, DashboardSummary, Envelope, ExchangeRate, Expense, Group, GroupAccess, Invitation, Membership, Meta, Notification, Settlement, Subscription, SubFlowEvent } from '../api/types'
import { useAuthStore } from './auth'
import { useToastStore } from './toast'
import { useI18n } from '../i18n'

type GroupInput = Pick<Group, 'name' | 'description' | 'currency' | 'timezone' | 'color'>
// startsOn is optional so an update can omit it when the user did not touch
// the date input: the backend then keeps the exact stored instant instead of
// a same-day reconstruction from the viewer's timezone, which would shift it
// by up to 24h relative to the original (see dateInput.ts).
type SubscriptionInput = Pick<Subscription, 'name'|'category'|'amountMinor'|'currency'|'billingCycle'|'status'|'notes'> & Partial<Pick<Subscription,'paidBy'|'endsOn'|'nextBilling'|'categoryId'|'rateMode'|'exchangeRate'|'billingInterval'|'startsOn'>>
// incurredOn is optional for the same reason startsOn is on SubscriptionInput.
type ExpenseInput = Pick<Expense, 'title'|'category'|'amountMinor'|'currency'|'paidBy'|'notes'> & Partial<Pick<Expense,'splitMode'|'splits'|'categoryId'|'rateMode'|'exchangeRate'|'incurredOn'>>

export const useWorkspaceStore = defineStore('workspace', () => {
  const auth = useAuthStore()
  const toast = useToastStore()
  const { tr } = useI18n()
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
  let sse: SSEClient | undefined
  let sseStarted = false
  let loadedGroupId = ''
  let groupRequest = 0
  let hydratingGroupId = ''
  let groupHydration: Promise<void> | undefined
  let lastRetry: (() => Promise<void>) | undefined

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

  async function loadGroups() {
    await run(async () => {
      const [groupResult,currencyResult]=await Promise.all([api.get<Group[]>('/groups?perPage=100'),api.get<CurrencyInfo[]>('/currencies')])
      groups.value = groupResult.data
      currencies.value = currencyResult.data
		  try { await loadInvitationInbox() } catch { pendingInvitations.value=[]; notifications.value=[] }
      if (currentGroupId.value && !groups.value.some(group => group.id === currentGroupId.value)) currentGroupId.value = ''
      if (!sseStarted) { sse = new SSEClient(() => auth.token, onEvent, auth.logout); sseStarted = true; void sse.start() }
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
    const load = async <T>(key: keyof typeof groupErrors, request: () => Promise<T>, apply: (value: T) => void) => {
      groupBusy[key]++
      try {
        const value = await request()
        if (expectedRequest === groupRequest && id === currentGroupId.value) { apply(value); groupErrors[key] = '' }
      } catch (reason) {
        if (expectedRequest === groupRequest && id === currentGroupId.value) groupErrors[key] = resourceError(reason)
      } finally { groupBusy[key] = Math.max(0, groupBusy[key] - 1) }
    }
    await Promise.all([
      load('members', () => api.get<Membership[]>(`/groups/${id}/members?perPage=100`).then(value => value.data), value => { members.value = value }),
      load('subscriptions', () => api.get<Subscription[]>(`/groups/${id}/subscriptions?perPage=100`).then(value => value.data), value => { subscriptions.value = value }),
      load('expenses', () => api.get<Expense[]>(`/groups/${id}/expenses?perPage=100`).then(value => value.data), value => { expenses.value = value }),
      load('settlements', () => api.get<Settlement[]>(`/groups/${id}/settlements?perPage=100`).then(value => value.data), value => { settlements.value = value }),
      load('summary', () => api.get<DashboardSummary>(`/groups/${id}/summary`).then(value => value.data), value => { summary.value = value }),
			load('access', () => api.get<GroupAccess>(`/groups/${id}/access`).then(value => value.data), value => { groupPermissions.value = value.permissions }),
    ])
    if (expectedRequest === groupRequest && id === currentGroupId.value) {
      if (groupPermissions.value.includes('group.members.manage')) {
        try { await loadInvitations() } catch { invitations.value = [] }
      } else invitations.value = []
      loadedGroupId = id
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
    await run(async () => {
      const [subscriptionPage, expensePage, dashboard] = await Promise.all([
        api.get<Subscription[]>('/subscriptions?perPage=100'),
        api.get<Expense[]>('/expenses?perPage=100'),
        api.get<DashboardSummary>(`/dashboard?scope=${scope}${month ? `&month=${encodeURIComponent(month)}` : ''}`),
      ])
      personalSubscriptions.value = subscriptionPage.data
      personalExpenses.value = expensePage.data
      personalSummary.value = dashboard.data
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

  async function onEvent(event: SubFlowEvent) {
    if (event.resource === 'groups' || event.resource === 'group_members') await loadGroups()
    await refreshPersonal()
    if (event.groupId && event.groupId === currentGroupId.value) await refreshGroup()
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
  async function loadGroupAuditLogs(query = ''): Promise<Envelope<AuditLog[]> | undefined> {
    if (!currentGroupId.value) return
    const params = new URLSearchParams(query)
    if (!params.has('perPage')) params.set('perPage', '25')
    const result = await api.get<AuditLog[]>(`/groups/${currentGroupId.value}/audit-logs?${params.toString()}`)
    groupAuditLogs.value = result.data
    groupAuditMeta.value = result.meta || { page:1, perPage:25, totalItems:result.data.length, totalPages:1 }
    return result
  }

  async function invite(email: string) {
    if (!currentGroupId.value) return
    await run(async () => {
      const invitation = (await api.post<Invitation>(`/groups/${currentGroupId.value}/invitations`, { email })).data
      invitations.value.unshift(invitation)
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

  async function addSubscription(input: SubscriptionInput) {
    if (!currentGroupId.value) return false
    return run(async () => {
      await api.post<Subscription>(`/groups/${currentGroupId.value}/subscriptions`, input)
      await refreshGroup()
    })
  }

  async function updateSubscription(id: string, input: SubscriptionInput) {
    return run(async () => {
      await api.patch<Subscription>(`/subscriptions/${id}`, input)
      if (currentGroupId.value) await refreshGroup()
      await refreshPersonal()
    })
  }

  async function deleteSubscription(id: string) {
    await run(async () => {
      await api.delete(`/subscriptions/${id}`)
      if (currentGroupId.value) await refreshGroup()
      await refreshPersonal()
    })
  }

  async function addExpense(input: ExpenseInput) {
    if (!currentGroupId.value) return false
    return run(async () => {
      await api.post<Expense>(`/groups/${currentGroupId.value}/expenses`, input)
      await refreshGroup()
    })
  }

  async function addPersonalExpense(input: ExpenseInput) {
    return run(async () => { await api.post<Expense>('/expenses', input); await refreshPersonal() })
  }
  async function addPersonalSubscription(input: SubscriptionInput) {
    return run(async () => { await api.post<Subscription>('/subscriptions', input); await refreshPersonal() })
  }
  async function stopSubscription(id: string, endsOn: string) {
    await run(async () => { await api.post<Subscription>(`/subscriptions/${id}/stop`, { endsOn }); await refreshPersonal(); if (currentGroupId.value) await refreshGroup() })
  }
  async function cancelSubscriptionStop(id: string) { await run(async () => { await api.delete<Subscription>(`/subscriptions/${id}/stop`); await refreshPersonal(); if (currentGroupId.value) await refreshGroup() }) }
  async function billingDates(id: string, cursor = '', includePast = false) { return (await api.get<BillingDates>(`/subscriptions/${id}/billing-dates?limit=12${cursor ? `&cursor=${encodeURIComponent(cursor)}` : ''}${includePast ? '&includePast=true' : ''}`)).data }

  async function addSettlement(input: Pick<Settlement,'fromUserId'|'toUserId'|'amountMinor'|'settledOn'|'notes'>) {
    if (!currentGroupId.value) return false
    return run(async () => { await api.post<Settlement>(`/groups/${currentGroupId.value}/settlements`, input); await refreshGroup() }, 'settlements')
  }
  async function deleteSettlement(id: string) { await run(async () => { await api.delete(`/settlements/${id}`); await refreshGroup() }, 'settlements') }

  async function updateExpense(id: string, input: ExpenseInput) {
    return run(async () => {
      await api.patch<Expense>(`/expenses/${id}`, input)
      if (currentGroupId.value) await refreshGroup()
      await refreshPersonal()
    })
  }

  async function deleteExpense(id: string) {
    await run(async () => {
      await api.delete(`/expenses/${id}`)
      if (currentGroupId.value) await refreshGroup()
      await refreshPersonal()
    })
  }

  async function exportLedger(groupId?: string) {
    const { blob, filename } = await api.getBlob(groupId ? `/groups/${groupId}/export` : '/export/personal')
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
  }

  function isForbidden(value: unknown) {
    return value instanceof ApiError && value.status === 403
  }

  return {
    groups, currencies, categories, currentGroupId, currentGroup, currentMembership, isOwner, members, invitations, invitationsMeta, loadInvitations, pendingInvitations, notifications,
    subscriptions, expenses, settlements, groupRoles, groupAuditLogs, groupAuditMeta, groupPermissions, groupErrors, groupBusy, personalSubscriptions, personalExpenses, personalSummary, summary, loading, busy, error, localizedError, permissionDenied, loadGroups, selectGroup,
    refreshGroup, createGroup, updateGroup, deleteGroup, removeMember, invite, resendInvitation,
    revokeInvitation, acceptInvitation, loadInvitationInbox, acceptPendingInvitation, declinePendingInvitation, markNotificationRead, loadGroupRoles, createGroupRole, updateGroupRole, deleteGroupRole, assignGroupRole, loadGroupAuditLogs, addSubscription, updateSubscription, deleteSubscription,
    addExpense, addPersonalExpense, updateExpense, deleteExpense, addPersonalSubscription, stopSubscription, cancelSubscriptionStop, billingDates, addSettlement, deleteSettlement, refreshPersonal, refreshDashboard, loadCategories, createCategory, updateCategory, archiveCategory, quoteRate, previewGroupCurrency, changeGroupCurrency, retryLast, clear, isForbidden, exportLedger,
  }
})

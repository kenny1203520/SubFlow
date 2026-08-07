import { computed, reactive, ref } from 'vue'
import { defineStore } from 'pinia'
import { ApiClient, ApiError } from '../api/client'
import { SSEClient } from '../api/sse'
import type { BillingDates, Category, Currency, CurrencyChangePreview, CurrencyInfo, DashboardSummary, ExchangeRate, Expense, Group, Invitation, Membership, Settlement, Subscription, SubFlowEvent } from '../api/types'
import { useAuthStore } from './auth'
import { useI18n } from '../i18n'

type GroupInput = Pick<Group, 'name' | 'description' | 'currency' | 'timezone' | 'color'>
type SubscriptionInput = Pick<Subscription, 'name'|'category'|'amountMinor'|'currency'|'billingCycle'|'startsOn'|'status'|'notes'> & Partial<Pick<Subscription,'paidBy'|'endsOn'|'nextBilling'|'categoryId'|'rateMode'|'exchangeRate'>>
type ExpenseInput = Pick<Expense, 'title'|'category'|'amountMinor'|'currency'|'paidBy'|'incurredOn'|'notes'> & Partial<Pick<Expense,'splitMode'|'splits'|'categoryId'|'rateMode'|'exchangeRate'>>

export const useWorkspaceStore = defineStore('workspace', () => {
  const auth = useAuthStore()
  const { tr } = useI18n()
  const api = new ApiClient(() => auth.token, auth.logout)
  const groups = ref<Group[]>([])
  const currencies = ref<CurrencyInfo[]>([])
  const categories = ref<Category[]>([])
  const currentGroupId = ref('')
  const members = ref<Membership[]>([])
  const invitations = ref<Invitation[]>([])
  const subscriptions = ref<Subscription[]>([])
  const expenses = ref<Expense[]>([])
  const settlements = ref<Settlement[]>([])
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

  async function run(task: () => Promise<void>, key = 'general') {
    busy[key] = (busy[key] || 0) + 1
    lastRetry = task
    error.value = ''
    errorCode.value = ''
    permissionDenied.value = false
    try {
      await task()
    } catch (reason) {
      const value = reason as { status?: number; message?: string }
      permissionDenied.value = value.status === 403
      error.value = value.message ?? ''
      errorCode.value = reason instanceof ApiError ? reason.code : 'internal_error'
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
      if (currentGroupId.value && !groups.value.some(group => group.id === currentGroupId.value)) currentGroupId.value = ''
      if (!sseStarted) { sse = new SSEClient(() => auth.token, onEvent, auth.logout); sseStarted = true; void sse.start() }
    }, 'groups')
  }

  async function loadCategories(scope:'personal'|'group',groupId='') { categories.value=(await api.get<Category[]>(`/categories?scope=${scope}${groupId?`&groupId=${encodeURIComponent(groupId)}`:''}`)).data }
  async function createCategory(scope:'personal'|'group',customName:string,groupId=''){const value=(await api.post<Category>('/categories',{scope,customName,groupId})).data;categories.value.push(value);return value}
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
      subscriptions.value = []
      expenses.value = []
      summary.value = null
      return
    }
    hydratingGroupId = id
    groupHydration = refreshGroup(request).finally(() => { if (hydratingGroupId === id) { hydratingGroupId='';groupHydration=undefined } })
    await groupHydration
  }

  async function refreshGroup(expectedRequest = ++groupRequest) {
    if (!currentGroupId.value) return
    const id = currentGroupId.value
    await run(async () => {
      const [memberPage, subscriptionPage, expensePage, settlementPage, dashboard] = await Promise.all([
        api.get<Membership[]>(`/groups/${id}/members?perPage=100`),
        api.get<Subscription[]>(`/groups/${id}/subscriptions?perPage=100`),
        api.get<Expense[]>(`/groups/${id}/expenses?perPage=100`),
        api.get<Settlement[]>(`/groups/${id}/settlements?perPage=100`),
        api.get<DashboardSummary>(`/groups/${id}/summary`),
      ])
      if (expectedRequest !== groupRequest || id !== currentGroupId.value) return
      members.value = memberPage.data
      subscriptions.value = subscriptionPage.data
      expenses.value = expensePage.data
      settlements.value = settlementPage.data
      summary.value = dashboard.data
      const me = memberPage.data.find(member => member.userId === auth.record?.id)
      invitations.value = me?.role === 'owner'
        ? (await api.get<Invitation[]>(`/groups/${id}/invitations?perPage=100`)).data
        : []
      if (expectedRequest === groupRequest && id === currentGroupId.value) loadedGroupId = id
    }, 'group')
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
    }, 'personal')
  }

  async function refreshDashboard(scope: 'personal'|'group'|'all', groupId: string, month: string) {
    if (scope === 'group') {
      if (groupId) await selectGroup(groupId)
      await run(async () => { summary.value = (await api.get<DashboardSummary>(`/dashboard?scope=group&groupId=${encodeURIComponent(groupId)}&month=${encodeURIComponent(month)}`)).data }, 'dashboard')
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

  async function addSubscription(input: SubscriptionInput) {
    if (!currentGroupId.value) return
    await run(async () => {
      await api.post<Subscription>(`/groups/${currentGroupId.value}/subscriptions`, input)
      await refreshGroup()
    })
  }

  async function updateSubscription(id: string, input: SubscriptionInput) {
    await run(async () => {
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
    if (!currentGroupId.value) return
    await run(async () => {
      await api.post<Expense>(`/groups/${currentGroupId.value}/expenses`, input)
      await refreshGroup()
    })
  }

  async function addPersonalExpense(input: ExpenseInput) {
    await run(async () => { await api.post<Expense>('/expenses', input); await refreshPersonal() })
  }
  async function addPersonalSubscription(input: SubscriptionInput) {
    await run(async () => { await api.post<Subscription>('/subscriptions', input); await refreshPersonal() })
  }
  async function stopSubscription(id: string, endsOn: string) {
    await run(async () => { await api.post<Subscription>(`/subscriptions/${id}/stop`, { endsOn }); await refreshPersonal(); if (currentGroupId.value) await refreshGroup() })
  }
  async function cancelSubscriptionStop(id: string) { await run(async () => { await api.delete<Subscription>(`/subscriptions/${id}/stop`); await refreshPersonal(); if (currentGroupId.value) await refreshGroup() }) }
  async function billingDates(id: string, cursor = '') { return (await api.get<BillingDates>(`/subscriptions/${id}/billing-dates?limit=12${cursor ? `&cursor=${encodeURIComponent(cursor)}` : ''}`)).data }

  async function addSettlement(input: Pick<Settlement,'fromUserId'|'toUserId'|'amountMinor'|'settledOn'|'notes'>) {
    if (!currentGroupId.value) return
    await run(async () => { await api.post<Settlement>(`/groups/${currentGroupId.value}/settlements`, input); await refreshGroup() }, 'settlements')
  }
  async function deleteSettlement(id: string) { await run(async () => { await api.delete(`/settlements/${id}`); await refreshGroup() }, 'settlements') }

  async function updateExpense(id: string, input: ExpenseInput) {
    await run(async () => {
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
    subscriptions.value = []
    expenses.value = []
    settlements.value = []
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
    groups, currencies, categories, currentGroupId, currentGroup, currentMembership, isOwner, members, invitations,
    subscriptions, expenses, settlements, personalSubscriptions, personalExpenses, personalSummary, summary, loading, busy, error, localizedError, permissionDenied, loadGroups, selectGroup,
    refreshGroup, createGroup, updateGroup, deleteGroup, removeMember, invite, resendInvitation,
    revokeInvitation, acceptInvitation, addSubscription, updateSubscription, deleteSubscription,
    addExpense, addPersonalExpense, updateExpense, deleteExpense, addPersonalSubscription, stopSubscription, cancelSubscriptionStop, billingDates, addSettlement, deleteSettlement, refreshPersonal, refreshDashboard, loadCategories, createCategory, archiveCategory, quoteRate, previewGroupCurrency, changeGroupCurrency, retryLast, clear, isForbidden,
  }
})

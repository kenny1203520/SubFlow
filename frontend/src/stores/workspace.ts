import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import { ApiClient, ApiError } from '../api/client'
import { SSEClient } from '../api/sse'
import type { DashboardSummary, Expense, Group, Invitation, Membership, Subscription, SubFlowEvent } from '../api/types'
import { useAuthStore } from './auth'

type GroupInput = Pick<Group, 'name' | 'description' | 'currency' | 'color'>
type SubscriptionInput = Pick<Subscription, 'name'|'category'|'amountMinor'|'currency'|'billingCycle'|'startsOn'|'status'|'notes'> & Partial<Pick<Subscription,'paidBy'|'endsOn'|'nextBilling'>>
type ExpenseInput = Pick<Expense, 'title'|'category'|'amountMinor'|'paidBy'|'incurredOn'|'notes'> & Partial<Pick<Expense,'splitMode'|'splits'>>

export const useWorkspaceStore = defineStore('workspace', () => {
  const auth = useAuthStore()
  const api = new ApiClient(() => auth.token, auth.logout)
  const groups = ref<Group[]>([])
  const currentGroupId = ref('')
  const members = ref<Membership[]>([])
  const invitations = ref<Invitation[]>([])
  const subscriptions = ref<Subscription[]>([])
  const expenses = ref<Expense[]>([])
  const personalSubscriptions = ref<Subscription[]>([])
  const personalExpenses = ref<Expense[]>([])
  const personalSummary = ref<DashboardSummary | null>(null)
  const summary = ref<DashboardSummary | null>(null)
  const loading = ref(false)
  const error = ref('')
  const permissionDenied = ref(false)
  const currentGroup = computed(() => groups.value.find(value => value.id === currentGroupId.value))
  const currentMembership = computed(() => members.value.find(value => value.userId === auth.record?.id))
  const isOwner = computed(() => currentMembership.value?.role === 'owner')
  let sse: SSEClient | undefined

  async function run(task: () => Promise<void>) {
    loading.value = true
    error.value = ''
    permissionDenied.value = false
    try {
      await task()
    } catch (reason) {
      const value = reason as { status?: number; message?: string }
      permissionDenied.value = value.status === 403
      error.value = value.message ?? '發生未預期錯誤'
    } finally {
      loading.value = false
    }
  }

  async function loadGroups() {
    await run(async () => {
      groups.value = (await api.get<Group[]>('/groups?perPage=100')).data
      if (!groups.value.some(group => group.id === currentGroupId.value)) currentGroupId.value = groups.value[0]?.id ?? ''
    })
  }

  async function selectGroup(id: string) {
    currentGroupId.value = id
    sse?.stop()
    if (!id) {
      members.value = []
      invitations.value = []
      subscriptions.value = []
      expenses.value = []
      summary.value = null
      return
    }
    sse = new SSEClient(() => auth.token, onEvent, auth.logout)
    void sse.start(id)
    await refreshGroup()
  }

  async function refreshGroup() {
    if (!currentGroupId.value) return
    const id = currentGroupId.value
    await run(async () => {
      const [memberPage, subscriptionPage, expensePage, dashboard] = await Promise.all([
        api.get<Membership[]>(`/groups/${id}/members?perPage=100`),
        api.get<Subscription[]>(`/groups/${id}/subscriptions?perPage=100`),
        api.get<Expense[]>(`/groups/${id}/expenses?perPage=100`),
        api.get<DashboardSummary>(`/groups/${id}/summary`),
      ])
      members.value = memberPage.data
      subscriptions.value = subscriptionPage.data
      expenses.value = expensePage.data
      summary.value = dashboard.data
      const me = memberPage.data.find(member => member.userId === auth.record?.id)
      invitations.value = me?.role === 'owner'
        ? (await api.get<Invitation[]>(`/groups/${id}/invitations?perPage=100`)).data
        : []
    })
  }

  async function refreshPersonal(scope: 'personal'|'all' = 'personal') {
    await run(async () => {
      const [subscriptionPage, expensePage, dashboard] = await Promise.all([
        api.get<Subscription[]>('/subscriptions?perPage=100'),
        api.get<Expense[]>('/expenses?perPage=100'),
        api.get<DashboardSummary>(`/dashboard?scope=${scope}`),
      ])
      personalSubscriptions.value = subscriptionPage.data
      personalExpenses.value = expensePage.data
      personalSummary.value = dashboard.data
    })
  }

  async function onEvent(event: SubFlowEvent) {
    if (event.groupId !== currentGroupId.value) return
    if (event.resource === 'groups' || event.resource === 'group_members') await loadGroups()
    await refreshGroup()
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
      await refreshGroup()
    })
  }

  async function deleteSubscription(id: string) {
    await run(async () => {
      await api.delete(`/subscriptions/${id}`)
      await refreshGroup()
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
    await run(async () => { await api.post<Subscription>(`/subscriptions/${id}/stop`, { endsOn }); await refreshPersonal() })
  }

  async function updateExpense(id: string, input: ExpenseInput) {
    await run(async () => {
      await api.patch<Expense>(`/expenses/${id}`, input)
      await refreshGroup()
    })
  }

  async function deleteExpense(id: string) {
    await run(async () => {
      await api.delete(`/expenses/${id}`)
      await refreshGroup()
    })
  }

  function clear() {
    sse?.stop()
    groups.value = []
    currentGroupId.value = ''
    members.value = []
    invitations.value = []
    subscriptions.value = []
    expenses.value = []
    personalSubscriptions.value = []
    personalExpenses.value = []
    personalSummary.value = null
    summary.value = null
    error.value = ''
    permissionDenied.value = false
  }

  function isForbidden(value: unknown) {
    return value instanceof ApiError && value.status === 403
  }

  return {
    groups, currentGroupId, currentGroup, currentMembership, isOwner, members, invitations,
    subscriptions, expenses, personalSubscriptions, personalExpenses, personalSummary, summary, loading, error, permissionDenied, loadGroups, selectGroup,
    refreshGroup, createGroup, updateGroup, deleteGroup, removeMember, invite, resendInvitation,
    revokeInvitation, acceptInvitation, addSubscription, updateSubscription, deleteSubscription,
    addExpense, addPersonalExpense, updateExpense, deleteExpense, addPersonalSubscription, stopSubscription, refreshPersonal, clear, isForbidden,
  }
})

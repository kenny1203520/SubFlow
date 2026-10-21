import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import { ApiClient, ApiError } from '../api/client'
import type { Meta, Notification, NotificationPreferences, NotificationSummary } from '../api/types'
import { useI18n } from '../i18n'
import { useAuthStore } from './auth'
import { useToastStore } from './toast'
import type { SubFlowEvent } from '../api/types'

export type NotificationFilter = 'all'|'unread'|'pending'|'resolved'

export const useNotificationsStore = defineStore('notifications', () => {
  const auth = useAuthStore()
  const toast = useToastStore()
  const api = new ApiClient(() => auth.token, () => auth.logout())
  const { tr } = useI18n()
  const items = ref<Notification[]>([])
  const summary = ref<NotificationSummary>({ unread:0, pending:0, unseen:0 })
  const preferences = ref<NotificationPreferences>({ tasksToast:true, tasksEmail:true, exceptionsToast:true, exceptionsEmail:true, securityToast:true, securityEmail:true })
  const meta = ref<Meta>({ page:1, perPage:20, totalItems:0, totalPages:0 })
  const filter = ref<NotificationFilter>('all')
  const category = ref('')
  const busy = ref(false)
  const error = ref('')
  const requestId = ref('')
  const hasNew = computed(() => summary.value.unseen > 0)
  const pendingCount = computed(() => summary.value.pending)
  let activeUserId = ''
  const announced = new Set<string>()

  function errorText(reason: unknown) {
    if (reason instanceof ApiError) return [reason.message, reason.code, reason.requestId].filter(Boolean).join(' · ')
    return reason instanceof Error ? reason.message : tr('requestFailed')
  }

  function reportActionError(reason: unknown) {
    toast.push('error', errorText(reason))
  }

  function ensureUser() {
    const current = auth.record?.id || ''
    if (activeUserId !== current) {
      items.value = []
      summary.value = { unread:0, pending:0, unseen:0 }
      preferences.value = { tasksToast:true, tasksEmail:true, exceptionsToast:true, exceptionsEmail:true, securityToast:true, securityEmail:true }
      meta.value = { page:1, perPage:20, totalItems:0, totalPages:0 }
      error.value = ''
      requestId.value = ''
      announced.clear()
      activeUserId = current
    }
  }

  async function load(page=1) {
    ensureUser()
    busy.value = true
    error.value = ''
    requestId.value = ''
    const params = new URLSearchParams({ page:String(page), perPage:String(meta.value.perPage) })
    if (filter.value === 'unread') params.set('read','unread')
    if (filter.value === 'pending') params.set('status','pending')
    if (filter.value === 'resolved') params.set('status','closed')
    if (category.value) params.set('category',category.value)
    try {
      const response = await api.get<Notification[]>(`/notifications?${params}`)
      items.value = response.data
      if (response.meta) meta.value = response.meta
      else meta.value = { ...meta.value, page, totalItems:response.data.length, totalPages:response.data.length ? 1 : 0 }
      await refreshSummary()
      return true
    } catch (reason) {
      error.value = errorText(reason)
      requestId.value = reason instanceof ApiError ? reason.requestId || '' : ''
      return false
    } finally { busy.value = false }
  }

  async function refreshSummary() {
    ensureUser()
    try { summary.value = (await api.get<NotificationSummary>('/notifications/summary')).data; return true }
    catch { return false }
  }

  async function loadPreferences() {
    try { preferences.value = (await api.get<NotificationPreferences>('/notifications/preferences')).data; return true }
    catch { return false }
  }

  async function savePreferences() {
    try {
      preferences.value = (await api.patch<NotificationPreferences>('/notifications/preferences',preferences.value)).data
      toast.push('success',tr('notificationPreferencesSaved'))
      return true
    } catch (reason) {
      reportActionError(reason)
      return false
    }
  }

  async function markVisibleViewed(ids=items.value.map(item=>item.id)) {
    if (!ids.length) return true
    try {
      await api.post('/notifications/viewed',{ ids })
      const viewedAt = new Date().toISOString()
      const visible = new Set(ids)
      items.value = items.value.map(item => visible.has(item.id) && !item.viewedAt ? { ...item, viewedAt } : item)
      await refreshSummary()
      return true
    } catch (reason) { reportActionError(reason); return false }
  }

  async function setRead(id:string, read:boolean) {
    try {
      await api.post(`/notifications/${id}/${read?'read':'unread'}`)
      const readAt = read ? new Date().toISOString() : undefined
      items.value = items.value.map(item => item.id===id ? { ...item, readAt } : item)
      await refreshSummary()
      return true
    } catch (reason) { reportActionError(reason); return false }
  }

  async function markAllRead() {
    try {
      await api.post('/notifications/read-all')
      const readAt = new Date().toISOString()
      items.value = items.value.map(item => ({ ...item, readAt:item.readAt||readAt }))
      await refreshSummary()
      return true
    } catch (reason) { reportActionError(reason); return false }
  }

  async function refresh() {
    await refreshSummary()
    return load(meta.value.page || 1)
  }

  async function handleEvent(event:SubFlowEvent) {
    if (event.resource !== 'notifications') return
    await refreshSummary()
    if (event.type !== 'created' || !event.resourceId || announced.has(event.resourceId)) return
    announced.add(event.resourceId)
    try {
      const response = await api.get<Notification[]>('/notifications?page=1&perPage=5')
      const item = response.data.find(value => value.id === event.resourceId)
      if (item) {
        const enabled = item.category === 'task' ? preferences.value.tasksToast : item.category === 'exception' ? preferences.value.exceptionsToast : preferences.value.securityToast
        if (enabled) toast.push(item.category === 'exception' ? 'error' : 'success', tr('newNotification') + ': ' + tr(item.title || item.type))
      }
    } catch { announced.delete(event.resourceId) /* Retry after a later refresh. */ }
  }

  function clear() {
    items.value = []
    summary.value = { unread:0, pending:0, unseen:0 }
    meta.value = { page:1, perPage:20, totalItems:0, totalPages:0 }
    error.value = ''
    requestId.value = ''
  }

  return { items, summary, preferences, meta, filter, category, busy, error, requestId, hasNew, pendingCount, load, refreshSummary, loadPreferences, savePreferences, markVisibleViewed, setRead, markAllRead, refresh, handleEvent, clear }
})

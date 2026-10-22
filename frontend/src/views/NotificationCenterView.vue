<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useNotificationsStore, type NotificationFilter } from '../stores/notifications'
import { useWorkspaceStore } from '../stores/workspace'
import { useI18n } from '../i18n'
import NotificationItem from '../components/notifications/NotificationItem.vue'

const notifications = useNotificationsStore()
const workspace = useWorkspaceStore()
const { tr, formatDate } = useI18n()
const preferencesOpen = ref(false)
const filters: { value:NotificationFilter; label:string }[] = [
  { value:'all', label:'notificationAll' }, { value:'unread', label:'notificationUnread' },
  { value:'pending', label:'notificationPending' }, { value:'resolved', label:'notificationResolved' },
]
const totalLabel = computed(() => tr('pendingCount',{count:notifications.pendingCount}))
async function copyRequestId() {
  if (notifications.requestId) await navigator.clipboard.writeText(notifications.requestId)
}

async function reload(page=1) {
  if (await notifications.load(page)) await notifications.markVisibleViewed()
}
function title(type:string, outcome?:string) {
  if (outcome === 'accepted') return tr('notificationAccepted')
  if (outcome === 'declined') return tr('notificationDeclined')
  if (outcome === 'revoked') return tr('notificationRevoked')
  if (outcome === 'expired') return tr('notificationExpired')
  if (type === 'group_invitation') return tr('pendingInvitations')
  return tr(type)
}
function statusLabel(status:string) {
  return status === 'pending' ? tr('notificationPending') : status === 'invalid' ? tr('notificationExpired') : tr('notificationResolved')
}
function categoryLabel(category:string) {
  return tr(`notification${category[0].toUpperCase()}${category.slice(1)}`)
}
function sourcePath(type:string, groupId?:string) {
  if (type === 'subscription_post_failed' || type === 'recurring_income_post_failed') return groupId ? `/groups/${groupId}/${type==='subscription_post_failed'?'subscriptions':'incomes'}` : `/personal/${type==='subscription_post_failed'?'subscriptions':'incomes'}`
  if (!groupId) return ''
  if (type === 'ownership_transfer') return `/groups/${groupId}/settings`
  if (type === 'member_transfer') return `/groups/${groupId}/members`
  if (type === 'subscription_post_failed') return `/groups/${groupId}/subscriptions`
  return ''
}
async function respond(id:string, accept:boolean) {
  if (accept) await workspace.acceptPendingInvitation(id)
  else await workspace.declinePendingInvitation(id)
  await workspace.loadInvitationInbox()
  await notifications.refresh()
}
watch([notifications.filter, notifications.category], () => { void reload(1) })
onMounted(async () => {
  await Promise.all([workspace.loadInvitationInbox().catch(()=>{}), notifications.loadPreferences(), reload(1)])
})
</script>

<template>
  <main class="page notification-center" aria-labelledby="notifications-title">
    <header class="page-heading">
      <div><p class="eyebrow">{{ tr('workspace') }}</p><h1 id="notifications-title">{{ tr('notificationCenter') }}</h1><p>{{ totalLabel }}</p></div>
      <button class="button ghost" type="button" :disabled="!notifications.summary.unread || notifications.busy" @click="notifications.markAllRead">{{ tr('notificationMarkAllRead') }}</button>
    </header>

    <div class="notification-filters" role="group" :aria-label="tr('notificationStatus')">
      <button v-for="item in filters" :key="item.value" type="button" class="filter-chip" :aria-pressed="notifications.filter===item.value" @click="notifications.filter=item.value">{{ tr(item.label) }}</button>
      <label class="category-filter"><span>{{ tr('notificationCategory') }}</span><select v-model="notifications.category"><option value="">{{ tr('notificationAll') }}</option><option value="task">{{ tr('notificationTask') }}</option><option value="exception">{{ tr('notificationException') }}</option><option value="security">{{ tr('notificationSecurity') }}</option></select></label>
    </div>

    <section class="card preference-panel" aria-labelledby="notification-preferences-title">
      <button id="notification-preferences-title" class="preference-toggle" type="button" :aria-expanded="preferencesOpen" aria-controls="notification-preferences-content" @click="preferencesOpen=!preferencesOpen">
        <span>{{ tr('notificationPreferences') }}</span><span aria-hidden="true">{{ preferencesOpen ? '−' : '+' }}</span>
      </button>
      <div v-if="preferencesOpen" id="notification-preferences-content" class="preference-content">
        <div class="preference-table-wrap">
          <table class="preference-table">
            <thead><tr><th scope="col">{{ tr('notificationCategory') }}</th><th scope="col">{{ tr('notificationToast') }}</th><th scope="col">{{ tr('notificationEmail') }}</th></tr></thead>
            <tbody>
              <tr><th scope="row">{{ tr('notificationTask') }}</th><td><label><span class="preference-cell-label" aria-hidden="true">{{ tr('notificationToastShort') }}</span><input v-model="notifications.preferences.tasksToast" type="checkbox" :aria-label="`${tr('notificationTask')} — ${tr('notificationToast')}`"></label></td><td><label><span class="preference-cell-label" aria-hidden="true">{{ tr('notificationEmailShort') }}</span><input v-model="notifications.preferences.tasksEmail" type="checkbox" :aria-label="`${tr('notificationTask')} — ${tr('notificationEmail')}`"></label></td></tr>
              <tr><th scope="row">{{ tr('notificationException') }}</th><td><label><span class="preference-cell-label" aria-hidden="true">{{ tr('notificationToastShort') }}</span><input v-model="notifications.preferences.exceptionsToast" type="checkbox" :aria-label="`${tr('notificationException')} — ${tr('notificationToast')}`"></label></td><td><label><span class="preference-cell-label" aria-hidden="true">{{ tr('notificationEmailShort') }}</span><input v-model="notifications.preferences.exceptionsEmail" type="checkbox" :aria-label="`${tr('notificationException')} — ${tr('notificationEmail')}`"></label></td></tr>
              <tr><th scope="row">{{ tr('notificationSecurity') }}</th><td><label><span class="preference-cell-label" aria-hidden="true">{{ tr('notificationToastShort') }}</span><input checked disabled type="checkbox" :aria-label="`${tr('notificationSecurity')} — ${tr('notificationToast')}`" aria-describedby="notification-security-note"></label></td><td><label><span class="preference-cell-label" aria-hidden="true">{{ tr('notificationEmailShort') }}</span><input checked disabled type="checkbox" :aria-label="`${tr('notificationSecurity')} — ${tr('notificationEmail')}`" aria-describedby="notification-security-note"></label></td></tr>
            </tbody>
          </table>
        </div>
        <p id="notification-security-note" class="security-note">{{ tr('notificationSecurityRequired') }}</p>
        <button class="button primary" type="button" :disabled="notifications.busy" @click="notifications.savePreferences">{{ tr('notificationSavePreferences') }}</button>
      </div>
    </section>

    <div v-if="notifications.error" class="notice danger notification-error" role="alert"><span>{{ tr('notificationLoadFailed') }} <code>{{ notifications.error }}</code></span><span v-if="notifications.requestId">{{ tr('requestId') }}: <code>{{ notifications.requestId }}</code> <button type="button" @click="copyRequestId">{{ tr('copyErrorReference') }}</button></span><button type="button" @click="reload(notifications.meta.page)">{{ tr('retry') }}</button></div>
    <div v-else-if="notifications.busy && !notifications.items.length" class="card" role="status" aria-live="polite">{{ tr('loading') }}</div>
    <div v-else-if="!notifications.items.length" class="card empty-state"><span class="empty-state-mark" aria-hidden="true">✓</span><h2>{{ tr('notificationEmpty') }}</h2><button class="button ghost" type="button" @click="preferencesOpen=true">{{ tr('notificationPreferences') }}</button></div>

    <ul v-else class="notification-list" :aria-label="tr('notificationCenter')">
      <li v-for="note in notifications.items" :key="note.id">
        <NotificationItem :title="title(note.type,note.outcome)" :category="categoryLabel(note.category)" :status="statusLabel(note.status)" :read-state="note.readAt ? tr('notificationReadState') : tr('notificationUnreadState')" :created-at="note.createdAt" :formatted-date="formatDate(note.createdAt)" :message="note.message" :unread="!note.readAt">
          <template #actions>
          <RouterLink v-if="sourcePath(note.type,note.groupId)" class="source-link" :to="sourcePath(note.type,note.groupId)">{{ tr('notificationOpenSource') }}</RouterLink>
          <template v-if="note.type==='group_invitation' && note.status==='pending' && workspace.pendingInvitations.some(invitation=>invitation.id===note.resourceId)">
            <button class="button primary" type="button" @click="respond(note.resourceId!,true)">{{ tr('acceptInvitation') }}</button>
            <button class="button ghost" type="button" @click="respond(note.resourceId!,false)">{{ tr('declineInvitation') }}</button>
          </template>
          <button class="button ghost" type="button" @click="notifications.setRead(note.id,!note.readAt)">{{ note.readAt ? tr('notificationUnreadAction') : tr('notificationRead') }}</button>
          </template>
        </NotificationItem>
      </li>
    </ul>

    <nav v-if="notifications.meta.totalPages>1" class="pagination" :aria-label="tr('pagination')">
      <button class="button ghost" type="button" :disabled="notifications.meta.page<=1 || notifications.busy" @click="reload(notifications.meta.page-1)">{{ tr('previousPage') }}</button>
      <span>{{ notifications.meta.page }} / {{ notifications.meta.totalPages }}</span>
      <button class="button ghost" type="button" :disabled="notifications.meta.page>=notifications.meta.totalPages || notifications.busy" @click="reload(notifications.meta.page+1)">{{ tr('nextPage') }}</button>
    </nav>
  </main>
</template>

<style scoped>
.notification-center { display: grid; gap: 1rem; min-inline-size: 0; }
.page-heading { display: flex; align-items: flex-start; justify-content: space-between; flex-wrap: wrap; gap: 1rem; }
.page-heading h1 { margin: .2rem 0; }
.page-heading p { margin: .25rem 0; color: var(--muted); }
.notification-filters { display: flex; align-items: center; flex-wrap: wrap; gap: .5rem; min-inline-size: 0; }
.filter-chip, .category-filter select { min-block-size: 40px; padding: .5rem .8rem; border: 1px solid var(--line-strong); border-radius: .6rem; background: var(--surface); color: var(--ink); }
.filter-chip { white-space: nowrap; }
.filter-chip[aria-pressed="true"] { border-color: var(--brand); color: var(--brand); font-weight: 650; }
.category-filter { display: flex; align-items: center; flex: 0 0 auto; gap: .5rem; color: var(--muted); white-space: nowrap; }
.category-filter select { max-inline-size: min(12rem, 45vw); }
.preference-panel { padding: .9rem 1rem; }
.preference-toggle { display: flex; align-items: center; justify-content: space-between; gap: 1rem; inline-size: 100%; padding: .2rem; border: 0; background: transparent; color: var(--ink); font: inherit; font-weight: 700; text-align: start; cursor: pointer; }
.preference-toggle span:last-child { display: grid; place-items: center; inline-size: 2rem; block-size: 2rem; border: 1px solid var(--line); border-radius: 50%; color: var(--brand); font-size: 1.2rem; }
.preference-toggle:focus-visible, .notification-center :deep(button:focus-visible), .notification-center :deep(a:focus-visible), .category-filter select:focus-visible { outline: 2px solid var(--brand); outline-offset: 3px; }
.preference-content { display: grid; gap: .8rem; margin-block-start: .8rem; }
.preference-table-wrap { overflow-x: auto; }
.preference-table { inline-size: 100%; border-collapse: collapse; text-align: start; }
.preference-table th, .preference-table td { padding: .75rem .8rem; border-block-end: 1px solid var(--line); }
.preference-table thead th { color: var(--muted); font-size: .85rem; font-weight: 650; }
.preference-table th:first-child { inline-size: 45%; text-align: start; }
.preference-table tbody th { font-weight: 650; }
.preference-table td { text-align: center; }
.preference-table td label { display: inline-flex; align-items: center; justify-content: center; min-inline-size: 2.75rem; min-block-size: 2.75rem; cursor: pointer; }
.preference-cell-label { display: none; }
.preference-table input { inline-size: 1.1rem; block-size: 1.1rem; accent-color: var(--brand); }
.preference-table input:disabled { cursor: not-allowed; }
.security-note { margin: 0; color: var(--muted); font-size: .9rem; }
.notification-error { display: flex; align-items: center; flex-wrap: wrap; gap: .75rem; overflow-wrap: anywhere; }
.notification-error > span:first-child { flex: 1 1 16rem; min-inline-size: 0; }
.notification-error code { overflow-wrap: anywhere; }
.notification-list { display: grid; gap: .65rem; min-inline-size: 0; padding: 0; margin: 0; list-style: none; }
.notification-list li { min-inline-size: 0; }
.notification-center :deep(.source-link) { color: var(--brand); text-decoration: underline; }
.notification-center :deep(.button) { min-block-size: 40px; padding: .5rem .8rem; border: 1px solid var(--line-strong); border-radius: .55rem; background: var(--surface); color: var(--ink); cursor: pointer; }
.notification-center :deep(.button.primary) { border-color: var(--brand); background: var(--brand); color: #fff; }
.notification-center :deep(.button:disabled) { cursor: not-allowed; opacity: .55; }
.empty-state { display: grid; justify-items: center; gap: .7rem; padding: 2rem 1rem; text-align: center; }
.empty-state h2 { margin: 0; font-size: 1.05rem; }
.empty-state-mark { display: grid; place-items: center; inline-size: 2.5rem; block-size: 2.5rem; border: 1px solid var(--line); border-radius: 50%; color: var(--brand); font-size: 1.3rem; }
.pagination { display: flex; justify-content: center; align-items: center; flex-wrap: wrap; gap: 1rem; }
@media (max-width: 600px) {
  .page-heading > :deep(button) { inline-size: 100%; }
  .notification-filters { align-items: stretch; }
  .category-filter { flex: 1 1 100%; justify-content: space-between; }
  .category-filter select { flex: 0 1 12rem; }
  .preference-table thead { position: absolute; inline-size: 1px; block-size: 1px; padding: 0; margin: -1px; overflow: hidden; clip: rect(0, 0, 0, 0); white-space: nowrap; }
  .preference-table, .preference-table tbody { display: grid; gap: .65rem; }
  .preference-table tr { display: grid; grid-template-columns: minmax(0, 1fr) minmax(5.5rem, auto) minmax(5.5rem, auto); align-items: center; border: 1px solid var(--line); border-radius: .7rem; }
  .preference-table th, .preference-table td { padding: .5rem; border: 0; }
  .preference-table th:first-child { inline-size: auto; }
  .preference-table td label { display: grid; justify-items: center; gap: .3rem; min-inline-size: 2.75rem; min-block-size: 3.5rem; text-align: center; }
  .preference-cell-label { display: block; max-inline-size: 100%; color: var(--muted); font-size: .72rem; line-height: 1.2; overflow-wrap: anywhere; }
}
@media (max-width: 380px) {
  .preference-table tr { grid-template-columns: minmax(0, 1fr) 4.5rem 4.5rem; }
  .preference-table tbody th { overflow-wrap: anywhere; }
}
</style>

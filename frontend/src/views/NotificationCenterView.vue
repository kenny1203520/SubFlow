<script setup lang="ts">
import { computed, onMounted, watch } from 'vue'
import { useNotificationsStore, type NotificationFilter } from '../stores/notifications'
import { useWorkspaceStore } from '../stores/workspace'
import { useI18n } from '../i18n'

const notifications = useNotificationsStore()
const workspace = useWorkspaceStore()
const { tr, formatDate } = useI18n()
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
      <h2 id="notification-preferences-title">{{ tr('notificationPreferences') }}</h2>
      <div class="preference-grid">
        <strong>{{ tr('notificationTask') }}</strong><label><input v-model="notifications.preferences.tasksToast" type="checkbox"> {{ tr('notificationToast') }}</label><label><input v-model="notifications.preferences.tasksEmail" type="checkbox"> {{ tr('notificationEmail') }}</label>
        <strong>{{ tr('notificationException') }}</strong><label><input v-model="notifications.preferences.exceptionsToast" type="checkbox"> {{ tr('notificationToast') }}</label><label><input v-model="notifications.preferences.exceptionsEmail" type="checkbox"> {{ tr('notificationEmail') }}</label>
        <strong>{{ tr('notificationSecurity') }}</strong><label><input checked disabled type="checkbox"> {{ tr('notificationToast') }}</label><label><input checked disabled type="checkbox"> {{ tr('notificationEmail') }}</label>
      </div>
      <p class="security-note">{{ tr('notificationSecurityRequired') }}</p>
      <button class="button primary" type="button" @click="notifications.savePreferences">{{ tr('notificationSavePreferences') }}</button>
    </section>

    <div v-if="notifications.error" class="notice danger" role="alert">{{ tr('notificationLoadFailed') }} <code>{{ notifications.error }}</code><span v-if="notifications.requestId">{{ tr('requestId') }}: <code>{{ notifications.requestId }}</code> <button type="button" @click="copyRequestId">{{ tr('copyErrorReference') }}</button></span> <button type="button" @click="reload(notifications.meta.page)">{{ tr('retry') }}</button></div>
    <div v-else-if="notifications.busy && !notifications.items.length" class="card" role="status" aria-live="polite">{{ tr('loading') }}</div>
    <div v-else-if="!notifications.items.length" class="card empty-state"><h2>{{ tr('notificationEmpty') }}</h2></div>

    <ul v-else class="notification-list" aria-label="Notifications">
      <li v-for="note in notifications.items" :key="note.id" class="card notification-card" :class="{ unread:!note.readAt }">
        <div class="notification-card-main">
          <div class="notification-card-heading"><span class="notification-category">{{ tr(`notification${note.category[0].toUpperCase()}${note.category.slice(1)}`) }}</span><span class="notification-state">{{ statusLabel(note.status) }}</span></div>
          <h2>{{ title(note.type,note.outcome) }}</h2>
          <p v-if="note.message">{{ note.message }}</p>
          <RouterLink v-if="sourcePath(note.type,note.groupId)" class="source-link" :to="sourcePath(note.type,note.groupId)">{{ tr('notificationOpenSource') }}</RouterLink>
          <time :datetime="note.createdAt">{{ formatDate(note.createdAt) }}</time>
        </div>
        <div class="notification-card-actions">
          <template v-if="note.type==='group_invitation' && note.status==='pending' && workspace.pendingInvitations.some(invitation=>invitation.id===note.resourceId)">
            <button class="button primary" type="button" @click="respond(note.resourceId!,true)">{{ tr('acceptInvitation') }}</button>
            <button class="button ghost" type="button" @click="respond(note.resourceId!,false)">{{ tr('declineInvitation') }}</button>
          </template>
          <button class="button ghost" type="button" @click="notifications.setRead(note.id,!note.readAt)">{{ note.readAt ? tr('notificationUnreadAction') : tr('notificationRead') }}</button>
        </div>
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
.notification-center{display:grid;gap:1rem}.page-heading{display:flex;align-items:flex-start;justify-content:space-between;gap:1rem}.page-heading h1{margin:.2rem 0}.page-heading p{margin:.25rem 0;color:var(--muted)}.notification-filters{display:flex;align-items:center;flex-wrap:wrap;gap:.5rem}.filter-chip,.category-filter select{min-height:40px;padding:.5rem .8rem;border:1px solid var(--line-strong);border-radius:.6rem;background:var(--surface);color:var(--ink)}.filter-chip[aria-pressed="true"]{border-color:var(--brand);color:var(--brand)}.category-filter{display:flex;align-items:center;gap:.5rem;color:var(--muted)}.notification-list{display:grid;gap:.75rem;padding:0;margin:0;list-style:none}.notification-card{display:flex;justify-content:space-between;gap:1rem;border:1px solid var(--line)}.notification-card.unread{border-inline-start:4px solid var(--brand)}.notification-card h2{font-size:1rem;margin:.4rem 0}.notification-card p{margin:.2rem 0 .5rem}.notification-card time{font-size:.85rem;color:var(--muted)}.notification-card-heading,.notification-card-actions{display:flex;align-items:center;flex-wrap:wrap;gap:.5rem}.notification-category,.notification-state{font-size:.8rem;padding:.2rem .5rem;border-radius:999px;background:var(--surface-raised);color:var(--muted)}.notification-card-actions{align-content:center}.pagination{display:flex;justify-content:center;align-items:center;gap:1rem}.preference-panel h2{margin-top:0}.preference-grid{display:grid;grid-template-columns:minmax(6rem,1fr) 1fr 1fr;align-items:center;gap:.8rem}.preference-grid label{display:flex;align-items:center;gap:.45rem}.security-note{color:var(--muted)}.empty-state{text-align:center;padding:2rem}@media(max-width:600px){.page-heading,.notification-card{align-items:stretch;flex-direction:column}.notification-card-actions>*{flex:1}.preference-grid{grid-template-columns:1fr 1fr}.preference-grid strong{grid-column:1/-1}}
.source-link{color:var(--brand);text-decoration:underline}
</style>

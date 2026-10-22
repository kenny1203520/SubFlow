<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import BaseDropdown from './BaseDropdown.vue'
import { useI18n } from '../i18n'
import { useWorkspaceStore } from '../stores/workspace'
import { useNotificationsStore } from '../stores/notifications'
import NotificationItem from './notifications/NotificationItem.vue'

const workspace=useWorkspaceStore(),notifications=useNotificationsStore(),router=useRouter(),{tr,formatDate}=useI18n()
const open=ref(false)
const invitationFor=(id:string)=>workspace.pendingInvitations?.find(item=>item.id===id)
async function accept(id:string){const groupId=invitationFor(id)?.groupId;await workspace.acceptPendingInvitation(id);await workspace.loadInvitationInbox();await notifications.refresh();open.value=false;if(groupId)await router.push(`/groups/${groupId}/overview`)}
async function decline(id:string){await workspace.declinePendingInvitation(id);await workspace.loadInvitationInbox();await notifications.refresh()}
async function opened(){notifications.filter='all';notifications.category='';if(await notifications.load(1))await notifications.markVisibleViewed()}
function notificationTitle(note:{type:string;outcome?:string}){if(note.outcome==='accepted')return tr('notificationAccepted');if(note.outcome==='declined')return tr('notificationDeclined');if(note.outcome==='revoked')return tr('notificationRevoked');if(note.outcome==='expired')return tr('notificationExpired');return note.type==='group_invitation'?tr('pendingInvitations'):tr(note.type)}
function categoryLabel(category:string){return tr(`notification${category[0].toUpperCase()}${category.slice(1)}`)}
function statusLabel(status:string){return status==='pending'?tr('notificationPending'):status==='invalid'?tr('notificationExpired'):tr('notificationResolved')}
function sourcePath(type:string,groupId?:string){if(type==='subscription_post_failed'||type==='recurring_income_post_failed')return groupId?`/groups/${groupId}/${type==='subscription_post_failed'?'subscriptions':'incomes'}`:`/personal/${type==='subscription_post_failed'?'subscriptions':'incomes'}`;if(!groupId)return '';if(type==='ownership_transfer')return `/groups/${groupId}/settings`;if(type==='member_transfer')return `/groups/${groupId}/members`;return ''}
onMounted(()=>{void notifications.refreshSummary();void notifications.loadPreferences()})
</script>

<template>
  <BaseDropdown v-model="open" class="notification-bell" :panel-label="tr('notifications')" panel-role="dialog" placement="bottom-end" mobile-sheet @opened="opened">
    <template #trigger="{toggle}">
      <button type="button" class="bell-trigger" :aria-label="tr('notifications')" :aria-expanded="open" @click="toggle">
        <svg aria-hidden="true" viewBox="0 0 24 24"><path d="M18 9a6 6 0 0 0-12 0c0 7-3 7-3 9h18c0-2-3-2-3-9M10 21h4" /></svg>
        <span v-if="notifications.hasNew" class="bell-count new" aria-hidden="true">!</span>
        <span v-else-if="notifications.pendingCount" class="bell-count" aria-hidden="true">{{notifications.pendingCount>99?'99+':notifications.pendingCount}}</span>
        <span class="sr-only">{{ notifications.hasNew ? tr('newNotification') : tr('pendingCount',{count:notifications.pendingCount}) }}</span>
      </button>
    </template>
    <template #default="{close}">
      <section class="notification-menu">
        <header><strong>{{tr('notifications')}}</strong><span v-if="notifications.pendingCount" class="count">{{tr('pendingCount',{count:notifications.pendingCount})}}</span></header>
        <p v-if="notifications.error" class="notice danger" role="alert">{{tr('notificationLoadFailed')}} <button type="button" @click="opened">{{tr('retry')}}</button></p>
        <p v-else-if="notifications.busy && !notifications.items.length" class="empty-inline" role="status" aria-live="polite">{{tr('loading')}}</p>
        <p v-else-if="!notifications.items.length" class="empty-inline">{{tr('notificationEmpty')}}</p>
        <ul v-else class="notification-items" :aria-label="tr('notificationCenter')">
          <li v-for="note in notifications.items.slice(0,5)" :key="note.id">
            <NotificationItem :title="notificationTitle(note)" :category="categoryLabel(note.category)" :status="statusLabel(note.status)" :read-state="note.readAt ? tr('notificationReadState') : tr('notificationUnreadState')" :created-at="note.createdAt" :formatted-date="formatDate(note.createdAt)" :message="note.message" :unread="!note.readAt" variant="preview">
              <template #actions>
              <RouterLink v-if="sourcePath(note.type,note.groupId)" :to="sourcePath(note.type,note.groupId)" @click="close">{{tr('notificationOpenSource')}}</RouterLink>
            <div v-if="note.type==='group_invitation' && note.status==='pending' && invitationFor(note.resourceId||'')" class="notification-actions">
              <button type="button" class="button primary" @click="accept(note.resourceId||'');close()">{{tr('acceptInvitation')}}</button>
              <button type="button" class="button ghost" @click="decline(note.resourceId||'')">{{tr('declineInvitation')}}</button>
            </div>
            <button v-if="!note.readAt" type="button" class="mark-read" @click="notifications.setRead(note.id,true)">{{tr('notificationRead')}}</button>
              </template>
            </NotificationItem>
          </li>
        </ul>
        <footer><RouterLink :to="{name:'notifications'}" @click="close">{{tr('notificationViewAll')}}</RouterLink><button v-if="notifications.summary.unread" type="button" @click="notifications.markAllRead">{{tr('notificationMarkAllRead')}}</button></footer>
      </section>
    </template>
  </BaseDropdown>
</template>

<style scoped>
.notification-bell{position:relative}.bell-trigger{position:relative;display:grid;place-items:center;width:42px;height:42px;border:1px solid var(--line-strong);border-radius:10px;background:var(--surface);color:var(--ink);cursor:pointer}.bell-trigger svg{width:20px;height:20px;fill:none;stroke:currentColor;stroke-width:1.7;stroke-linecap:round;stroke-linejoin:round}.bell-trigger:hover{border-color:var(--brand);color:var(--brand)}.bell-count,.count{display:grid;place-items:center;min-width:18px;height:18px;padding:0 5px;border-radius:99px;background:var(--brand);color:white;font-size:11px}.bell-count{position:absolute;right:-5px;top:-5px}.bell-count.new{font-size:14px;font-weight:800}.notification-menu{display:grid;gap:.6rem;width:min(28rem,calc(100vw - 2rem));padding:.8rem}.notification-menu header,.notification-menu footer{display:flex;justify-content:space-between;align-items:center;gap:.75rem}.count{inline-size:auto;max-inline-size:14rem;padding-inline:.65rem;white-space:nowrap}.notification-items{display:grid;gap:.5rem;max-block-size:min(65dvh,32rem);overflow-y:auto;overscroll-behavior:contain;padding:0;margin:0;list-style:none}.notification-items li{min-inline-size:0}.notification-menu :deep(.notification-item__actions){justify-content:flex-start}.notification-menu :deep(.notification-item__actions a),.notification-menu :deep(.notification-item__actions button),.notification-menu footer button{justify-self:start;background:none;border:0;color:var(--brand);text-decoration:underline;cursor:pointer}.notification-menu :deep(.notification-item__actions .button){min-block-size:34px;padding:.35rem .65rem;border:1px solid var(--line-strong);border-radius:.5rem;background:var(--surface);color:var(--ink);text-decoration:none}.notification-menu :deep(.notification-item__actions .button.primary){background:var(--brand);border-color:var(--brand);color:white}.notification-actions{display:flex;flex-wrap:wrap;gap:.45rem;margin-top:.25rem}.notification-menu footer a{color:var(--brand)}.notification-menu :focus-visible{outline:2px solid var(--brand);outline-offset:2px}.sr-only{position:absolute;width:1px;height:1px;padding:0;margin:-1px;overflow:hidden;clip:rect(0,0,0,0);white-space:nowrap;border:0}
</style>

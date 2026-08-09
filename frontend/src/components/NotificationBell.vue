<script setup lang="ts">
import { computed, nextTick, ref } from 'vue'
import { useRouter } from 'vue-router'
import BaseDropdown from './BaseDropdown.vue'
import { useI18n } from '../i18n'
import { useWorkspaceStore } from '../stores/workspace'

const workspace=useWorkspaceStore(),router=useRouter(),{tr,formatDate}=useI18n()
const open=ref(false)
const unread=computed(()=>workspace.notifications?.filter(item=>!item.readAt).length||0)
const invitationFor=(id:string)=>workspace.pendingInvitations?.find(item=>item.id===id)
async function accept(id:string){const groupId=invitationFor(id)?.groupId;await workspace.acceptPendingInvitation(id);open.value=false;if(groupId)await router.push(`/groups/${groupId}/overview`)}
async function decline(id:string){await workspace.declinePendingInvitation(id)}
function opened(){void workspace.loadInvitationInbox?.();nextTick()}
</script>
<template>
  <BaseDropdown v-model="open" class="notification-bell" :panel-label="tr('notifications')" @opened="opened">
    <template #trigger="{toggle}">
      <button type="button" class="bell-trigger" :aria-label="tr('notifications')" :aria-expanded="open" @click="toggle">
        <span aria-hidden="true">♧</span><span v-if="unread" class="bell-count">{{unread}}</span>
      </button>
    </template>
    <template #default="{close}">
      <section class="notification-menu">
        <header><strong>{{tr('notifications')}}</strong><span v-if="unread" class="count">{{unread}}</span></header>
        <p v-if="!workspace.notifications?.length" class="empty-inline">{{tr('noInvitations')}}</p>
        <article v-for="note in workspace.notifications||[]" :key="note.id" class="notification-item" :class="{unread:!note.readAt}">
          <template v-if="note.type==='group_invitation' && invitationFor(note.resourceId||'')">
            <strong>{{invitationFor(note.resourceId||'')?.groupInfo?.name||tr('pendingInvitations')}}</strong>
            <small>{{tr('pendingInvitations')}} · {{formatDate(note.createdAt)}}</small>
            <div class="notification-actions"><button type="button" class="button primary" @click="accept(note.resourceId||'');close()">{{tr('acceptInvitation')}}</button><button type="button" class="button ghost" @click="decline(note.resourceId||'')">{{tr('declineInvitation')}}</button></div>
          </template>
          <template v-else><strong>{{tr('pendingInvitations')}}</strong><small>{{formatDate(note.createdAt)}}</small></template>
        </article>
      </section>
    </template>
  </BaseDropdown>
</template>
<style scoped>
.notification-bell{position:relative}.bell-trigger{position:relative;display:grid;place-items:center;width:42px;height:42px;border:1px solid var(--line-strong);border-radius:10px;background:var(--surface);color:var(--ink);font-size:18px}.bell-trigger:hover{border-color:var(--brand);color:var(--brand)}.bell-count,.count{display:grid;place-items:center;min-width:18px;height:18px;padding:0 5px;border-radius:99px;background:var(--brand);color:white;font-size:11px}.bell-count{position:absolute;right:-5px;top:-5px}.notification-menu{display:grid;gap:.6rem;width:min(23rem,calc(100vw - 2rem));padding:.8rem}.notification-menu header{display:flex;justify-content:space-between;align-items:center}.notification-item{display:grid;gap:.3rem;padding:.7rem;border:1px solid var(--line);border-radius:.65rem}.notification-item.unread{border-color:var(--brand)}.notification-item small{color:var(--muted)}.notification-actions{display:flex;gap:.45rem;margin-top:.25rem}.button{min-height:34px;padding:.35rem .65rem;border-radius:.5rem;border:1px solid var(--line-strong);background:var(--surface);color:var(--ink)}.button.primary{background:var(--brand);border-color:var(--brand);color:white}.button.ghost{background:transparent}
</style>

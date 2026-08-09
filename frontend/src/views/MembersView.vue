<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useWorkspaceStore } from '../stores/workspace'
import EmptyState from '../components/EmptyState.vue'
import ConfirmDialog from '../components/ConfirmDialog.vue'
import { useI18n } from '../i18n'
import RoleSelect from '../components/RoleSelect.vue'

const workspace = useWorkspaceStore()
const email = ref('')
const pendingRemoval = ref<{userId:string;label:string}>()
const { tr, formatDate } = useI18n()
const canManageMembers=computed(()=>workspace.groupPermissions.includes('*')||workspace.groupPermissions.includes('group.members.manage'))
const canManageRoles=computed(()=>workspace.groupPermissions.includes('*')||workspace.groupPermissions.includes('group.roles.manage'))

async function invite() {
    await workspace.invite(email.value)
    email.value = ''
}

async function removeMember(userId: string, label: string) {
    pendingRemoval.value={userId,label}
}
async function confirmRemoval(){if(!pendingRemoval.value)return;await workspace.removeMember(pendingRemoval.value.userId);pendingRemoval.value=undefined}
async function assignRole(userId:string,roleId:string){if(roleId)await workspace.assignGroupRole(userId,roleId)}
onMounted(()=>{if(canManageRoles.value)void workspace.loadGroupRoles()})
</script>

<template>
    <section class="page">
        <div class="page-heading">
            <div>
                <p class="eyebrow">{{tr('members')}}</p>
                <h1>{{tr('membersInvites')}}</h1>
                <p>{{tr('membersDesc')}}</p>
            </div>
        </div>
        <div class="two-column">
            <div class="card">
                <h2>{{tr('groupMembers')}}</h2>
                <div v-if="workspace.members.length" class="rows">
                    <div v-for="member in workspace.members" :key="member.id" class="row">
                        <div class="avatar">{{ (member.user?.name || member.user?.email || '?').slice(0,
                            1).toUpperCase() }}</div>
                        <div class="grow"><strong>{{ member.user?.name || tr('unnamedMember') }}</strong><small>{{
                                member.user?.email }}</small></div>
                        <span class="pill">{{ member.roleName || tr(member.role) }}</span>
                        <RoleSelect v-if="canManageRoles && member.role !== 'owner' && workspace.groupRoles.length" :model-value="member.roleId||''" :roles="workspace.groupRoles" :label="tr('assignRole')" @update:model-value="assignRole(member.userId,$event)" />
                        <button v-if="canManageMembers && member.role !== 'owner'" class="ghost danger-text"
                            @click="removeMember(member.userId, member.user?.name || member.user?.email || tr('thisMember'))">{{tr('remove')}}</button>
                    </div>
                </div>
                <EmptyState v-else :title="tr('noMemberData')" :description="tr('noMemberDataDesc')" />
            </div>
            <div v-if="canManageMembers">
                <form class="card form-card" @submit.prevent="invite">
                    <h2>{{tr('inviteMember')}}</h2>
                    <label>Email<input v-model="email" type="email" required placeholder="friend@example.com"></label>
                    <button class="primary" :disabled="workspace.loading">{{tr('sendInvitation')}}</button>
                </form>
                <div class="card invitations">
                    <h2>{{tr('invitationHistory')}}</h2>
                    <div v-if="workspace.invitations.length">
                        <div v-for="item in workspace.invitations" :key="item.id" class="invite">
                            <div><strong>{{ item.email }}</strong><small>{{ tr(item.status==='delivery_failed'?'deliveryFailed':item.status) }} · {{ formatDate(item.expiresAt) }}</small><a v-if="item.debugUrl"
                                    :href="item.debugUrl">{{tr('developmentLink')}}</a></div>
                            <div v-if="item.status === 'pending' || item.status === 'delivery_failed'" class="actions">
                                <button class="ghost" @click="workspace.resendInvitation(item.id)">{{tr('resend')}}</button>
                                <button class="ghost danger-text"
                                    @click="workspace.revokeInvitation(item.id)">{{tr('revoke')}}</button>
                            </div>
                        </div>
                    </div>
                    <p v-else class="empty-inline">{{tr('noInvitations')}}</p>
                </div>
            </div>
            <div v-else class="card owner-note">
                <h2>{{tr('memberManagement')}}</h2>
                <p>{{tr('memberManagePermissionRequired')}}</p>
            </div>
        </div>
        <ConfirmDialog :open="!!pendingRemoval" :title="pendingRemoval?tr('removeMemberConfirm',{name:pendingRemoval.label}):''" danger @cancel="pendingRemoval=undefined" @confirm="confirmRemoval"/>
    </section>
</template>

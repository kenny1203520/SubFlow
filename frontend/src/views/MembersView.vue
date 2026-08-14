<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useWorkspaceStore } from '../stores/workspace'
import { useAuthStore } from '../stores/auth'
import EmptyState from '../components/EmptyState.vue'
import ConfirmDialog from '../components/ConfirmDialog.vue'
import Pagination from '../components/Pagination.vue'
import PageSizeSelect from '../components/PageSizeSelect.vue'
import { useI18n } from '../i18n'
import RoleSelect from '../components/RoleSelect.vue'
import { roleLabel } from '../role'
import { defaultPageSize } from '../pageSize'

const workspace = useWorkspaceStore()
const auth = useAuthStore()
const email = ref('')
const tempMemberName = ref('')
const bindTarget = ref<{userId:string;label:string}>()
const pendingRemoval = ref<{userId:string;label:string}>()
const invitationsPageSize = ref(defaultPageSize.value)
const { tr, formatDate } = useI18n()
const canManageMembers=computed(()=>workspace.groupPermissions.includes('*')||workspace.groupPermissions.includes('group.members.manage'))
const canManageRoles=computed(()=>workspace.groupPermissions.includes('*')||workspace.groupPermissions.includes('group.roles.manage'))
// The owner role can only change hands via the ownership-transfer flow
// below, not this generic per-member role assignment (the backend rejects
// it), so it must not be offered here as a selectable option.
const assignableRoles=computed(()=>workspace.groupRoles.filter(role=>role.key!=='owner'))

function memberLabel(userId?:string) { const member=workspace.members.find(value=>value.userId===userId); return member?.user?.name||member?.user?.email||tr('thisMember') }
const isTransferTarget = computed(()=>workspace.ownershipTransfer?.status==='pending'&&workspace.ownershipTransfer.toUserId===auth.record?.id)
// Placeholders can't sign in, so they can't hold the owner slot.
const transferCandidates = computed(()=>workspace.members.filter(member=>member.userId!==auth.record?.id&&!member.user?.placeholder))
const transferTarget = ref('')
async function startTransfer() { if(!transferTarget.value) return; if(await workspace.createOwnershipTransfer(transferTarget.value)) transferTarget.value='' }
async function cancelTransfer() { if(workspace.ownershipTransfer) await workspace.cancelOwnershipTransfer(workspace.ownershipTransfer.id) }
async function respondTransfer(accept:boolean) { if(workspace.ownershipTransfer) await workspace.respondOwnershipTransfer(workspace.ownershipTransfer.id, accept) }

async function invite() {
    await workspace.invite(email.value, bindTarget.value?.userId)
    email.value = ''
    bindTarget.value = undefined
}
function startBinding(userId:string, label:string) { bindTarget.value = { userId, label } }
async function addTempMember() {
    if (!tempMemberName.value.trim()) return
    await workspace.createTempMember(tempMemberName.value.trim())
    tempMemberName.value = ''
}
function setInvitationsPage(page:number) { void workspace.loadInvitations(page, invitationsPageSize.value) }
function setInvitationsPageSize(value:number) { invitationsPageSize.value = value; void workspace.loadInvitations(1, value) }

async function removeMember(userId: string, label: string) {
    pendingRemoval.value={userId,label}
}
async function confirmRemoval(){if(!pendingRemoval.value)return;await workspace.removeMember(pendingRemoval.value.userId);pendingRemoval.value=undefined}
async function assignRole(userId:string,roleId:string){if(roleId)await workspace.assignGroupRole(userId,roleId)}
function memberRoleLabel(roleId?:string, fallback?:string, legacyRole?:string) {
    const role = workspace.groupRoles.find(value => value.id === roleId)
    return roleLabel(role, tr) || (legacyRole === 'owner' ? tr('owner') : legacyRole === 'member' ? tr('member') : fallback || tr('member'))
}
onMounted(()=>{if(canManageRoles.value)void workspace.loadGroupRoles();void workspace.loadOwnershipTransfer()})
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
                        <span class="pill">{{ memberRoleLabel(member.roleId, member.roleName, member.role) }}</span>
                        <span v-if="member.user?.placeholder" class="pill">{{ member.user?.linkedUserId ? tr('tempMemberBound') : tr('tempMember') }}</span>
                        <RoleSelect v-if="canManageRoles && member.role !== 'owner' && workspace.groupRoles.length" :model-value="member.roleId||''" :roles="assignableRoles" :label="tr('assignRole')" @update:model-value="assignRole(member.userId,$event)" />
                        <button v-if="canManageMembers && member.user?.placeholder && !member.user?.linkedUserId" class="ghost"
                            @click="startBinding(member.userId, member.user?.name || tr('thisMember'))">{{tr('bindTempMember')}}</button>
                        <button v-if="canManageMembers && member.role !== 'owner'" class="ghost danger-text"
                            @click="removeMember(member.userId, member.user?.name || member.user?.email || tr('thisMember'))">{{tr('remove')}}</button>
                    </div>
                </div>
                <EmptyState v-else :title="tr('noMemberData')" :description="tr('noMemberDataDesc')" />
            </div>
            <div v-if="workspace.isOwner || isTransferTarget" class="card">
                <h2>{{tr('transferOwnership')}}</h2>
                <p class="field-help">{{tr('transferOwnershipDesc')}}</p>
                <div v-if="workspace.ownershipTransfer" class="notice inline">
                    <template v-if="workspace.isOwner">
                        <p>{{tr('transferPendingFor',{name:memberLabel(workspace.ownershipTransfer.toUserId)})}}</p>
                        <button class="ghost danger-text" :disabled="workspace.loading" @click="cancelTransfer">{{tr('cancelTransfer')}}</button>
                    </template>
                    <template v-else>
                        <p>{{tr('transferOfferedBy',{name:memberLabel(workspace.ownershipTransfer.fromUserId)})}}</p>
                        <button class="primary" :disabled="workspace.loading" @click="respondTransfer(true)">{{tr('acceptTransfer')}}</button>
                        <button class="ghost" :disabled="workspace.loading" @click="respondTransfer(false)">{{tr('declineTransfer')}}</button>
                    </template>
                </div>
                <form v-else-if="workspace.isOwner" class="form-card" @submit.prevent="startTransfer">
                    <label>{{tr('transferTo')}}
                        <select v-model="transferTarget" required>
                            <option value="" disabled>{{tr('chooseMember')}}</option>
                            <option v-for="member in transferCandidates" :key="member.userId" :value="member.userId">{{member.user?.name || member.user?.email}}</option>
                        </select>
                    </label>
                    <button class="primary" :disabled="workspace.loading || !transferTarget">{{tr('startTransfer')}}</button>
                </form>
            </div>
            <div v-if="canManageMembers" class="card">
                <h2>{{tr('addTempMember')}}</h2>
                <p class="field-help">{{tr('addTempMemberDesc')}}</p>
                <form class="form-card" @submit.prevent="addTempMember">
                    <label>{{tr('displayName')}}<input v-model="tempMemberName" required :placeholder="tr('addTempMember')"></label>
                    <button class="primary" :disabled="workspace.loading||!workspace.online" :title="workspace.online?'':tr('offlineActionDisabled')">{{tr('addTempMember')}}</button>
                </form>
            </div>
            <div v-if="canManageMembers">
                <form class="card form-card" @submit.prevent="invite">
                    <h2>{{tr('inviteMember')}}</h2>
                    <div v-if="bindTarget" class="notice inline">{{tr('bindingInviteNotice',{name:bindTarget.label})}} <button type="button" @click="bindTarget=undefined">{{tr('cancel')}}</button></div>
                    <label>Email<input v-model="email" type="email" required placeholder="friend@example.com"></label>
                    <button class="primary" :disabled="workspace.loading||!workspace.online" :title="workspace.online?'':tr('offlineActionDisabled')">{{tr('sendInvitation')}}</button>
                </form>
                <div class="card invitations">
                    <div class="audit-list-heading">
                        <h2>{{tr('invitationHistory')}}</h2>
                        <span v-if="workspace.invitationsMeta.totalItems" class="pill">{{ tr('auditResults', { count: workspace.invitationsMeta.totalItems }) }}</span>
                        <PageSizeSelect :model-value="invitationsPageSize" @update:model-value="setInvitationsPageSize" />
                    </div>
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
                    <Pagination :meta="workspace.invitationsMeta" @page="setInvitationsPage" />
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

<script setup lang="ts">
import { ref, computed } from 'vue';
import { socket } from '../socket';
import { useI18n } from 'vue-i18n';
import { useUIStore } from '../stores/ui';

const { t } = useI18n();
const ui = useUIStore();

const props = defineProps<{
    groupId: string;
    members: any[];
    currentUserRole: string;
    permissions: any;
}>();

const emit = defineEmits<{
    refresh: [];
}>();

const inviteType = ref('username');
const inviteValue = ref('');

const activeMembers = computed(() => 
    props.members.filter(m => m.status === 'active' || !m.status || m.joined_at)
);

const pendingMembers = computed(() => 
    props.members.filter(m => m.status === 'invited' && !m.joined_at)
);

const canManageMembers = computed(() => props.permissions.invite?.members || props.permissions.remove?.members);

const inviteMember = () => {
    if (!inviteValue.value) return;

    const payload: any = { groupId: props.groupId };
    
    if (inviteType.value === 'temp') {
        payload.name = inviteValue.value;
    } else {
        payload[inviteType.value] = inviteValue.value;
    }

    socket.emit('group:add_member', payload, (res: any) => {
        if (res.status === 'ok') {
            ui.alert(t('groups.inviteSent'));
            inviteValue.value = '';
            emit('refresh');
        } else {
            ui.alert(res.message);
        }
    });
};

const bindAccount = (memberId: string) => {
    const input = prompt(t('groups.bindAccountPrompt'));
    if (!input) return;

    const payload: any = { groupId: props.groupId, memberId };
    if (input.includes('@')) payload.email = input;
    else payload.username = input;

    socket.emit('group:bind_member_invite', payload, (res: any) => {
        if (res.status === 'ok') ui.alert(t('groups.inviteSent'));
        else ui.alert(res.message);
    });
};

const removeMember = async (memberId: string) => {
    if (!await ui.confirm(t('groups.confirmKick'))) return;
    
    socket.emit('group:remove_member', { groupId: props.groupId, memberId }, (res: any) => {
        if (res.status === 'ok') emit('refresh');
        else ui.alert(res.message);
    });
};

const cancelInvite = async (memberId: string) => {
    if (!await ui.confirm(t('groups.confirmKick'))) return;
    
    socket.emit('group:cancel_invite', { groupId: props.groupId, memberId }, (res: any) => {
        if (res.status === 'ok') emit('refresh');
        else ui.alert(res.message);
    });
};

const getRoleBadgeClass = (role: string) => {
    switch (role) {
        case 'owner': return 'bg-purple-100 text-purple-700 border-purple-200';
        case 'admin': return 'bg-blue-100 text-blue-700 border-blue-200';
        case 'treasurer': return 'bg-green-100 text-green-700 border-green-200';
        case 'viewer': return 'bg-gray-100 text-gray-700 border-gray-200';
        default: return 'bg-slate-100 text-slate-700 border-slate-200';
    }
};
</script>

<template>
    <div class="members-tab">
        <!-- Invite Section -->
        <div v-if="canManageMembers" class="card mb-6">
            <h3 class="text-lg font-bold text-slate-800 mb-4">{{ t('groups.inviteNewMember') }}</h3>
            
            <div class="grid grid-cols-1 sm:grid-cols-3 gap-4 mb-4">
                <label class="invite-type-option">
                    <input type="radio" v-model="inviteType" value="username" class="sr-only peer">
                    <div class="option-card">
                        <div class="text-2xl mb-2">👤</div>
                        <div class="font-semibold">{{ t('groups.inviteUsername') }}</div>
                        <div class="text-xs text-slate-500 mt-1">{{ t('groups.inviteUsernameDesc') }}</div>
                    </div>
                </label>

                <label class="invite-type-option">
                    <input type="radio" v-model="inviteType" value="email" class="sr-only peer">
                    <div class="option-card">
                        <div class="text-2xl mb-2">📧</div>
                        <div class="font-semibold">{{ t('groups.inviteEmail') }}</div>
                        <div class="text-xs text-slate-500 mt-1">{{ t('groups.inviteEmailDesc') }}</div>
                    </div>
                </label>

                <label class="invite-type-option">
                    <input type="radio" v-model="inviteType" value="temp" class="sr-only peer">
                    <div class="option-card">
                        <div class="text-2xl mb-2">➕</div>
                        <div class="font-semibold">{{ t('groups.tempMember') }}</div>
                        <div class="text-xs text-slate-500 mt-1">{{ t('groups.tempMemberDesc') }}</div>
                    </div>
                </label>
            </div>

            <div class="flex gap-2">
                <input 
                    v-model="inviteValue" 
                    :placeholder="t('groups.invitePlaceholder')"
                    class="input-field flex-1"
                    @keyup.enter="inviteMember"
                />
                <button @click="inviteMember" class="btn btn-primary">
                    {{ t('groups.inviteMember') }}
                </button>
            </div>
        </div>

        <!-- Active Members -->
        <div class="card mb-6">
            <h3 class="text-lg font-bold text-slate-800 mb-4">
                {{ t('groups.members') }} ({{ activeMembers.length }})
            </h3>
            
            <div v-if="activeMembers.length === 0" class="text-center py-8 text-slate-500">
                {{ t('groups.noMembersYet') }}
            </div>

            <div class="space-y-3">
                <div 
                    v-for="member in activeMembers" 
                    :key="member.member_id"
                    class="member-card"
                >
                    <div class="flex items-center gap-4 flex-1">
                        <div class="w-12 h-12 rounded-xl bg-gradient-to-br from-primary-400 to-primary-600 flex items-center justify-center text-white font-bold text-lg shrink-0">
                            {{ (member.username || member.temp_name || 'U')[0].toUpperCase() }}
                        </div>
                        
                        <div class="flex-1 min-w-0">
                            <div class="font-semibold text-slate-800 truncate">
                                {{ member.username || member.temp_name || member.display_name }}
                            </div>
                            <div class="text-sm text-slate-500 truncate">
                                {{ member.email || t('groups.nonMember') }}
                            </div>
                        </div>

                        <div class="flex items-center gap-2">
                            <span :class="['badge', getRoleBadgeClass(member.role)]">
                                {{ t(`groups.roles.${member.role}`) }}
                            </span>
                            
                            <button 
                                v-if="!member.user_id && canManageMembers"
                                @click="bindAccount(member.member_id)"
                                class="btn btn-sm btn-outline"
                            >
                                {{ t('groups.bindAccount') }}
                            </button>
                            
                            <button 
                                v-if="member.role !== 'owner' && canManageMembers"
                                @click="removeMember(member.member_id)"
                                class="btn btn-sm btn-danger-outline"
                            >
                                {{ t('groups.kickMember') }}
                            </button>
                        </div>
                    </div>
                </div>
            </div>
        </div>

        <!-- Pending Invitations -->
        <div v-if="pendingMembers.length > 0" class="card">
            <h3 class="text-lg font-bold text-slate-800 mb-4">
                {{ t('groups.inviteNewMember') }} ({{ pendingMembers.length }})
            </h3>

            <div class="space-y-3">
                <div 
                    v-for="member in pendingMembers" 
                    :key="member.member_id"
                    class="member-card opacity-60"
                >
                    <div class="flex items-center gap-4 flex-1">
                        <div class="w-12 h-12 rounded-xl bg-slate-200 flex items-center justify-center text-slate-500 font-bold text-lg shrink-0">
                            {{ (member.username || member.temp_name || '?')[0].toUpperCase() }}
                        </div>
                        
                        <div class="flex-1 min-w-0">
                            <div class="font-semibold text-slate-600 truncate">
                                {{ member.username || member.temp_name }}
                            </div>
                            <div class="text-sm text-slate-400">
                                {{ t('groups.status_pending') }}
                            </div>
                        </div>

                        <button 
                            v-if="canManageMembers"
                            @click="cancelInvite(member.member_id)"
                            class="btn btn-sm btn-danger-outline"
                        >
                            {{ t('common.cancel') }}
                        </button>
                    </div>
                </div>
            </div>
        </div>
    </div>
</template>

<style scoped>
.members-tab {
    animation: fadeIn 0.3s ease;
}

.invite-type-option .option-card {
    padding: 1rem;
    border-radius: 0.75rem;
    border: 2px solid #e2e8f0;
    background-color: white;
    cursor: pointer;
    transition: all 0.3s ease;
    text-align: center;
}

.invite-type-option input:checked ~ .option-card {
    border-color: hsl(250, 80%, 60%);
    background-color: hsl(250, 100%, 97%);
    box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.1);
}

.invite-type-option .option-card:hover {
    border-color: hsl(250, 90%, 80%);
    background-color: hsl(250, 100%, 98%);
}

.member-card {
    display: flex;
    align-items: center;
    gap: 1rem;
    padding: 1rem;
    border-radius: 0.75rem;
    background-color: white;
    border: 1px solid #e2e8f0;
    transition: all 0.3s ease;
}

.member-card:hover {
    border-color: hsl(250, 95%, 88%);
    box-shadow: 0 1px 3px -1px rgba(0, 0, 0, 0.1);
}

.badge {
    padding: 0.25rem 0.75rem;
    border-radius: 9999px;
    font-size: 0.75rem;
    font-weight: 600;
    border: 1px solid currentColor;
}
</style>

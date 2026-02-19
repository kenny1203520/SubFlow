<script setup lang="ts">
import { ref, computed, onMounted } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { socket } from '../socket';
import { useAuthStore } from '../stores/auth';
import { useI18n } from 'vue-i18n';
import MainLayout from './MainLayout.vue';
import AddExpenseModal from '../components/AddExpenseModal.vue';
import FileManager from '../components/FileManager.vue';

const { t } = useI18n();
const authStore = useAuthStore();
const route = useRoute();
const router = useRouter();
const groupId = route.params.id as string;

// State
const initialLoading = ref(true);
const error = ref('');
const group = ref<any>(null);
const members = ref<any[]>([]);
const expenses = ref<any[]>([]);
const splits = ref<any[]>([]);
const currentTab = ref('overview');

// Invite State
const inviteType = ref('username');
const inviteValue = ref('');

// Modals
const showAddExpense = ref(false);

const fetchData = (isRefreshing = false) => {
    if (!isRefreshing) initialLoading.value = true;

    // Parallel fetch
    socket.emit('group:get', { groupId }, (res: any) => {
        if (res.status === 'ok') {
            group.value = res.group;
            members.value = res.members;
        } else {
            error.value = res.message;
        }
        if (!isRefreshing) initialLoading.value = false;
    });

    socket.emit('expense:list', { groupId }, (res: any) => {
        if (res.status === 'ok') expenses.value = res.expenses;
    });

    socket.emit('expense:get_splits', { groupId }, (res: any) => {
        if (res.status === 'ok') splits.value = res.splits;
    });
};

onMounted(() => {
    if (socket.connected) fetchData();
    else socket.once('connect', () => fetchData());
});

// Actions
const handleExpenseAdded = () => {
    showAddExpense.value = false;
    fetchData(true); // Refresh without full loader
};

const inviteMember = () => {
    if (!inviteValue.value) return;

    const payload: any = { groupId };
    if (inviteType.value === 'temp') {
        payload.name = inviteValue.value;
        // Backend expects 'name' for temp, 'username'/'email' for invite
        socket.emit('group:add_member', payload, (res: any) => {
            if (res.status === 'ok') {
                inviteValue.value = '';
                fetchData(true);
            } else {
                alert(res.message);
            }
        });
    } else {
        // Invite logic (username/email)
        payload[inviteType.value] = inviteValue.value;
        socket.emit('group:add_member', payload, (res: any) => {
            if (res.status === 'ok') {
                alert(t('groups.inviteSent', 'Invitation sent!'));
                inviteValue.value = '';
            } else {
                alert(res.message);
            }
        });
    }
};

const bindAccount = (memberId: string) => {
    const input = prompt(t('groups.bindAccountPrompt', 'Enter username or email:'));
    if (!input) return;

    const payload: any = { groupId, memberId };
    if (input.includes('@')) payload.email = input;
    else payload.username = input;

    socket.emit('group:bind_member_invite', payload, (res: any) => {
        if (res.status === 'ok') alert(t('groups.inviteSent'));
        else alert(res.message);
    });
};

const settleExpense = (expenseId: string, userId: string) => {
    if (!confirm(t('groups.confirmSettle'))) return;
    socket.emit('expense:settle', { expenseId, userId }, (res: any) => {
        if (res.status === 'ok') fetchData(true);
        else alert(res.message);
    });
};

const deleteGroup = () => {
    if (!confirm(t('groups.confirmDelete'))) return;
    socket.emit('group:delete', { groupId }, (res: any) => {
        if (res.status === 'ok') router.push('/groups');
        else alert(res.message);
    });
};

const leaveGroup = () => {
    if (!confirm(t('groups.confirmLeave'))) return;
    socket.emit('group:leave', { groupId }, (res: any) => {
        if (res.status === 'ok') router.push('/groups');
        else alert(res.message);
    });
};

const getMemberName = (userId: string) => {
    const m = members.value.find(m => m.user_id === userId);
    return m ? (m.username || m.temp_name) : 'Unknown';
};

const canSettle = (split: any) => {
    // Basic check: Auth user is the payer OR the group creator
    return split.user_id === authStore.user?.id || group.value?.created_by === authStore.user?.id;
};
</script>

<template>
    <MainLayout>
        <!-- Initial Loading State (Full Screen) -->
        <div v-if="initialLoading" class="flex flex-col items-center justify-center p-20 min-h-screen">
            <div class="animate-spin rounded-full h-12 w-12 border-b-2 border-primary-600 mb-4"></div>
            <p class="text-slate-500 animate-pulse">{{ t('common.status.loading') }}</p>
        </div>

        <div v-else-if="error" class="flex flex-col items-center justify-center p-20 min-h-screen text-center">
            <p class="text-red-500 font-bold text-lg mb-2">{{ t('common.status.error') }}</p>
            <p class="text-slate-600">{{ error }}</p>
            <button @click="router.push('/groups')" class="mt-4 btn btn-primary">{{ t('groups.backToGroups')
                }}</button>
        </div>

        <div class="group-detail animate-fade-in" v-else-if="group">
            <!-- Header Area -->
            <header class="flex flex-col md:flex-row justify-between items-start md:items-center gap-4 mb-8">
                <div class="flex items-center gap-4">
                    <button @click="router.push('/groups')"
                        class="p-2 rounded-xl bg-white/50 hover:bg-white text-slate-500 hover:text-primary-600 transition-all shadow-sm">
                        <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" fill="none" viewBox="0 0 24 24"
                            stroke="currentColor">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                                d="M10 19l-7-7 7-7m-7 7h18" />
                        </svg>
                    </button>

                    <!-- Group Icon -->
                    <div
                        class="h-16 w-16 rounded-2xl bg-white shadow-sm border border-slate-100 flex items-center justify-center overflow-hidden shrink-0">
                        <img v-if="group.icon_url" :src="group.icon_url" alt="Icon"
                            class="w-full h-full object-cover" />
                        <span v-else class="text-2xl font-bold text-primary-500">{{
                            group.name.charAt(0).toUpperCase()
                            }}</span>
                    </div>

                    <div>
                        <h1 class="text-3xl font-extrabold text-slate-800">{{ group.name }}</h1>
                        <div class="flex items-center gap-2 text-sm text-slate-500">
                            <span
                                class="px-2 py-0.5 rounded-full bg-primary-50 text-primary-700 font-bold uppercase text-xs">{{
                                    group.service_name || t('groups.genericService') }}</span>
                            <span>•</span>
                            <span v-if="group.billing_type === 'recurring'">
                                {{ group.currency }} {{ group.amount }} / {{
                                    t(`common.cycles.${group.interval_unit}`)
                                }}
                            </span>
                            <span v-else>
                                {{ group.currency }} {{ group.amount }} ({{ t('common.cycles.once') }})
                            </span>
                        </div>
                    </div>
                </div>
                <!-- Top Actions -->
                <div class="flex gap-3">
                    <button @click="showAddExpense = true"
                        class="btn btn-primary shadow-lg hover:shadow-xl hover:-translate-y-0.5 transition-all">
                        <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" fill="none" viewBox="0 0 24 24"
                            stroke="currentColor">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                                d="M12 6v6m0 0v6m0-6h6m-6 0H6" />
                        </svg>
                        {{ t('groups.addExpense') }}
                    </button>
                    <!-- More Actions Dropdown (Export, etc.) could go here -->
                </div>
            </header>

            <!-- Tabs Navigation -->
            <div class="border-b border-slate-200 mb-6 flex gap-6">
                <button @click="currentTab = 'overview'"
                    :class="['pb-3 px-2 text-sm font-bold transition-all border-b-2', currentTab === 'overview' ? 'border-primary-500 text-primary-600' : 'border-transparent text-slate-500 hover:text-slate-700']">
                    {{ t('groups.overview', 'Overview') }}
                </button>
                <button @click="currentTab = 'members'"
                    :class="['pb-3 px-2 text-sm font-bold transition-all border-b-2', currentTab === 'members' ? 'border-primary-500 text-primary-600' : 'border-transparent text-slate-500 hover:text-slate-700']">
                    {{ t('groups.members', 'Members') }}
                </button>
                <button @click="currentTab = 'settings'"
                    :class="['pb-3 px-2 text-sm font-bold transition-all border-b-2', currentTab === 'settings' ? 'border-primary-500 text-primary-600' : 'border-transparent text-slate-500 hover:text-slate-700']">
                    {{ t('groups.settings', 'Settings') }}
                </button>
            </div>

            <!-- Tab Content -->
            <div class="min-h-[400px]">
                <!-- TAB: OVERVIEW -->
                <div v-show="currentTab === 'overview'" class="animate-fade-in space-y-8">
                    <!-- Info Cards -->
                    <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
                        <div class="glass-panel p-4 flex flex-col justify-center">
                            <span class="text-xs uppercase font-bold text-slate-400 mb-1">{{ t('groups.planName')
                                }}</span>
                            <span class="text-lg font-bold text-slate-800">{{ group.plan_name || '---' }}</span>
                        </div>
                        <div class="glass-panel p-4 flex flex-col justify-center">
                            <span class="text-xs uppercase font-bold text-slate-400 mb-1">{{
                                t('groups.billingMethod')
                                }}</span>
                            <span class="text-lg font-bold text-slate-800 flex items-center gap-2">
                                {{ t(`groups.methods.${group.billing_method}`) }}
                            </span>
                        </div>
                        <!-- Next Payment -->
                        <div
                            class="glass-panel p-4 flex flex-col justify-center bg-primary-50 border border-primary-100">
                            <span class="text-xs uppercase font-bold text-primary-400 mb-1">{{
                                t('groups.nextPayment') }}</span>
                            <span class="text-lg font-bold text-primary-800">
                                {{ group.next_payment_date ? new Date(group.next_payment_date).toLocaleDateString()
                                    : '---' }}
                            </span>
                        </div>
                    </div>

                    <div class="grid grid-cols-1 lg:grid-cols-3 gap-8">
                        <div class="lg:col-span-2 space-y-8">
                            <!-- Expenses List -->
                            <section>
                                <div class="flex justify-between items-center mb-4">
                                    <h2 class="text-xl font-bold text-slate-800">{{ t('groups.expenses') }}</h2>
                                    <button @click="showAddExpense = true"
                                        class="text-sm font-bold text-primary-600 hover:underline">
                                        {{ t('common.actions.add') }}
                                    </button>
                                </div>
                                <div v-if="expenses.length === 0"
                                    class="text-center p-8 border-2 border-dashed border-slate-200 rounded-xl">
                                    <p class="text-slate-400">{{ t('groups.noExpenses') }}</p>
                                </div>
                                <div v-else class="space-y-3">
                                    <div v-for="expense in expenses" :key="expense.id"
                                        class="glass-card p-4 flex justify-between items-center">
                                        <div>
                                            <p class="font-bold text-slate-800">{{ expense.description }}</p>
                                            <p class="text-xs text-slate-500">
                                                <span class="font-medium text-slate-600">{{
                                                    getMemberName(expense.paid_by) }}</span> •
                                                {{ new Date(expense.date).toLocaleDateString() }}
                                            </p>
                                        </div>
                                        <div class="font-bold text-slate-700">
                                            $ {{ expense.amount }}
                                        </div>
                                    </div>
                                </div>
                            </section>

                            <!-- Debts (Splits) -->
                            <section>
                                <h2 class="text-xl font-bold text-slate-800 mb-4">{{ t('groups.unpaidDebts') }}</h2>
                                <div v-if="splits.length === 0"
                                    class="p-8 bg-green-50 rounded-xl text-center border border-green-100">
                                    <p class="text-green-600 font-bold">{{ t('groups.settled') }}</p>
                                </div>
                                <div v-else class="grid grid-cols-1 md:grid-cols-2 gap-4">
                                    <div v-for="split in splits" :key="split.id"
                                        class="glass-card p-4 border-l-4 border-yellow-400">
                                        <div class="text-sm mb-2">
                                            <span class="font-bold">{{ split.temp_name || split.real_username
                                                }}</span>
                                            <span class="text-slate-400 mx-1">{{ t('groups.owe') }}</span>
                                            <span class="font-bold">{{ split.payer_name }}</span>
                                        </div>
                                        <div class="flex justify-between items-end">
                                            <span class="text-xl font-bold text-red-500">$ {{ split.amount_owed
                                                }}</span>
                                            <button v-if="canSettle(split)"
                                                @click="settleExpense(split.expense_id, split.user_id)"
                                                class="text-xs bg-green-100 text-green-700 px-2 py-1 rounded hover:bg-green-200 font-bold">
                                                {{ t('groups.settle') }}
                                            </button>
                                        </div>
                                    </div>
                                </div>
                            </section>
                        </div>

                        <!-- Right Sidebar (Files) -->
                        <aside>
                            <FileManager :group-id="groupId" />
                        </aside>
                    </div>
                </div>

                <!-- TAB: MEMBERS -->
                <div v-show="currentTab === 'members'" class="animate-fade-in">
                    <div class="flex justify-between items-center mb-6">
                        <h2 class="text-xl font-bold text-slate-800">{{ t('groups.members') }} ({{ members.length
                            }})</h2>
                    </div>

                    <!-- Enhanced Add Member Area -->
                    <div class="glass-panel p-6 mb-8 border border-primary-100">
                        <h3 class="text-sm font-bold text-primary-700 uppercase mb-4">{{ t('groups.inviteNewMember')
                            }}</h3>
                        <div class="flex flex-col md:flex-row gap-4 items-end">
                            <div class="flex-1 w-full">
                                <label class="block text-xs font-bold text-slate-500 uppercase mb-1 ml-1">{{
                                    t('common.fields.type') }}</label>
                                <select v-model="inviteType" class="glass-input w-full">
                                    <option value="username">{{ t('common.fields.username') }}</option>
                                    <option value="email">{{ t('common.fields.email') }}</option>
                                    <option value="temp">{{ t('common.fields.tempMember') }}</option>
                                </select>
                            </div>
                            <div class="flex-[2] w-full">
                                <label class="block text-xs font-bold text-slate-500 uppercase mb-1 ml-1">{{
                                    t('common.fields.value') }}</label>
                                <input v-model="inviteValue" type="text"
                                    :placeholder="inviteType === 'temp' ? t('groups.tempNamePlaceholder') : t('groups.invitePlaceholder')"
                                    class="glass-input w-full" @keyup.enter="inviteMember" />
                            </div>
                            <button @click="inviteMember" :disabled="!inviteValue"
                                class="btn btn-primary w-full md:w-auto h-[42px]">
                                {{ inviteType === 'temp' ? t('common.actions.add') : t('common.actions.invite') }}
                            </button>
                        </div>
                        <p class="text-xs text-slate-400 mt-2">
                            <span v-if="inviteType === 'temp'">{{ t('groups.tempMemberDesc') }}</span>
                            <span v-else>{{ t('groups.inviteDesc') }}</span>
                        </p>
                    </div>

                    <!-- Member List -->
                    <div class="space-y-3">
                        <div v-for="member in members" :key="member.member_id"
                            class="glass-card p-4 flex items-center gap-4">
                            <div
                                class="w-12 h-12 rounded-full bg-slate-200 flex items-center justify-center font-bold text-slate-500 text-lg">
                                {{ (member.username || member.temp_name || '?')[0].toUpperCase() }}
                            </div>
                            <div class="flex-1">
                                <p class="font-bold text-slate-800 text-lg">{{ member.username || member.temp_name
                                    }}</p>
                                <div class="flex items-center gap-2 text-sm">
                                    <span class="uppercase font-bold text-xs"
                                        :class="member.role === 'admin' ? 'text-primary-600' : 'text-slate-500'">
                                        {{ member.role }}
                                    </span>
                                    <span v-if="member.temp_name"
                                        class="bg-yellow-100 text-yellow-700 text-xs px-2 py-0.5 rounded-full font-bold">
                                        {{ t('groups.nonMember') }}
                                    </span>
                                    <span v-if="member.user_id === authStore.user?.id" class="text-slate-400 italic">({{
                                        t('common.you') }})</span>
                                </div>
                            </div>

                            <!-- Actions -->
                            <div class="flex items-center gap-2">
                                <button v-if="member.temp_name" @click="bindAccount(member.member_id)"
                                    class="p-2 text-primary-500 hover:bg-primary-50 rounded-lg transition-colors"
                                    :title="t('groups.bindAccount')">
                                    <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" fill="none"
                                        viewBox="0 0 24 24" stroke="currentColor">
                                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                                            d="M13.828 10.172a4 4 0 00-5.656 0l-4 4a4 4 0 105.656 5.656l1.102-1.101" />
                                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                                            d="M10.172 13.828a4 4 0 005.656 0l4-4a4 4 0 00-5.656-5.656l-1.102 1.101" />
                                    </svg>
                                </button>

                                <div v-if="canManageMembers && member.user_id !== authStore.user?.id"
                                    class="flex gap-1">
                                    <!-- Role Toggle -->
                                    <button v-if="member.role === 'member'"
                                        @click="updateMemberRole(member.member_id, 'admin')"
                                        class="text-xs font-bold text-blue-600 hover:bg-blue-50 px-2 py-1 rounded border border-blue-100"
                                        :title="t('groups.promoteToAdmin')">
                                        Admin
                                    </button>
                                    <button v-else-if="member.role === 'admin'"
                                        @click="updateMemberRole(member.member_id, 'member')"
                                        class="text-xs font-bold text-slate-500 hover:bg-slate-100 px-2 py-1 rounded border border-slate-200"
                                        :title="t('groups.demoteToMember')">
                                        Member
                                    </button>

                                    <!-- Kick -->
                                    <button @click="removeMember(member.member_id)"
                                        class="text-xs font-bold text-red-600 hover:bg-red-50 px-2 py-1 rounded border border-red-100 ml-1"
                                        :title="t('groups.kickMember')">
                                        {{ t('common.actions.remove') }}
                                    </button>
                                </div>
                            </div>
                        </div>
                    </div>
                </div>

                <!-- TAB: SETTINGS -->
                <div v-show="currentTab === 'settings'" class="animate-fade-in space-y-8">
                    <!-- Basic Info Form -->
                    <section class="glass-panel p-6">
                        <div class="flex justify-between items-center mb-6">
                            <h3 class="text-lg font-bold text-slate-800">{{ t('groups.editGroup') }}</h3>
                            <button v-if="canEditGroup && !isEditing" @click="startEdit"
                                class="text-primary-600 font-bold text-sm hover:underline">
                                {{ t('common.actions.edit') }}
                            </button>
                            <div v-if="isEditing" class="flex gap-2">
                                <button @click="isEditing = false"
                                    class="text-slate-500 font-bold text-sm hover:underline mr-2">
                                    {{ t('common.actions.cancel') }}
                                </button>
                                <button @click="saveSettings"
                                    class="text-primary-600 font-bold text-sm hover:underline">
                                    {{ t('common.actions.save') }}
                                </button>
                            </div>
                        </div>

                        <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
                            <div>
                                <label class="block text-xs font-bold text-slate-500 uppercase mb-1 ml-1">{{
                                    t('common.fields.name') }}</label>
                                <input v-if="isEditing" v-model="editForm.name" class="glass-input w-full" />
                                <input v-else v-model="group.name" class="glass-input w-full" disabled />
                            </div>
                            <div>
                                <label class="block text-xs font-bold text-slate-500 uppercase mb-1 ml-1">{{
                                    t('groups.serviceName') }}</label>
                                <input v-if="isEditing" v-model="editForm.service_name" class="glass-input w-full" />
                                <input v-else v-model="group.service_name" class="glass-input w-full" disabled />
                            </div>
                            <div>
                                <label class="block text-xs font-bold text-slate-500 uppercase mb-1 ml-1">{{
                                    t('groups.planName') }}</label>
                                <input v-if="isEditing" v-model="editForm.plan_name" class="glass-input w-full" />
                                <input v-else v-model="group.plan_name" class="glass-input w-full" disabled />
                            </div>
                            <div>
                                <label class="block text-xs font-bold text-slate-500 uppercase mb-1 ml-1">{{
                                    t('groups.amount') }}</label>
                                <div class="relative">
                                    <input v-if="isEditing" v-model.number="editForm.amount" type="number"
                                        class="glass-input w-full pl-6" />
                                    <input v-else v-model="group.amount" class="glass-input w-full pl-6" disabled />
                                    <span class="absolute left-3 top-1/2 -translate-y-1/2 text-slate-500">$</span>
                                </div>
                            </div>
                        </div>
                    </section>

                    <!-- Danger Zone -->
                    <section class="glass-panel p-6 border-l-4 border-red-500">
                        <h3 class="text-lg font-bold text-red-600 mb-4">{{ t('groups.dangerZone') }}</h3>
                        <div class="flex flex-col gap-4">
                            <div v-if="group.created_by === authStore.user?.id">
                                <p class="text-sm text-slate-600 mb-2">{{ t('groups.confirmDelete') }}</p>
                                <button @click="deleteGroup"
                                    class="btn bg-red-50 text-red-600 hover:bg-red-100 border border-red-200">
                                    {{ t('groups.deleteGroup') }}
                                </button>
                            </div>
                            <div v-else>
                                <p class="text-sm text-slate-600 mb-2">{{ t('groups.confirmLeave') }}</p>
                                <button @click="leaveGroup"
                                    class="btn bg-red-50 text-red-600 hover:bg-red-100 border border-red-200">
                                    {{ t('groups.leaveGroup') }}
                                </button>
                            </div>
                        </div>
                    </section>
                </div>
            </div>
        </div>

        <!-- Modals -->
        <AddExpenseModal v-if="showAddExpense" :group-id="groupId" :members="members" @close="showAddExpense = false"
            @added="handleExpenseAdded" />

    </MainLayout>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue';
import { useI18n } from 'vue-i18n';
import http from '../../http';
import { useUIStore } from '../../stores/ui';

const { t } = useI18n();
const ui = useUIStore();

interface User {
    id: string;
    username: string;
    email: string;
    created_at: string;
    is_suspended: boolean;
    suspended_until: string | null;
    is_blocked: boolean;
    two_factor_enabled: boolean;
}

interface Session {
    id: string;
    ip_address: string;
    user_agent: string;
    device_fingerprint: string;
    created_at: string;
    expires_at: string;
}

const users = ref<User[]>([]);
const loading = ref(true);

const viewSessionsModal = ref(false);
const activeUserForSessions = ref<User | null>(null);
const userSessions = ref<Session[]>([]);
const sessionsLoading = ref(false);

const fetchUsers = async () => {
    loading.value = true;
    try {
        const res = await http.get('/api/admin/users');
        if (res.data.status === 'ok') {
            users.value = res.data.users;
        }
    } catch (e) {
        console.error(e);
        ui.alert(t('admin.fetchUsersFailed'));
    } finally {
        loading.value = false;
    }
};

const handleStatusChange = async (user: User, action: 'ban' | 'unban' | 'suspend' | 'unsuspend') => {
    const confirmKey = `admin.confirm${action.charAt(0).toUpperCase() + action.slice(1)}` as any;
    if (!await ui.confirm(t(confirmKey, { username: user.username }))) return;

    try {
        let payload = { action, reason: '' };
        if (action === 'ban' || action === 'suspend') {
            const reason = prompt(t('admin.reasonPrompt', { action }));
            if (reason === null) return;
            payload.reason = reason;
        }

        const res = await http.put(`/api/admin/users/${user.id}/status`, payload);
        if (res.data.status === 'ok') {
            ui.alert(t('admin.actionSuccess', { action }));
            fetchUsers();
        }
    } catch (e) {
        ui.alert(t('admin.actionSuccess', { action }));
    }
};

const viewSessions = async (user: User) => {
    activeUserForSessions.value = user;
    viewSessionsModal.value = true;
    sessionsLoading.value = true;
    try {
        const res = await http.get(`/api/admin/users/${user.id}/sessions`);
        if (res.data.status === 'ok') {
            userSessions.value = res.data.sessions;
        }
    } catch (e) {
        ui.alert(t('admin.fetchSessionsFailed'));
    } finally {
        sessionsLoading.value = false;
    }
};

const revokeSession = async (sessionId: string) => {
    if (!activeUserForSessions.value) return;
    if (!await ui.confirm(t('security.confirmRevokeSession'))) return;

    try {
        const res = await http.delete(`/api/admin/users/${activeUserForSessions.value.id}/sessions/${sessionId}`);
        if (res.data.status === 'ok') {
            ui.alert(t('admin.sessionRevoked'));
            viewSessions(activeUserForSessions.value);
        }
    } catch (e) {
        ui.alert(t('admin.revokeSessionFailed'));
    }
};

onMounted(() => {
    fetchUsers();
});

const formatDate = (dateStr: string) => {
    return new Date(dateStr).toLocaleString();
};
</script>

<template>
    <div class="space-y-6">
        <div>
            <h2 class="text-2xl font-bold bg-gradient-to-r from-red-400 to-rose-600 bg-clip-text text-transparent">
                {{ t('admin.userManagement') }}</h2>
            <p class="text-neutral-400 mt-1">{{ t('admin.subtitle') }}</p>
        </div>

        <div v-if="loading" class="text-center py-10">
            <div class="w-8 h-8 border-4 border-red-500/20 border-t-red-500 rounded-full animate-spin mx-auto mb-4">
            </div>
            <p class="text-neutral-400">{{ t('admin.loadingUsers') }}</p>
        </div>

        <div v-else class="bg-neutral-950/50 rounded-2xl border border-neutral-800/50 overflow-hidden backdrop-blur-sm">
            <div class="overflow-x-auto">
                <table class="w-full text-left border-collapse">
                    <thead>
                        <tr
                            class="bg-neutral-900/80 border-b border-neutral-800/80 text-sm font-semibold text-neutral-300">
                            <th class="p-4">{{ t('common.fields.username') }}</th>
                            <th class="p-4">{{ t('common.fields.email') }}</th>
                            <th class="p-4">{{ t('admin.joined') }}</th>
                            <th class="p-4">{{ t('admin.status') }}</th>
                            <th class="p-4">{{ t('admin.twoFactor') }}</th>
                            <th class="p-4 text-right">{{ t('admin.actions') }}</th>
                        </tr>
                    </thead>
                    <tbody class="divide-y divide-neutral-800/50">
                        <tr v-for="user in users" :key="user.id" class="hover:bg-neutral-800/20 transition-colors">
                            <td class="p-4 font-medium text-white">{{ user.username }}</td>
                            <td class="p-4 text-neutral-400">{{ user.email }}</td>
                            <td class="p-4 text-sm text-neutral-500">{{ new Date(user.created_at).toLocaleDateString()
                                }}</td>
                            <td class="p-4">
                                <span v-if="user.is_blocked"
                                    class="px-2.5 py-1 text-xs font-semibold rounded-full bg-red-500/10 text-red-500 border border-red-500/20">{{
                                    t('admin.banned') }}</span>
                                <span v-else-if="user.is_suspended"
                                    class="px-2.5 py-1 text-xs font-semibold rounded-full bg-orange-500/10 text-orange-500 border border-orange-500/20">{{
                                    t('admin.suspended') }}</span>
                                <span v-else
                                    class="px-2.5 py-1 text-xs font-semibold rounded-full bg-emerald-500/10 text-emerald-500 border border-emerald-500/20">{{
                                    t('admin.active') }}</span>
                            </td>
                            <td class="p-4">
                                <span v-if="user.two_factor_enabled" class="text-emerald-500 text-sm">{{
                                    t('admin.enabled') }}</span>
                                <span v-else class="text-neutral-500 text-sm">{{ t('admin.disabled') }}</span>
                            </td>
                            <td class="p-4 text-right space-x-2">
                                <button @click="viewSessions(user)"
                                    class="px-3 py-1.5 text-sm font-medium rounded-lg bg-neutral-800 text-neutral-300 hover:bg-neutral-700 transition">
                                    {{ t('admin.sessions') }}
                                </button>
                                <button v-if="!user.is_blocked" @click="handleStatusChange(user, 'ban')"
                                    class="px-3 py-1.5 text-sm font-medium rounded-lg bg-red-500/10 text-red-400 hover:bg-red-500/20 transition">
                                    {{ t('admin.ban') }}
                                </button>
                                <button v-else @click="handleStatusChange(user, 'unban')"
                                    class="px-3 py-1.5 text-sm font-medium rounded-lg bg-emerald-500/10 text-emerald-400 hover:bg-emerald-500/20 transition">
                                    {{ t('admin.unban') }}
                                </button>
                            </td>
                        </tr>
                    </tbody>
                </table>
            </div>
        </div>

        <!-- Sessions Modal -->
        <div v-if="viewSessionsModal"
            class="fixed inset-0 bg-black/60 backdrop-blur-sm z-50 flex items-center justify-center p-4">
            <div
                class="bg-neutral-900 border border-neutral-800 rounded-2xl shadow-xl w-full max-w-3xl max-h-[90vh] flex flex-col">
                <div class="p-6 border-b border-neutral-800 flex justify-between items-center">
                    <div>
                        <h3 class="text-xl font-semibold text-white">{{ t('admin.activeSessions') }}</h3>
                        <p class="text-sm text-neutral-400 mt-1">{{ t('admin.userLabel') }}: <span class="text-white">{{
                            activeUserForSessions?.username }}</span></p>
                    </div>
                    <button @click="viewSessionsModal = false" class="text-neutral-400 hover:text-white transition">
                        <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                                d="M6 18L18 6M6 6l12 12"></path>
                        </svg>
                    </button>
                </div>

                <div class="p-6 overflow-y-auto flex-1">
                    <div v-if="sessionsLoading" class="text-center py-8">
                        <div
                            class="w-6 h-6 border-2 border-red-500/20 border-t-red-500 rounded-full animate-spin mx-auto mb-2">
                        </div>
                    </div>
                    <div v-else-if="userSessions.length === 0" class="text-center py-8 text-neutral-400">
                        {{ t('admin.noSessions') }}
                    </div>
                    <div v-else class="space-y-4">
                        <div v-for="session in userSessions" :key="session.id"
                            class="p-4 rounded-xl bg-neutral-950/50 border border-neutral-800 flex items-start justify-between">
                            <div>
                                <div class="flex items-center space-x-2 mb-1">
                                    <span class="font-medium text-white">{{ session.ip_address }}</span>
                                    <span v-if="new Date(session.expires_at) < new Date()"
                                        class="px-2 py-0.5 text-xs rounded bg-red-500/10 text-red-500">{{
                                        t('admin.expired') }}</span>
                                    <span v-else
                                        class="px-2 py-0.5 text-xs rounded bg-emerald-500/10 text-emerald-500">{{
                                        t('admin.active') }}</span>
                                </div>
                                <p class="text-sm text-neutral-400">{{ session.user_agent }}</p>
                                <p class="text-xs text-neutral-500 mt-2">{{ t('admin.started') }}: {{
                                    formatDate(session.created_at) }}
                                </p>
                            </div>
                            <button @click="revokeSession(session.id)"
                                class="px-3 py-1.5 text-sm font-medium rounded-lg border border-red-500/20 text-red-400 hover:bg-red-500/10 hover:border-red-500/40 transition shrink-0 ml-4">
                                {{ t('admin.revoke') }}
                            </button>
                        </div>
                    </div>
                </div>
            </div>
        </div>
    </div>
</template>

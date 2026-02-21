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

// Sessions modal
const viewSessionsModal = ref(false);
const activeUserForSessions = ref<User | null>(null);
const userSessions = ref<Session[]>([]);
const sessionsLoading = ref(false);

// Suspend modal
const suspendModal = ref(false);
const suspendTarget = ref<User | null>(null);
const suspendDuration = ref('24h');
const suspendReason = ref('');

// Ban modal
const banModal = ref(false);
const banTarget = ref<User | null>(null);
const banReason = ref('');

// Change password modal
const changePasswordModal = ref(false);
const passwordTarget = ref<User | null>(null);
const newPassword = ref('');
const showPassword = ref(false);

// Custom suspension duration (date string)
const customSuspendDate = ref('');

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

// --- Block / Unblock ---
const openBanModal = (user: User) => {
    banTarget.value = user;
    banReason.value = '';
    banModal.value = true;
};

const submitBan = async () => {
    if (!banTarget.value) return;
    try {
        await http.put(`/api/admin/users/${banTarget.value.id}/status`, {
            action: 'ban',
            reason: banReason.value
        });
        ui.alert(t('admin.actionSuccess', { action: t('admin.ban') }));
        banModal.value = false;
        fetchUsers();
    } catch (e) {
        ui.alert(t('admin.fetchUsersFailed'));
    }
};

const handleUnblock = async (user: User) => {
    if (!await ui.confirm(t('admin.confirmUnban', { username: user.username }))) return;
    try {
        await http.put(`/api/admin/users/${user.id}/status`, { action: 'unban' });
        ui.alert(t('admin.actionSuccess', { action: t('admin.unban') }));
        fetchUsers();
    } catch (e) {
        ui.alert(t('admin.fetchUsersFailed'));
    }
};

// --- Suspend with duration ---
const openSuspendModal = (user: User) => {
    suspendTarget.value = user;
    suspendDuration.value = '24h';
    suspendReason.value = '';
    suspendModal.value = true;
};

const submitSuspend = async () => {
    if (!suspendTarget.value) return;
    const durationMap: Record<string, number> = {
        '1h': 1 * 60 * 60 * 1000,
        '24h': 24 * 60 * 60 * 1000,
        '7d': 7 * 24 * 60 * 60 * 1000,
        '30d': 30 * 24 * 60 * 60 * 1000,
        'permanent': 0,
        'custom': -1
    };
    const ms = durationMap[suspendDuration.value] ?? 0;
    let until = null;

    if (ms === -1) {
        if (!customSuspendDate.value) {
            ui.alert(t('admin.customDurationRequired', 'Please select a date for custom duration'));
            return;
        }
        until = new Date(customSuspendDate.value).toISOString();
    } else if (ms > 0) {
        until = new Date(Date.now() + ms).toISOString();
    } else if (ms === 0) {
        until = null; // Permanent
    }
    try {
        await http.put(`/api/admin/users/${suspendTarget.value.id}/status`, {
            action: 'suspend',
            reason: suspendReason.value,
            until
        });
        ui.alert(t('admin.actionSuccess', { action: t('admin.suspend') }));
        suspendModal.value = false;
        fetchUsers();
    } catch (e) {
        ui.alert(t('admin.fetchUsersFailed'));
    }
};

const handleUnsuspend = async (user: User) => {
    if (!await ui.confirm(t('admin.confirmUnsuspend', { username: user.username }))) return;
    try {
        await http.put(`/api/admin/users/${user.id}/status`, { action: 'unsuspend' });
        ui.alert(t('admin.actionSuccess', { action: t('admin.unsuspend') }));
        fetchUsers();
    } catch (e) {
        ui.alert(t('admin.fetchUsersFailed'));
    }
};

// --- Change Password ---
const openChangePassword = (user: User) => {
    passwordTarget.value = user;
    newPassword.value = '';
    showPassword.value = false;
    changePasswordModal.value = true;
};

const submitChangePassword = async () => {
    if (!passwordTarget.value || !newPassword.value) return;
    try {
        await http.put(`/api/admin/users/${passwordTarget.value.id}/password`, {
            newPassword: newPassword.value
        });
        ui.alert(t('admin.passwordChanged'));
        changePasswordModal.value = false;
    } catch (e) {
        ui.alert(t('admin.passwordChangeFailed'));
    }
};

// --- Sessions ---
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

onMounted(() => { fetchUsers(); });

const formatDate = (d: string) => new Date(d).toLocaleString();

const formatUserAgent = (uaString: string) => {
    if (!uaString) return 'Unknown Device';

    let browser = 'Unknown Browser';
    if (uaString.indexOf("Firefox") > -1) browser = "Firefox";
    else if (uaString.indexOf("SamsungBrowser") > -1) browser = "Samsung Internet";
    else if (uaString.indexOf("Opera") > -1 || uaString.indexOf("OPR") > -1) browser = "Opera";
    else if (uaString.indexOf("Trident") > -1) browser = "Internet Explorer";
    else if (uaString.indexOf("Edge") > -1) browser = "Edge";
    else if (uaString.indexOf("Chrome") > -1) browser = "Chrome";
    else if (uaString.indexOf("Safari") > -1) browser = "Safari";

    let os = 'Unknown OS';
    if (uaString.indexOf("Win") > -1) os = "Windows";
    else if (uaString.indexOf("Mac") > -1) os = "MacOS";
    else if (uaString.indexOf("Linux") > -1) os = "Linux";
    else if (uaString.indexOf("Android") > -1) os = "Android";
    else if (uaString.indexOf("like Mac") > -1) os = "iOS";

    return `${browser} on ${os}`;
};

const getRemainingTime = (until: string | null) => {
    if (!until) return t('admin.suspendOptions.permanent');
    const diff = new Date(until).getTime() - Date.now();
    if (diff <= 0) return t('admin.expired');

    const minutes = Math.floor(diff / (1000 * 60));
    const hours = Math.floor(minutes / 60);
    const days = Math.floor(hours / 24);

    let timeStr = '';
    if (days > 0) timeStr = `${days}d ${hours % 24}h`;
    else if (hours > 0) timeStr = `${hours}h ${minutes % 60}m`;
    else timeStr = `${minutes}m`;

    return t('admin.endsIn', { time: timeStr });
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
                                <div v-if="user.is_blocked" class="flex flex-col">
                                    <span
                                        class="px-2.5 py-1 text-xs font-semibold rounded-full bg-red-500/10 text-red-500 border border-red-500/20 w-fit">
                                        {{ t('admin.banned') }}
                                    </span>
                                </div>
                                <div v-else-if="user.is_suspended" class="flex flex-col gap-1">
                                    <span
                                        class="px-2.5 py-1 text-xs font-semibold rounded-full bg-orange-500/10 text-orange-500 border border-orange-500/20 w-fit">
                                        {{ t('admin.suspended') }}
                                    </span>
                                    <div class="text-xs text-neutral-400 leading-tight">
                                        <div>{{ getRemainingTime(user.suspended_until) }}</div>
                                        <div v-if="user.suspended_until" class="text-neutral-600">
                                            {{ formatDate(user.suspended_until) }}
                                        </div>
                                    </div>
                                </div>
                                <span v-else
                                    class="px-2.5 py-1 text-xs font-semibold rounded-full bg-emerald-500/10 text-emerald-500 border border-emerald-500/20">
                                    {{ t('admin.active') }}
                                </span>
                            </td>
                            <td class="p-4">
                                <span v-if="user.two_factor_enabled" class="text-emerald-500 text-sm">{{
                                    t('admin.enabled') }}</span>
                                <span v-else class="text-neutral-500 text-sm">{{ t('admin.disabled') }}</span>
                            </td>
                            <td class="p-4 text-right">
                                <div class="flex flex-wrap justify-end gap-2">
                                    <button @click="viewSessions(user)"
                                        class="px-3 py-1.5 text-sm font-medium rounded-lg bg-neutral-800 text-neutral-300 hover:bg-neutral-700 transition">
                                        {{ t('admin.sessions') }}
                                    </button>
                                    <button @click="openChangePassword(user)"
                                        class="px-3 py-1.5 text-sm font-medium rounded-lg bg-blue-500/10 text-blue-400 hover:bg-blue-500/20 transition">
                                        {{ t('admin.changePassword') }}
                                    </button>
                                    <!-- Block / Unblock -->
                                    <button v-if="!user.is_blocked" @click="openBanModal(user)"
                                        class="px-3 py-1.5 text-sm font-medium rounded-lg bg-red-500/10 text-red-400 hover:bg-red-500/20 transition">
                                        {{ t('admin.ban') }}
                                    </button>
                                    <button v-else @click="handleUnblock(user)"
                                        class="px-3 py-1.5 text-sm font-medium rounded-lg bg-emerald-500/10 text-emerald-400 hover:bg-emerald-500/20 transition">
                                        {{ t('admin.unban') }}
                                    </button>
                                    <!-- Suspend / Unsuspend -->
                                    <button v-if="!user.is_suspended && !user.is_blocked"
                                        @click="openSuspendModal(user)"
                                        class="px-3 py-1.5 text-sm font-medium rounded-lg bg-orange-500/10 text-orange-400 hover:bg-orange-500/20 transition">
                                        {{ t('admin.suspend') }}
                                    </button>
                                    <button v-else-if="user.is_suspended" @click="handleUnsuspend(user)"
                                        class="px-3 py-1.5 text-sm font-medium rounded-lg bg-cyan-500/10 text-cyan-400 hover:bg-cyan-500/20 transition">
                                        {{ t('admin.unsuspend') }}
                                    </button>
                                </div>
                            </td>
                        </tr>
                    </tbody>
                </table>
            </div>
        </div>

        <!-- Suspend Modal -->
        <div v-if="suspendModal"
            class="fixed inset-0 bg-black/60 backdrop-blur-sm z-50 flex items-center justify-center p-4">
            <div class="bg-neutral-900 border border-neutral-800 rounded-2xl shadow-xl w-full max-w-md flex flex-col">
                <div class="p-6 border-b border-neutral-800 flex justify-between items-center">
                    <h3 class="text-xl font-semibold text-white">{{ t('admin.suspend') }} — {{ suspendTarget?.username
                        }}</h3>
                    <button @click="suspendModal = false" class="text-neutral-400 hover:text-white transition">
                        <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                                d="M6 18L18 6M6 6l12 12" />
                        </svg>
                    </button>
                </div>
                <div class="p-6 space-y-4 overflow-y-auto max-h-[70vh]">
                    <div>
                        <label class="block text-sm font-medium text-neutral-400 mb-2">{{ t('admin.suspendDuration')
                            }}</label>
                        <select v-model="suspendDuration"
                            class="w-full bg-neutral-950/50 border border-neutral-800 rounded-xl px-4 py-2.5 text-neutral-200 focus:ring-2 focus:ring-red-500/50 outline-none">
                            <option value="1h">{{ t('admin.suspendOptions.1h') }}</option>
                            <option value="24h">{{ t('admin.suspendOptions.24h') }}</option>
                            <option value="7d">{{ t('admin.suspendOptions.7d') }}</option>
                            <option value="30d">{{ t('admin.suspendOptions.30d') }}</option>
                            <option value="permanent">{{ t('admin.suspendOptions.permanent') }}</option>
                            <option value="custom">{{ t('admin.suspendOptions.custom') }}</option>
                        </select>
                    </div>
                    <div v-if="suspendDuration === 'custom'">
                        <label class="block text-sm font-medium text-neutral-400 mb-2">{{ t('admin.customDate')
                            }}</label>
                        <input type="datetime-local" v-model="customSuspendDate"
                            class="w-full bg-neutral-950/50 border border-neutral-800 rounded-xl px-4 py-2.5 text-neutral-200 focus:ring-2 focus:ring-red-500/50 outline-none" />
                    </div>
                    <div>
                        <label class="block text-sm font-medium text-neutral-400 mb-2">{{ t('admin.ip.reason')
                            }}</label>
                        <input type="text" v-model="suspendReason"
                            class="w-full bg-neutral-950/50 border border-neutral-800 rounded-xl px-4 py-2.5 text-neutral-200 focus:ring-2 focus:ring-red-500/50 outline-none" />
                    </div>
                </div>
                <div class="p-6 border-t border-neutral-800 flex justify-end gap-3">
                    <button @click="suspendModal = false"
                        class="px-4 py-2 text-neutral-400 hover:text-white transition">{{ t('common.actions.cancel')
                        }}</button>
                    <button @click="submitSuspend"
                        class="px-4 py-2 bg-orange-500/10 text-orange-400 border border-orange-500/20 rounded-xl hover:bg-orange-500/20 font-medium transition">
                        {{ t('admin.suspend') }}
                    </button>
                </div>
            </div>
        </div>

        <!-- Ban Modal -->
        <div v-if="banModal"
            class="fixed inset-0 bg-black/60 backdrop-blur-sm z-50 flex items-center justify-center p-4">
            <div class="bg-neutral-900 border border-neutral-800 rounded-2xl shadow-xl w-full max-w-md">
                <div class="p-6 border-b border-neutral-800 flex justify-between items-center">
                    <h3 class="text-xl font-semibold text-white">{{ t('admin.ban') }} — {{ banTarget?.username }}</h3>
                    <button @click="banModal = false" class="text-neutral-400 hover:text-white transition">
                        <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                                d="M6 18L18 6M6 6l12 12" />
                        </svg>
                    </button>
                </div>
                <div class="p-6 space-y-4">
                    <div class="p-4 rounded-xl bg-red-500/5 border border-red-500/10 text-red-400 text-sm">
                        {{ t('admin.confirmBan', { username: banTarget?.username }) }}
                    </div>
                    <div>
                        <label class="block text-sm font-medium text-neutral-400 mb-2">{{ t('admin.ip.reason')
                            }}</label>
                        <input type="text" v-model="banReason"
                            class="w-full bg-neutral-950/50 border border-neutral-800 rounded-xl px-4 py-2.5 text-neutral-200 focus:ring-2 focus:ring-red-500/50 outline-none" />
                    </div>
                    <div class="flex justify-end gap-3">
                        <button @click="banModal = false"
                            class="px-4 py-2 text-neutral-400 hover:text-white transition">{{ t('common.actions.cancel')
                            }}</button>
                        <button @click="submitBan"
                            class="px-4 py-2 bg-red-500/10 text-red-400 border border-red-500/20 rounded-xl hover:bg-red-500/20 font-medium transition">
                            {{ t('admin.ban') }}
                        </button>
                    </div>
                </div>
            </div>
        </div>

        <!-- Change Password Modal -->
        <div v-if="changePasswordModal"
            class="fixed inset-0 bg-black/60 backdrop-blur-sm z-50 flex items-center justify-center p-4">
            <div class="bg-neutral-900 border border-neutral-800 rounded-2xl shadow-xl w-full max-w-md">
                <div class="p-6 border-b border-neutral-800 flex justify-between items-center">
                    <h3 class="text-xl font-semibold text-white">{{ t('admin.changePassword') }} — {{
                        passwordTarget?.username }}</h3>
                    <button @click="changePasswordModal = false" class="text-neutral-400 hover:text-white transition">
                        <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                                d="M6 18L18 6M6 6l12 12" />
                        </svg>
                    </button>
                </div>
                <div class="p-6 space-y-4">
                    <div>
                        <label class="block text-sm font-medium text-neutral-400 mb-2">{{ t('admin.newPassword')
                            }}</label>
                        <div class="relative">
                            <input :type="showPassword ? 'text' : 'password'" v-model="newPassword"
                                class="w-full bg-neutral-950/50 border border-neutral-800 rounded-xl px-4 py-2.5 pr-12 text-neutral-200 focus:ring-2 focus:ring-blue-500/50 outline-none" />
                            <button @click="showPassword = !showPassword"
                                class="absolute right-3 top-1/2 -translate-y-1/2 text-neutral-500 hover:text-white transition p-1">
                                <svg v-if="showPassword" class="w-5 h-5" fill="none" stroke="currentColor"
                                    viewBox="0 0 24 24">
                                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                                        d="M13.875 18.825A10.05 10.05 0 0112 19c-4.478 0-8.268-2.943-9.543-7a9.97 9.97 0 011.563-3.029m5.858.908a3 3 0 114.243 4.243M9.878 9.878l4.242 4.242M9.88 9.88l-3.29-3.29m7.532 7.532l3.29 3.29M3 3l18 18" />
                                </svg>
                                <svg v-else class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                                        d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
                                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                                        d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
                                </svg>
                            </button>
                        </div>
                    </div>
                    <div class="flex justify-end gap-3">
                        <button @click="changePasswordModal = false"
                            class="px-4 py-2 text-neutral-400 hover:text-white transition">{{ t('common.actions.cancel')
                            }}</button>
                        <button @click="submitChangePassword" :disabled="!newPassword"
                            class="px-4 py-2 bg-blue-500/10 text-blue-400 border border-blue-500/20 rounded-xl hover:bg-blue-500/20 font-medium transition disabled:opacity-50">
                            {{ t('admin.changePassword') }}
                        </button>
                    </div>
                </div>
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
                                d="M6 18L18 6M6 6l12 12" />
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
                                    <span class="font-bold text-white">{{ session.ip_address }}</span>
                                    <span v-if="new Date(session.expires_at) < new Date()"
                                        class="px-2 py-0.5 text-[10px] font-bold uppercase rounded bg-red-500/10 text-red-500">{{
                                            t('admin.expired') }}</span>
                                    <span v-else
                                        class="px-2 py-0.5 text-[10px] font-bold uppercase rounded bg-emerald-500/10 text-emerald-500">{{
                                            t('admin.active') }}</span>
                                </div>
                                <div class="text-sm text-neutral-200 font-semibold mb-1">{{
                                    formatUserAgent(session.user_agent) }}</div>
                                <p class="text-[11px] text-neutral-500 font-mono break-all leading-tight max-w-lg">{{
                                    session.user_agent }}</p>
                                <p class="text-[10px] text-neutral-500 mt-2 italic">{{ t('admin.started') }}: {{
                                    formatDate(session.created_at) }}</p>
                            </div>
                            <button @click="revokeSession(session.id)"
                                class="px-3 py-1.5 text-sm font-medium rounded-lg border border-red-500/20 text-red-400 hover:bg-red-500/10 transition shrink-0 ml-4">
                                {{ t('admin.revoke') }}
                            </button>
                        </div>
                    </div>
                </div>
            </div>
        </div>
    </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, watch } from 'vue';
import MainLayout from './MainLayout.vue';
import http from '../http';
import { useI18n } from 'vue-i18n';

const { t } = useI18n();
const logs = ref<any[]>([]);
const loading = ref(true);
const error = ref('');
const page = ref(1);
const limit = 20;
const total = ref(0);

const autoRefresh = ref(false);
const refreshInterval = ref(30000);
let refreshTimer: any = null;

const intervalOptions = [
    { label: '1s', value: 1000 },
    { label: '3s', value: 3000 },
    { label: '5s', value: 5000 },
    { label: '15s', value: 15000 },
    { label: '30s', value: 30000 },
    { label: '60s', value: 60000 },
    { label: '3m', value: 180000 },
    { label: '5m', value: 300000 },
    { label: '10m', value: 600000 },
    { label: '15m', value: 900000 }
];

const startTimer = () => {
    stopTimer();
    refreshTimer = setInterval(() => {
        if (!loading.value) fetchLogs();
    }, refreshInterval.value);
};

const stopTimer = () => {
    if (refreshTimer) {
        clearInterval(refreshTimer);
        refreshTimer = null;
    }
};

const fetchLogs = async () => {
    loading.value = true;
    try {
        const offset = (page.value - 1) * limit;
        const res = await http.get(`/api/audit/user/activity?limit=${limit}&offset=${offset}`);

        if (res.data && res.data.logs) {
            logs.value = res.data.logs;
            total.value = res.data.pagination?.total || 0;
        } else {
            console.warn('Unexpected response format:', res.data);
            logs.value = [];
            total.value = 0;
        }
    } catch (err: any) {
        console.error(err);
        const msg = err.response?.data?.message || err.message || 'auth.errors.unknownError';
        error.value = t(msg, msg);
    } finally {
        loading.value = false;
    }
};

const nextPage = () => {
    if ((page.value * limit) < total.value) {
        page.value++;
        fetchLogs();
    }
};

const prevPage = () => {
    if (page.value > 1) {
        page.value--;
        fetchLogs();
    }
};

watch(autoRefresh, (val) => {
    if (val) startTimer();
    else stopTimer();
});

watch(refreshInterval, () => {
    if (autoRefresh.value) startTimer();
});

onMounted(() => {
    fetchLogs();
});

onUnmounted(stopTimer);

const getRiskColor = (level: string) => {
    switch (level) {
        case 'critical': return 'text-red-600 bg-red-100';
        case 'high': return 'text-orange-600 bg-orange-100';
        case 'medium': return 'text-yellow-600 bg-yellow-100';
        default: return 'text-slate-600 bg-slate-100';
    }
};
</script>

<template>
    <MainLayout>
        <div class="activity-log-view o">
            <header class="mb-8 flex flex-col md:flex-row md:items-end justify-between gap-4">
                <div>
                    <h1 class="text-3xl font-extrabold text-slate-800">{{ t('activity.title', 'Activity Log') }}</h1>
                    <p class="text-slate-500">
                        {{ t('activity.subtitle', 'View your recent account activity and security events.') }}
                    </p>
                </div>
                <div class="flex items-center gap-4">
                    <div class="flex items-center gap-2 text-sm text-slate-500 font-medium">
                        <input type="checkbox" v-model="autoRefresh" id="auto-refresh"
                            class="rounded border-slate-300 text-primary-600 focus:ring-primary-500" />
                        <label for="auto-refresh" class="cursor-pointer">{{ t('common.actions.autoRefresh') }}</label>
                        <select v-if="autoRefresh" v-model="refreshInterval"
                            class="ml-2 bg-slate-50 border border-slate-200 rounded-lg text-xs py-1 px-2 focus:ring-1 focus:ring-primary-500 outline-none text-primary-700">
                            <option v-for="opt in intervalOptions" :key="opt.value" :value="opt.value">
                                {{ t(`common.intervals.${opt.label}`) }}
                            </option>
                        </select>
                    </div>
                    <button @click="fetchLogs" :disabled="loading"
                        class="btn btn-secondary flex items-center gap-2 py-2">
                        <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" :class="{ 'animate-spin': loading }"
                            fill="none" viewBox="0 0 24 24" stroke="currentColor">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                                d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
                        </svg>
                        {{ t('common.actions.refresh') }}
                    </button>
                </div>
            </header>

            <div class="glass-panel p-6 min-h-[400px]">
                <div v-if="loading" class="flex justify-center p-12">
                    <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-primary-600"></div>
                </div>

                <div v-else-if="error" class="text-center p-12 text-red-500">
                    {{ error }}
                </div>

                <div v-else>
                    <div class="overflow-x-auto">
                        <table class="w-full text-left text-sm">
                            <thead class="bg-slate-50 text-slate-500 font-bold uppercase border-b border-slate-200">
                                <tr>
                                    <th class="p-3">{{ t('common.fields.date', 'Date') }}</th>
                                    <th class="p-3">{{ t('activity.action', 'Action') }}</th>
                                    <th class="p-3">{{ t('activity.description', 'Description') }}</th>
                                    <th class="p-3">{{ t('activity.ip', 'IP Address') }}</th>
                                </tr>
                            </thead>
                            <tbody class="divide-y divide-slate-100">
                                <tr v-for="log in logs" :key="log.id" class="hover:bg-slate-50 transition-colors">
                                    <td class="p-3 text-slate-500 whitespace-nowrap">
                                        {{ new Date(log.created_at).toLocaleString() }}
                                    </td>
                                    <td class="p-3">
                                        <span class="px-2 py-0.5 rounded-full text-xs font-bold uppercase"
                                            :class="getRiskColor(log.risk_level)">
                                            {{ log.action }}
                                        </span>
                                    </td>
                                    <td class="p-3 text-slate-800 font-medium">
                                        {{ log.description || log.behavior_type }}
                                    </td>
                                    <td class="p-3 text-slate-400 font-mono text-xs">
                                        {{ log.ip_address || '---' }}
                                    </td>
                                </tr>
                                <tr v-if="logs.length === 0">
                                    <td colspan="4" class="p-8 text-center text-slate-400">
                                        {{ t('activity.noLogs', 'No activity logs found.') }}
                                    </td>
                                </tr>
                            </tbody>
                        </table>
                    </div>

                    <!-- Pagination -->
                    <div class="flex justify-between items-center mt-6 pt-4 border-t border-slate-100">
                        <button @click="prevPage" :disabled="page === 1"
                            class="px-4 py-2 rounded-lg text-sm font-bold transition-colors"
                            :class="page === 1 ? 'text-slate-300 cursor-not-allowed' : 'text-slate-600 hover:bg-slate-100'">
                            &larr; {{ t('common.actions.prev', 'Previous') }}
                        </button>
                        <span class="text-sm text-slate-500">
                            Page {{ page }} of {{ Math.ceil(total / limit) || 1 }}
                        </span>
                        <button @click="nextPage" :disabled="(page * limit) >= total"
                            class="px-4 py-2 rounded-lg text-sm font-bold transition-colors"
                            :class="(page * limit) >= total ? 'text-slate-300 cursor-not-allowed' : 'text-slate-600 hover:bg-slate-100'">
                            {{ t('common.actions.next', 'Next') }} &rarr;
                        </button>
                    </div>
                </div>
            </div>
        </div>
    </MainLayout>
</template>

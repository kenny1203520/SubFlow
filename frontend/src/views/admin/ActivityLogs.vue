<script setup lang="ts">
import { ref, onMounted, computed } from 'vue';
import { useI18n } from 'vue-i18n';
import http from '../../http';
import { useUIStore } from '../../stores/ui';

const { t } = useI18n();
const ui = useUIStore();

interface LogEntry {
    id: string;
    user_id: string | null;
    username: string | null;
    action: string;
    behavior_type: string;
    risk_level: string;
    description: string;
    details: any;
    ip_address: string;
    user_agent: string;
    device_fingerprint: string;
    geo_location: string | null;
    created_at: string;
}

const logs = ref<LogEntry[]>([]);
const loading = ref(true);
const page = ref(1);
const limit = 30;
const total = ref(0);

// Filters
const filterAction = ref('');
const filterRisk = ref('');
const filterBehavior = ref('');
const filterUser = ref('');
const filterStartDate = ref('');
const filterEndDate = ref('');

// Filter options (populated from backend)
const actionOptions = ref<string[]>([]);
const riskOptions = ref<string[]>([]);
const behaviorOptions = ref<string[]>([]);

// Detail modal
const detailModal = ref(false);
const selectedLog = ref<LogEntry | null>(null);

const fetchLogs = async () => {
    loading.value = true;
    try {
        const offset = (page.value - 1) * limit;
        const params = new URLSearchParams();
        params.set('limit', String(limit));
        params.set('offset', String(offset));
        if (filterAction.value) params.set('action', filterAction.value);
        if (filterRisk.value) params.set('risk_level', filterRisk.value);
        if (filterBehavior.value) params.set('behavior_type', filterBehavior.value);
        if (filterUser.value) params.set('user_id', filterUser.value);
        if (filterStartDate.value) params.set('startDate', new Date(filterStartDate.value).toISOString());
        if (filterEndDate.value) params.set('endDate', new Date(filterEndDate.value).toISOString());

        const res = await http.get(`/api/admin/logs?${params.toString()}`);
        if (res.data.status === 'ok') {
            logs.value = res.data.logs;
            total.value = res.data.pagination.total;
            if (res.data.filters) {
                actionOptions.value = res.data.filters.actions || [];
                riskOptions.value = res.data.filters.riskLevels || [];
                behaviorOptions.value = res.data.filters.behaviorTypes || [];
            }
        }
    } catch (e) {
        ui.alert(t('admin.logs.fetchFailed'));
    } finally {
        loading.value = false;
    }
};

const totalPages = computed(() => Math.ceil(total.value / limit) || 1);

const nextPage = () => { if (page.value < totalPages.value) { page.value++; fetchLogs(); } };
const prevPage = () => { if (page.value > 1) { page.value--; fetchLogs(); } };

const applyFilters = () => {
    page.value = 1;
    fetchLogs();
};

const clearFilters = () => {
    filterAction.value = '';
    filterRisk.value = '';
    filterBehavior.value = '';
    filterUser.value = '';
    filterStartDate.value = '';
    filterEndDate.value = '';
    page.value = 1;
    fetchLogs();
};

const hasActiveFilters = computed(() =>
    filterAction.value || filterRisk.value || filterBehavior.value ||
    filterUser.value || filterStartDate.value || filterEndDate.value
);

const openDetail = (log: LogEntry) => {
    selectedLog.value = log;
    detailModal.value = true;
};

const getRiskClasses = (level: string) => {
    switch (level) {
        case 'critical': return 'bg-red-500/10 text-red-400 border-red-500/20';
        case 'high': return 'bg-orange-500/10 text-orange-400 border-orange-500/20';
        case 'medium': return 'bg-yellow-500/10 text-yellow-400 border-yellow-500/20';
        case 'low': return 'bg-blue-500/10 text-blue-400 border-blue-500/20';
        default: return 'bg-neutral-800 text-neutral-400 border-neutral-700';
    }
};

const formatDate = (d: string) => new Date(d).toLocaleString();

const formatUserAgent = (uaString: string) => {
    if (!uaString) return 'Unknown';
    let browser = 'Unknown';
    if (uaString.indexOf("Firefox") > -1) browser = "Firefox";
    else if (uaString.indexOf("OPR") > -1 || uaString.indexOf("Opera") > -1) browser = "Opera";
    else if (uaString.indexOf("Edge") > -1) browser = "Edge";
    else if (uaString.indexOf("Chrome") > -1) browser = "Chrome";
    else if (uaString.indexOf("Safari") > -1) browser = "Safari";
    let os = 'Unknown';
    if (uaString.indexOf("Win") > -1) os = "Windows";
    else if (uaString.indexOf("Mac") > -1) os = "macOS";
    else if (uaString.indexOf("Linux") > -1) os = "Linux";
    else if (uaString.indexOf("Android") > -1) os = "Android";
    else if (uaString.indexOf("like Mac") > -1) os = "iOS";
    return `${browser} · ${os}`;
};

const formatDetails = (details: any) => {
    if (!details) return '';
    try {
        return JSON.stringify(details, null, 2);
    } catch { return String(details); }
};

onMounted(fetchLogs);
</script>

<template>
    <div class="space-y-6">
        <div>
            <h2 class="text-2xl font-bold bg-gradient-to-r from-red-400 to-rose-600 bg-clip-text text-transparent">
                {{ t('admin.logs.title') }}</h2>
            <p class="text-neutral-400 mt-1">{{ t('admin.logs.subtitle') }}</p>
        </div>

        <!-- Filters -->
        <section class="bg-neutral-900/50 border border-neutral-800 rounded-2xl p-5">
            <div class="flex items-center justify-between mb-4">
                <h3 class="text-sm font-semibold text-neutral-300 uppercase tracking-wider">{{
                    t('admin.logs.filters') }}</h3>
                <button v-if="hasActiveFilters" @click="clearFilters"
                    class="text-xs text-red-400 hover:text-red-300 transition">
                    {{ t('admin.logs.clearFilters') }}
                </button>
            </div>
            <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-6 gap-3">
                <!-- Risk Level -->
                <select v-model="filterRisk" @change="applyFilters"
                    class="bg-neutral-950/50 border border-neutral-800 rounded-xl px-3 py-2 text-sm text-neutral-200 focus:ring-2 focus:ring-red-500/50 outline-none">
                    <option value="">{{ t('admin.logs.allRiskLevels') }}</option>
                    <option v-for="r in riskOptions" :key="r" :value="r">{{ r }}</option>
                </select>
                <!-- Action -->
                <select v-model="filterAction" @change="applyFilters"
                    class="bg-neutral-950/50 border border-neutral-800 rounded-xl px-3 py-2 text-sm text-neutral-200 focus:ring-2 focus:ring-red-500/50 outline-none">
                    <option value="">{{ t('admin.logs.allActions') }}</option>
                    <option v-for="a in actionOptions" :key="a" :value="a">{{ a }}</option>
                </select>
                <!-- Behavior Type -->
                <select v-model="filterBehavior" @change="applyFilters"
                    class="bg-neutral-950/50 border border-neutral-800 rounded-xl px-3 py-2 text-sm text-neutral-200 focus:ring-2 focus:ring-red-500/50 outline-none">
                    <option value="">{{ t('admin.logs.allBehaviors') }}</option>
                    <option v-for="b in behaviorOptions" :key="b" :value="b">{{ b }}</option>
                </select>
                <!-- User filter -->
                <input v-model="filterUser" @keydown.enter="applyFilters"
                    class="bg-neutral-950/50 border border-neutral-800 rounded-xl px-3 py-2 text-sm text-neutral-200 focus:ring-2 focus:ring-red-500/50 outline-none"
                    :placeholder="t('admin.logs.filterByUser')" />
                <!-- Date range -->
                <input type="datetime-local" v-model="filterStartDate" @change="applyFilters"
                    class="bg-neutral-950/50 border border-neutral-800 rounded-xl px-3 py-2 text-sm text-neutral-200 focus:ring-2 focus:ring-red-500/50 outline-none" />
                <input type="datetime-local" v-model="filterEndDate" @change="applyFilters"
                    class="bg-neutral-950/50 border border-neutral-800 rounded-xl px-3 py-2 text-sm text-neutral-200 focus:ring-2 focus:ring-red-500/50 outline-none" />
            </div>
        </section>

        <!-- Loading -->
        <div v-if="loading" class="text-center py-10">
            <div class="w-8 h-8 border-4 border-red-500/20 border-t-red-500 rounded-full animate-spin mx-auto mb-4">
            </div>
        </div>

        <!-- Table -->
        <div v-else class="bg-neutral-950/50 rounded-2xl border border-neutral-800/50 overflow-hidden backdrop-blur-sm">
            <div class="overflow-x-auto">
                <table class="w-full text-left border-collapse text-sm">
                    <thead>
                        <tr
                            class="bg-neutral-900/80 border-b border-neutral-800/80 text-xs font-semibold text-neutral-400 uppercase tracking-wider">
                            <th class="p-4">{{ t('admin.logs.date') }}</th>
                            <th class="p-4">{{ t('admin.logs.user') }}</th>
                            <th class="p-4">{{ t('admin.logs.action') }}</th>
                            <th class="p-4">{{ t('admin.logs.type') }}</th>
                            <th class="p-4">{{ t('admin.logs.description') }}</th>
                            <th class="p-4">{{ t('admin.logs.risk') }}</th>
                            <th class="p-4">{{ t('admin.logs.ip') }}</th>
                        </tr>
                    </thead>
                    <tbody class="divide-y divide-neutral-800/50">
                        <tr v-for="log in logs" :key="log.id"
                            class="hover:bg-neutral-800/20 transition-colors cursor-pointer" @click="openDetail(log)">
                            <td class="p-4 text-neutral-500 whitespace-nowrap text-xs">
                                {{ formatDate(log.created_at) }}
                            </td>
                            <td class="p-4 text-neutral-200 font-medium whitespace-nowrap">
                                {{ log.username || '—' }}
                            </td>
                            <td class="p-4">
                                <span
                                    class="px-2 py-0.5 rounded-lg text-[11px] font-bold uppercase bg-neutral-800 text-neutral-300 border border-neutral-700">
                                    {{ log.action }}
                                </span>
                            </td>
                            <td class="p-4 text-neutral-400 text-xs capitalize">{{ log.behavior_type }}</td>
                            <td class="p-4 text-neutral-300 max-w-xs truncate">
                                {{ log.description || '—' }}
                            </td>
                            <td class="p-4">
                                <span class="px-2 py-0.5 rounded-full text-[10px] font-bold uppercase border"
                                    :class="getRiskClasses(log.risk_level)">
                                    {{ log.risk_level }}
                                </span>
                            </td>
                            <td class="p-4 text-neutral-500 font-mono text-xs whitespace-nowrap">
                                {{ log.ip_address || '—' }}
                            </td>
                        </tr>
                        <tr v-if="logs.length === 0">
                            <td colspan="7" class="p-12 text-center text-neutral-500">
                                {{ t('admin.logs.noLogs') }}
                            </td>
                        </tr>
                    </tbody>
                </table>
            </div>

            <!-- Pagination -->
            <div class="flex items-center justify-between px-4 py-3 border-t border-neutral-800/50">
                <button @click="prevPage" :disabled="page === 1" :class="[
                    page === 1 ? 'text-neutral-600 cursor-not-allowed' : 'text-neutral-300 hover:text-white',
                    'px-3 py-1.5 text-sm font-medium rounded-lg transition'
                ]">
                    ← {{ t('common.actions.prev') }}
                </button>
                <span class="text-xs text-neutral-500">
                    {{ t('admin.logs.pageInfo', { current: page, total: totalPages }) }}
                </span>
                <button @click="nextPage" :disabled="page >= totalPages" :class="[
                    page >= totalPages ? 'text-neutral-600 cursor-not-allowed' : 'text-neutral-300 hover:text-white',
                    'px-3 py-1.5 text-sm font-medium rounded-lg transition'
                ]">
                    {{ t('common.actions.next') }} →
                </button>
            </div>
        </div>

        <!-- Detail Modal -->
        <div v-if="detailModal"
            class="fixed inset-0 bg-black/60 backdrop-blur-sm z-50 flex items-center justify-center p-4">
            <div
                class="bg-neutral-900 border border-neutral-800 rounded-2xl shadow-xl w-full max-w-2xl max-h-[85vh] flex flex-col">
                <div class="p-6 border-b border-neutral-800 flex justify-between items-center">
                    <div>
                        <h3 class="text-xl font-semibold text-white">{{ t('admin.logs.detail') }}</h3>
                        <p class="text-sm text-neutral-400 mt-1">{{ selectedLog?.action }} —
                            {{ formatDate(selectedLog?.created_at || '') }}</p>
                    </div>
                    <button @click="detailModal = false" class="text-neutral-400 hover:text-white transition">
                        <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                                d="M6 18L18 6M6 6l12 12" />
                        </svg>
                    </button>
                </div>
                <div v-if="selectedLog" class="p-6 overflow-y-auto flex-1 space-y-4">
                    <!-- Key info grid -->
                    <div class="grid grid-cols-2 gap-4">
                        <div class="space-y-1">
                            <div class="text-[10px] font-semibold uppercase text-neutral-500">{{
                                t('admin.logs.user') }}</div>
                            <div class="text-sm text-white">{{ selectedLog.username || '—' }}</div>
                        </div>
                        <div class="space-y-1">
                            <div class="text-[10px] font-semibold uppercase text-neutral-500">{{
                                t('admin.logs.risk') }}</div>
                            <span class="px-2 py-0.5 rounded-full text-[10px] font-bold uppercase border"
                                :class="getRiskClasses(selectedLog.risk_level)">
                                {{ selectedLog.risk_level }}
                            </span>
                        </div>
                        <div class="space-y-1">
                            <div class="text-[10px] font-semibold uppercase text-neutral-500">{{
                                t('admin.logs.action') }}</div>
                            <div class="text-sm text-white">{{ selectedLog.action }}</div>
                        </div>
                        <div class="space-y-1">
                            <div class="text-[10px] font-semibold uppercase text-neutral-500">{{
                                t('admin.logs.type') }}</div>
                            <div class="text-sm text-white capitalize">{{ selectedLog.behavior_type }}</div>
                        </div>
                        <div class="space-y-1">
                            <div class="text-[10px] font-semibold uppercase text-neutral-500">{{
                                t('admin.logs.ip') }}</div>
                            <div class="text-sm text-white font-mono">{{ selectedLog.ip_address || '—' }}</div>
                        </div>
                        <div class="space-y-1">
                            <div class="text-[10px] font-semibold uppercase text-neutral-500">{{
                                t('admin.logs.device') }}</div>
                            <div class="text-sm text-white">{{ formatUserAgent(selectedLog.user_agent) }}</div>
                        </div>
                    </div>

                    <!-- Description -->
                    <div v-if="selectedLog.description"
                        class="p-4 rounded-xl bg-neutral-950/50 border border-neutral-800">
                        <div class="text-[10px] font-semibold uppercase text-neutral-500 mb-2">{{
                            t('admin.logs.description') }}</div>
                        <div class="text-sm text-neutral-200">{{ selectedLog.description }}</div>
                    </div>

                    <!-- User Agent (full) -->
                    <div class="p-4 rounded-xl bg-neutral-950/50 border border-neutral-800">
                        <div class="text-[10px] font-semibold uppercase text-neutral-500 mb-2">{{
                            t('admin.logs.userAgent') }}</div>
                        <div class="text-xs text-neutral-400 font-mono break-all leading-relaxed">
                            {{ selectedLog.user_agent || '—' }}
                        </div>
                    </div>

                    <!-- Fingerprint -->
                    <div class="p-4 rounded-xl bg-neutral-950/50 border border-neutral-800">
                        <div class="text-[10px] font-semibold uppercase text-neutral-500 mb-2">{{
                            t('admin.logs.fingerprint') }}</div>
                        <div class="text-xs text-neutral-400 font-mono break-all">
                            {{ selectedLog.device_fingerprint || '—' }}
                        </div>
                    </div>

                    <!-- Details JSON -->
                    <div v-if="selectedLog.details" class="p-4 rounded-xl bg-neutral-950/50 border border-neutral-800">
                        <div class="text-[10px] font-semibold uppercase text-neutral-500 mb-2">{{
                            t('admin.logs.details') }}</div>
                        <pre
                            class="text-xs text-neutral-300 font-mono whitespace-pre-wrap break-all leading-relaxed max-h-64 overflow-y-auto">{{ formatDetails(selectedLog.details) }}</pre>
                    </div>
                </div>
            </div>
        </div>
    </div>
</template>

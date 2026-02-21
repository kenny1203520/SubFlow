<script setup lang="ts">
import { ref, onMounted } from 'vue';
import { useI18n } from 'vue-i18n';
import http from '../../http';

const { t } = useI18n();

const stats = ref({
    totalUsers: 0,
    blockedUsers: 0,
    suspendedUsers: 0,
    activeSessions: 0,
    blockedIps: 0
});
const loading = ref(true);

onMounted(async () => {
    try {
        const res = await http.get('/api/admin/stats');
        if (res.data.status === 'ok') {
            stats.value = res.data.stats;
        }
    } catch (e) {
        // fallback silently
    } finally {
        loading.value = false;
    }
});
</script>

<template>
    <div class="space-y-6">
        <div>
            <h2 class="text-3xl font-bold text-white mb-2">{{ t('admin.dashboard') }}</h2>
            <p class="text-neutral-400">{{ t('admin.dashboardWelcome') }}</p>
        </div>

        <div v-if="loading" class="text-center py-10">
            <div class="w-8 h-8 border-4 border-red-500/20 border-t-red-500 rounded-full animate-spin mx-auto mb-4">
            </div>
        </div>

        <div v-else class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-5 gap-6 mt-8">
            <div class="bg-neutral-900/50 border border-neutral-800 rounded-2xl p-6">
                <div class="text-neutral-400 text-sm font-medium mb-1">{{ t('admin.totalUsers') }}</div>
                <div class="text-3xl font-bold text-white">{{ stats.totalUsers }}</div>
            </div>

            <div class="bg-neutral-900/50 border border-neutral-800 rounded-2xl p-6">
                <div class="text-neutral-400 text-sm font-medium mb-1">{{ t('admin.activeSessions') }}</div>
                <div class="text-3xl font-bold text-emerald-400">{{ stats.activeSessions }}</div>
            </div>

            <div class="bg-neutral-900/50 border border-neutral-800 rounded-2xl p-6">
                <div class="text-neutral-400 text-sm font-medium mb-1">{{ t('admin.bannedUsers') }}</div>
                <div class="text-3xl font-bold text-red-400">{{ stats.blockedUsers }}</div>
            </div>

            <div class="bg-neutral-900/50 border border-neutral-800 rounded-2xl p-6">
                <div class="text-neutral-400 text-sm font-medium mb-1">{{ t('admin.suspendedUsers') }}</div>
                <div class="text-3xl font-bold text-orange-400">{{ stats.suspendedUsers }}</div>
            </div>

            <div class="bg-neutral-900/50 border border-neutral-800 rounded-2xl p-6">
                <div class="text-neutral-400 text-sm font-medium mb-1">{{ t('admin.blockedIps') }}</div>
                <div class="text-3xl font-bold text-yellow-400">{{ stats.blockedIps }}</div>
            </div>
        </div>

        <div class="mt-8 bg-neutral-900/30 border border-neutral-800 rounded-2xl p-8 text-center text-neutral-400">
            {{ t('admin.useSidebar') }}
        </div>
    </div>
</template>

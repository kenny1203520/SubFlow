<script setup lang="ts">
import { ref, onMounted } from 'vue';
import { useI18n } from 'vue-i18n';
import { socket } from '../socket';
import MainLayout from './MainLayout.vue';

const { t } = useI18n();
const stats = ref<any>({
    totalUsers: 0,
    totalGroups: 0,
    activeSubscriptions: 0,
    totalRevenue: 0
});
const loading = ref(true);

const fetchAdminStats = () => {
    loading.value = true;
    socket.emit('admin:stats', (res: any) => {
        if (res.status === 'ok') {
            stats.value = res.stats;
        }
        loading.value = false;
    });
};

onMounted(() => {
    if (socket.connected) {
        fetchAdminStats();
    } else {
        socket.on('connect', fetchAdminStats);
    }
});
</script>

<template>
    <MainLayout>
        <div class="admin-container animate-fade-in-up">
            <header class="page-header mb-8">
                <h1 class="page-title">{{ t('admin.adminDashboard') }}</h1>
                <p class="text-muted">{{ t('admin.subtitle', 'System overview and management') }}</p>
            </header>

            <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-6">
                <!-- Total Users -->
                <div class="stat-card glass-card">
                    <div class="stat-icon bg-primary-100 text-primary-600">
                        <svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4.354a4 4 0 110 5.292M15 21H3v-1a6 6 0 0112 0v1zm0 0h6v-1a6 6 0 00-9-5.197M13 7a4 4 0 11-8 0 4 4 0 018 0z" /></svg>
                    </div>
                    <div class="stat-data">
                        <h3 class="stat-label">{{ t('admin.totalUsers') }}</h3>
                        <p class="stat-value">{{ stats.totalUsers }}</p>
                    </div>
                </div>

                <!-- Total Groups -->
                <div class="stat-card glass-card">
                    <div class="stat-icon bg-emerald-100 text-emerald-600" style="background-color: #d1fae5; color: #059669;">
                        <svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z" /></svg>
                    </div>
                    <div class="stat-data">
                        <h3 class="stat-label">{{ t('admin.totalGroups') }}</h3>
                        <p class="stat-value">{{ stats.totalGroups }}</p>
                    </div>
                </div>

                <!-- Active Subs -->
                <div class="stat-card glass-card">
                    <div class="stat-icon bg-amber-100 text-amber-600" style="background-color: #fef3c7; color: #d97706;">
                        <svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z" /></svg>
                    </div>
                    <div class="stat-data">
                        <h3 class="stat-label">{{ t('admin.activeSubscriptions') }}</h3>
                        <p class="stat-value">{{ stats.activeSubscriptions }}</p>
                    </div>
                </div>

                <!-- Revenue -->
                <div class="stat-card glass-card">
                    <div class="stat-icon bg-blue-100 text-blue-600" style="background-color: #dbeafe; color: #2563eb;">
                        <svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8c-1.657 0-3 .895-3 2s1.343 2 3 2 3 .895 3 2-1.343 2-3 2m0-8c1.11 0 2.08.402 2.599 1M12 8V7m0 1v8m0 0v1m0-1c-1.11 0-2.08-.402-2.599-1M21 12a9 9 0 11-18 0 9 9 0 0118 0z" /></svg>
                    </div>
                    <div class="stat-data">
                        <h3 class="stat-label">{{ t('admin.totalRevenue') }}</h3>
                        <p class="stat-value">${{ stats.totalRevenue.toFixed(2) }}</p>
                    </div>
                </div>
            </div>

            <div class="mt-8 grid grid-cols-1 lg:grid-cols-3 gap-8">
                <section class="lg:col-span-2 glass-panel p-6">
                    <div class="flex items-center justify-between mb-6">
                        <h3 class="text-xl font-bold">{{ t('admin.systemLogs') }}</h3>
                        <button class="btn btn-secondary btn-sm">{{ t('common.actions.viewAll') }}</button>
                    </div>
                    <div class="log-table-container">
                        <div class="empty-state">
                             <div class="empty-icon">📂</div>
                             <p>{{ t('admin.noLogs', 'System logs will appear here') }}</p>
                        </div>
                    </div>
                </section>

                <section class="glass-panel p-6">
                    <h3 class="text-xl font-bold mb-6">{{ t('admin.reconciliation') }}</h3>
                    <div class="empty-state">
                        <div class="empty-icon">📊</div>
                        <p>{{ t('admin.notReady', 'Feature in progress') }}</p>
                    </div>
                </section>
            </div>
        </div>
    </MainLayout>
</template>

<style scoped>
.page-title {
    font-size: 2rem;
    font-weight: 800;
    margin-bottom: 0.25rem;
}

.stat-card {
    padding: 1.5rem;
    display: flex;
    align-items: center;
    gap: 1.25rem;
}

.stat-icon {
    width: 56px;
    height: 56px;
    border-radius: 16px;
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
}

.stat-label {
    font-size: 0.8125rem;
    font-weight: 600;
    color: var(--slate-500);
    text-transform: uppercase;
    letter-spacing: 0.025em;
    margin-bottom: 0.25rem;
}

.stat-value {
    font-size: 1.75rem;
    font-weight: 800;
    color: var(--slate-900);
}

.empty-state {
    padding: 4rem 2rem;
    text-align: center;
    color: var(--slate-400);
}

.empty-icon {
    font-size: 3rem;
    margin-bottom: 1rem;
    opacity: 0.5;
}

.log-table-container {
    min-height: 300px;
    display: flex;
    align-items: center;
    justify-content: center;
    background: rgba(0,0,0,0.02);
    border-radius: var(--radius-lg);
}
</style>

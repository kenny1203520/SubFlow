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
        <div class="admin-container">
            <header class="page-header">
                <h1 class="page-title">{{ t('admin.adminDashboard') }}</h1>
            </header>

            <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-6">
                <!-- Total Users -->
                <div class="stat-card card">
                    <h3 class="stat-label">{{ t('admin.totalUsers') }}</h3>
                    <p class="stat-value">{{ stats.totalUsers }}</p>
                </div>

                <!-- Total Groups -->
                <div class="stat-card card">
                    <h3 class="stat-label">{{ t('admin.totalGroups') }}</h3>
                    <p class="stat-value">{{ stats.totalGroups }}</p>
                </div>

                <!-- Active Subs -->
                <div class="stat-card card">
                    <h3 class="stat-label">{{ t('admin.activeSubscriptions') }}</h3>
                    <p class="stat-value">{{ stats.activeSubscriptions }}</p>
                </div>

                <!-- Revenue -->
                <div class="stat-card card">
                    <h3 class="stat-label">{{ t('admin.totalRevenue') }}</h3>
                    <p class="stat-value">${{ stats.totalRevenue.toFixed(2) }}</p>
                </div>
            </div>

            <div class="mt-8 grid grid-cols-1 md:grid-cols-2 gap-8">
                <section class="card">
                    <h3 class="section-title">{{ t('admin.systemLogs') }}</h3>
                    <div class="empty-state">Coming Soon</div>
                </section>
                <section class="card">
                    <h3 class="section-title">{{ t('admin.reconciliation') }}</h3>
                    <div class="empty-state">Coming Soon</div>
                </section>
            </div>
        </div>
    </MainLayout>
</template>

<style scoped>
.stat-card {
    padding: 1.5rem;
    text-align: center;
}
.stat-label {
    font-size: 0.875rem;
    color: var(--text-muted);
}
.stat-value {
    font-size: 2rem;
    font-weight: 800;
    color: var(--primary-color);
}
.empty-state {
    padding: 2rem;
    text-align: center;
    color: var(--text-muted);
}
</style>

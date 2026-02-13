<script setup lang="ts">
import { ref, onMounted } from 'vue';
import http from '../http';
import { useRouter } from 'vue-router';
import { useAuthStore } from '../stores/auth';
import { socket } from '../socket';
import { useI18n } from 'vue-i18n';
import MainLayout from './MainLayout.vue';

const router = useRouter();
const authStore = useAuthStore();
const { t } = useI18n();
const stats = ref({
    totalOwedToMe: 0,
    totalIOwe: 0,
    activeSubscriptions: 0
});

const fetchStats = () => {
    socket.emit('dashboard:stats', (res: any) => {
        if (res.status === 'ok') {
            stats.value = res.stats;
        }
    });
};

onMounted(async () => {
    if (!authStore.user) {
        try {
            const res = await http.get('/auth/user');
            authStore.setUser(res.data);
        } catch (err) {
            router.push('/auth');
        }
    }

    if (socket.connected) {
        fetchStats();
    } else {
        socket.on('connect', fetchStats);
    }
});
</script>

<template>
    <MainLayout>
        <div class="dashboard-wrapper">
            <header class="page-header">
                <div>
                    <h1 class="page-title">{{ t('dashboard.dashboard') }}</h1>
                    <p class="page-subtitle">{{ t('dashboard.welcomeBack', { name: authStore.user?.username }) }}</p>
                </div>
            </header>

            <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-6">
                <!-- Owe Card -->
                <div class="stat-card card">
                    <div class="icon-wrapper bg-danger-light">
                        <!-- Icon: Trending Down -->
                        <svg xmlns="http://www.w3.org/2000/svg" class="h-8 w-8 text-danger" fill="none"
                            viewBox="0 0 24 24" stroke="currentColor">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                                d="M13 17h8m0 0V9m0 8l-8-8-4 4-6-6" />
                        </svg>
                    </div>
                    <div>
                        <h3 class="stat-label">{{ t('dashboard.totalOwed') }}</h3>
                        <p :class="['stat-value', stats.totalIOwe > 0 ? 'text-danger' : 'text-neutral']">
                            ${{ stats.totalIOwe.toFixed(2) }}
                        </p>
                    </div>
                </div>

                <!-- Owed To You Card -->
                <div class="stat-card card">
                    <div class="icon-wrapper bg-success-light">
                        <!-- Icon: Trending Up -->
                        <svg xmlns="http://www.w3.org/2000/svg" class="h-8 w-8 text-success" fill="none"
                            viewBox="0 0 24 24" stroke="currentColor">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                                d="M13 7h8m0 0v8m0-8l-8 8-4-4-6 6" />
                        </svg>
                    </div>
                    <div>
                        <h3 class="stat-label">{{ t('dashboard.totalOwedToYou') }}</h3>
                        <p :class="['stat-value', stats.totalOwedToMe > 0 ? 'text-success' : 'text-neutral']">
                            ${{ stats.totalOwedToMe.toFixed(2) }}
                        </p>
                    </div>
                </div>

                <!-- Subscriptions Card -->
                <div class="stat-card card">
                    <div class="icon-wrapper bg-primary-light">
                        <!-- Icon: Refresh/Cycle -->
                        <svg xmlns="http://www.w3.org/2000/svg" class="h-8 w-8 text-primary" fill="none"
                            viewBox="0 0 24 24" stroke="currentColor">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                                d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
                        </svg>
                    </div>
                    <div>
                        <h3 class="stat-label">{{ t('dashboard.activeSubscriptions') }}</h3>
                        <p class="stat-value text-primary">{{ stats.activeSubscriptions }}</p>
                    </div>
                </div>
            </div>

            <!-- Recent Activity -->
            <section class="activity-section mt-8">
                <h3 class="section-title">{{ t('dashboard.recentActivity') }}</h3>
                <div class="card empty-activity">
                    <div class="empty-icon">
                        <!-- Icon: Inbox -->
                        <svg xmlns="http://www.w3.org/2000/svg" class="h-12 w-12 text-muted" fill="none"
                            viewBox="0 0 24 24" stroke="currentColor">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                                d="M20 13V6a2 2 0 00-2-2H6a2 2 0 00-2 2v7m16 0v5a2 2 0 01-2 2H6a2 2 0 01-2-2v-5m16 0h-2.586a1 1 0 00-.707.293l-2.414 2.414a1 1 0 01-.707.293h-3.172a1 1 0 01-.707-.293l-2.414-2.414A1 1 0 006.586 13H4" />
                        </svg>
                    </div>
                    <p>{{ t('dashboard.noActivity') }}</p>
                </div>
            </section>
        </div>
    </MainLayout>
</template>

<style scoped>
.page-header {
    margin-bottom: 2rem;
}

.page-title {
    font-size: 1.875rem;
    font-weight: 700;
    color: var(--text-main);
    margin-bottom: 0.25rem;
}

.page-subtitle {
    color: var(--text-muted);
    font-size: 1rem;
}

/* Stat Cards */
.stat-card {
    display: flex;
    align-items: center;
    gap: 1.25rem;
    padding: 1.5rem;
}

.icon-wrapper {
    width: 56px;
    height: 56px;
    border-radius: var(--radius-lg);
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 1.75rem;
}

.bg-danger-light {
    background-color: #fee2e2;
}

.bg-success-light {
    background-color: #dcfce7;
}

.bg-primary-light {
    background-color: var(--primary-100);
}

.stat-label {
    font-size: 0.875rem;
    font-weight: 600;
    color: var(--text-muted);
    text-transform: uppercase;
    letter-spacing: 0.05em;
    margin-bottom: 0.25rem;
}

.stat-value {
    font-size: 1.75rem;
    font-weight: 800;
    line-height: 1;
}

.text-danger {
    color: var(--danger-color);
}

.text-success {
    color: var(--success-color);
}

.text-primary {
    color: var(--primary-color);
}

.text-neutral {
    color: var(--text-main);
}

/* Activity Section */
.mt-8 {
    margin-top: 2.5rem;
}

.section-title {
    font-size: 1.25rem;
    font-weight: 600;
    color: var(--text-main);
    margin-bottom: 1rem;
}

.empty-activity {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    padding: 4rem 2rem;
    border: 2px dashed var(--border-color);
    background-color: transparent;
    box-shadow: none;
}

.empty-icon {
    font-size: 3rem;
    margin-bottom: 1rem;
    opacity: 0.5;
}
</style>

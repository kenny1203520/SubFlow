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
        <div class="dashboard-wrapper space-y-8">
            <!-- Welcome Banner -->
            <header
                class="relative overflow-hidden rounded-3xl bg-primary-gradient p-8 text-white shadow-lg animate-fade-in">
                <div class="absolute top-0 right-0 p-4 opacity-10">
                    <svg xmlns="http://www.w3.org/2000/svg" class="h-64 w-64" fill="currentColor" viewBox="0 0 24 24">
                        <path
                            d="M12 2L2 7l10 5 10-5-10-5zm0 9l2.5-1.25L12 8.5l-2.5 1.25L12 11zm0 2.5l-5-2.5-5 2.5L12 22l10-8.5-5-2.5-5 2.5z" />
                    </svg>
                </div>
                <div class="relative z-10">
                    <h1 class="text-3xl font-extrabold mb-2">{{ t('dashboard.dashboard') }}</h1>
                    <p class="text-primary-100 text-lg">
                        {{ t('dashboard.welcomeBack', { name: authStore.user?.username }) }}
                    </p>
                </div>
            </header>

            <!-- Stats Grid -->
            <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-6 animate-fade-in"
                style="animation-delay: 0.1s">
                <!-- Owe Card -->
                <div class="glass-card p-6 flex items-center gap-4 relative overflow-hidden group">
                    <div class="absolute right-0 top-0 p-4 opacity-5 group-hover:scale-110 transition-transform">
                        <svg xmlns="http://www.w3.org/2000/svg" class="h-24 w-24" fill="currentColor"
                            viewBox="0 0 24 24">
                            <path d="M13 17h8m0 0V9m0 8l-8-8-4 4-6-6" stroke="currentColor" stroke-width="2"
                                stroke-linecap="round" stroke-linejoin="round" />
                        </svg>
                    </div>
                    <div
                        class="w-14 h-14 rounded-2xl bg-red-100 text-red-500 flex items-center justify-center shadow-sm">
                        <svg xmlns="http://www.w3.org/2000/svg" class="h-8 w-8" fill="none" viewBox="0 0 24 24"
                            stroke="currentColor">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                                d="M13 17h8m0 0V9m0 8l-8-8-4 4-6-6" />
                        </svg>
                    </div>
                    <div>
                        <h3 class="text-sm font-semibold text-slate-500 uppercase tracking-wider">{{
                            t('dashboard.totalOwed') }}</h3>
                        <p class="text-3xl font-extrabold mt-1"
                            :class="stats.totalIOwe > 0 ? 'text-red-500' : 'text-slate-800'">
                            ${{ stats.totalIOwe.toFixed(2) }}
                        </p>
                    </div>
                </div>

                <!-- Owed To You Card -->
                <div class="glass-card p-6 flex items-center gap-4 relative overflow-hidden group">
                    <div class="absolute right-0 top-0 p-4 opacity-5 group-hover:scale-110 transition-transform">
                        <svg xmlns="http://www.w3.org/2000/svg" class="h-24 w-24" fill="currentColor"
                            viewBox="0 0 24 24">
                            <path d="M13 7h8m0 0v8m0-8l-8 8-4-4-6 6" stroke="currentColor" stroke-width="2"
                                stroke-linecap="round" stroke-linejoin="round" />
                        </svg>
                    </div>
                    <div
                        class="w-14 h-14 rounded-2xl bg-emerald-100 text-emerald-500 flex items-center justify-center shadow-sm">
                        <svg xmlns="http://www.w3.org/2000/svg" class="h-8 w-8" fill="none" viewBox="0 0 24 24"
                            stroke="currentColor">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                                d="M13 7h8m0 0v8m0-8l-8 8-4-4-6 6" />
                        </svg>
                    </div>
                    <div>
                        <h3 class="text-sm font-semibold text-slate-500 uppercase tracking-wider">{{
                            t('dashboard.totalOwedToYou') }}</h3>
                        <p class="text-3xl font-extrabold mt-1"
                            :class="stats.totalOwedToMe > 0 ? 'text-emerald-500' : 'text-slate-800'">
                            ${{ stats.totalOwedToMe.toFixed(2) }}
                        </p>
                    </div>
                </div>

                <!-- Subscriptions Card -->
                <div class="glass-card p-6 flex items-center gap-4 relative overflow-hidden group">
                    <div class="absolute right-0 top-0 p-4 opacity-5 group-hover:scale-110 transition-transform">
                        <svg xmlns="http://www.w3.org/2000/svg" class="h-24 w-24" fill="currentColor"
                            viewBox="0 0 24 24">
                            <path
                                d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"
                                stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" />
                        </svg>
                    </div>
                    <div
                        class="w-14 h-14 rounded-2xl bg-indigo-100 text-indigo-500 flex items-center justify-center shadow-sm">
                        <svg xmlns="http://www.w3.org/2000/svg" class="h-8 w-8" fill="none" viewBox="0 0 24 24"
                            stroke="currentColor">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                                d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
                        </svg>
                    </div>
                    <div>
                        <h3 class="text-sm font-semibold text-slate-500 uppercase tracking-wider">{{
                            t('dashboard.activeSubscriptions') }}</h3>
                        <p class="text-3xl font-extrabold mt-1 text-slate-800">
                            {{ stats.activeSubscriptions }}
                        </p>
                    </div>
                </div>
            </div>

            <!-- Recent Activity & Quick Actions -->
            <div class="grid grid-cols-1 lg:grid-cols-3 gap-8 animate-fade-in" style="animation-delay: 0.2s">
                <div class="lg:col-span-2 space-y-6">
                    <h3 class="text-xl font-bold text-slate-800 flex items-center gap-2">
                        <div
                            class="w-8 h-8 rounded-lg bg-primary-100 flex items-center justify-center text-primary-600">
                            <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" fill="none" viewBox="0 0 24 24"
                                stroke="currentColor">
                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                                    d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
                            </svg>
                        </div>
                        {{ t('dashboard.recentActivity') }}
                    </h3>

                    <div
                        class="glass-panel p-12 text-center border-2 border-dashed border-slate-200 bg-white/40 group hover:border-primary-300 transition-colors">
                        <div
                            class="w-20 h-20 bg-white rounded-2xl flex items-center justify-center mx-auto mb-6 text-slate-300 shadow-sm group-hover:scale-110 transition-transform duration-500">
                            <svg xmlns="http://www.w3.org/2000/svg" class="h-10 w-10" fill="none" viewBox="0 0 24 24"
                                stroke="currentColor">
                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5"
                                    d="M20 13V6a2 2 0 00-2-2H6a2 2 0 00-2 2v7m16 0v5a2 2 0 01-2 2H6a2 2 0 01-2-2v-5m16 0h-2.586a1 1 0 00-.707.293l-2.414 2.414a1 1 0 01-.707.293h-3.172a1 1 0 01-.707-.293l-2.414-2.414A1 1 0 006.586 13H4" />
                            </svg>
                        </div>
                        <h4 class="text-lg font-bold text-slate-700 mb-2">{{ t('dashboard.noActivityTitle') }}</h4>
                        <p class="text-slate-400 max-w-xs mx-auto text-sm">{{ t('dashboard.noActivityDesc') }}</p>
                    </div>
                </div>

                <div class="space-y-6">
                    <h3 class="text-xl font-bold text-slate-800">{{ t('dashboard.quickActions', 'Quick Actions') }}</h3>
                    <div class="glass-card p-4 space-y-3">
                        <button @click="router.push('/groups')"
                            class="w-full flex items-center gap-3 p-3 rounded-xl hover:bg-slate-50 transition-colors text-left group">
                            <div
                                class="w-10 h-10 rounded-lg bg-primary-50 text-primary-600 flex items-center justify-center group-hover:bg-primary-600 group-hover:text-white transition-colors">
                                <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" fill="none" viewBox="0 0 24 24"
                                    stroke="currentColor">
                                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                                        d="M12 4v16m8-8H4" />
                                </svg>
                            </div>
                            <span class="font-medium text-slate-700 group-hover:text-primary-700">Create Group</span>
                        </button>
                        <button @click="router.push('/subscriptions')"
                            class="w-full flex items-center gap-3 p-3 rounded-xl hover:bg-slate-50 transition-colors text-left group">
                            <div
                                class="w-10 h-10 rounded-lg bg-emerald-50 text-emerald-600 flex items-center justify-center group-hover:bg-emerald-600 group-hover:text-white transition-colors">
                                <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" fill="none" viewBox="0 0 24 24"
                                    stroke="currentColor">
                                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                                        d="M12 9v3m0 0v3m0-3h3m-3 0H9m12 0a9 9 0 11-18 0 9 9 0 0118 0z" />
                                </svg>
                            </div>
                            <span class="font-medium text-slate-700 group-hover:text-emerald-700">Add
                                Subscription</span>
                        </button>
                    </div>
                </div>
            </div>
        </div>
    </MainLayout>
</template>

<style scoped>
/* Scoped styles are minimal as we use utility classes from style.css */
</style>

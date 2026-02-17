<script setup lang="ts">
import { ref, onMounted } from 'vue';
import { socket } from '../socket';
import MainLayout from './MainLayout.vue';
import { useI18n } from 'vue-i18n';

const { t } = useI18n();
const subscriptions = ref<any[]>([]);
const loading = ref(true);

const fetchData = () => {
    loading.value = true;
    socket.emit('subscription:all', (res: any) => {
        if (res.status === 'ok') {
            subscriptions.value = res.subscriptions;
            loading.value = false;
        }
    });
};

const updateStatus = (subscriptionId: string, status: string) => {
    socket.emit('subscription:update_status', { subscriptionId, status }, (res: any) => {
        if (res.status === 'ok') {
            fetchData();
        } else {
            alert(res.message);
        }
    });
};

onMounted(() => {
    if (socket.connected) {
        fetchData();
    } else {
        socket.once('connect', fetchData);
    }
});
</script>

<template>
    <MainLayout>
        <div class="flex flex-col gap-8 animate-fade-in">
            <div
                class="flex justify-between items-center bg-white/40 sticky top-0 z-10 backdrop-blur-md p-4 rounded-2xl border border-white/50 shadow-sm">
                <div>
                    <h1 class="text-2xl font-bold text-slate-800">{{ t('subscriptions.title') }}</h1>
                    <p class="text-slate-500 text-sm">Track your recurring payments</p>
                </div>
                <!-- Add Subscription button could go here if we have a global create modal, or rely on group detail -->
            </div>

            <div v-if="loading" class="flex flex-col items-center justify-center p-20">
                <div class="animate-spin rounded-full h-12 w-12 border-b-2 border-primary-600"></div>
                <p class="mt-4 text-slate-500 font-medium">{{ t('subscriptions.loading', 'Loading subscriptions...') }}
                </p>
            </div>

            <div v-else-if="subscriptions.length === 0"
                class="flex flex-col items-center justify-center p-16 glass-panel border-dashed border-2 border-slate-300">
                <div class="w-16 h-16 bg-slate-100 rounded-full flex items-center justify-center mb-4 text-slate-400">
                    <svg xmlns="http://www.w3.org/2000/svg" class="h-8 w-8" fill="none" viewBox="0 0 24 24"
                        stroke="currentColor">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                            d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
                    </svg>
                </div>
                <p class="text-slate-500 text-lg">{{ t('subscriptions.noSubscriptions') }}</p>
            </div>

            <div v-else class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-6">
                <div v-for="sub in subscriptions" :key="sub.id"
                    class="glass-card p-6 flex flex-col h-full relative overflow-hidden group hover:border-primary-300 transition-colors">
                    <div class="flex justify-between items-start mb-4">
                        <div class="flex items-center gap-3">
                            <div
                                class="w-10 h-10 rounded-lg bg-indigo-50 text-indigo-600 flex items-center justify-center font-bold shadow-sm">
                                {{ sub.name[0].toUpperCase() }}
                            </div>
                            <div>
                                <h3 class="font-bold text-slate-800 text-lg leading-tight">{{ sub.name }}</h3>
                                <span
                                    class="text-xs px-2 py-0.5 rounded-full font-bold uppercase tracking-wide inline-block mt-1"
                                    :class="{
                                        'bg-green-100 text-green-700': sub.status === 'active',
                                        'bg-yellow-100 text-yellow-700': sub.status === 'paused',
                                        'bg-red-100 text-red-700': sub.status === 'cancelled'
                                    }">
                                    {{ t(`subscriptions.${sub.status}`) }}
                                </span>
                            </div>
                        </div>
                    </div>

                    <div class="mb-6">
                        <div class="flex items-baseline gap-1">
                            <span class="text-3xl font-extrabold text-slate-800">${{ sub.amount }}</span>
                            <span class="text-sm text-slate-500 font-medium">/ {{ sub.cycle }}</span>
                        </div>
                        <p class="text-xs text-slate-400 mt-1 flex items-center gap-1">
                            <svg xmlns="http://www.w3.org/2000/svg" class="h-3 w-3" fill="none" viewBox="0 0 24 24"
                                stroke="currentColor">
                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                                    d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
                            </svg>
                            {{ t('subscriptions.nextPayment') }}: <span class="font-semibold text-slate-600">{{
                                sub.next_payment_date ? new Date(sub.next_payment_date).toLocaleDateString() : 'N/A'
                                }}</span>
                        </p>
                    </div>

                    <div class="mt-auto grid grid-cols-2 gap-2">
                        <button v-if="sub.status === 'active'" @click="updateStatus(sub.id, 'paused')"
                            class="px-3 py-2 rounded-lg bg-yellow-50 text-yellow-700 hover:bg-yellow-100 font-medium text-sm transition-colors flex items-center justify-center gap-1">
                            <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24"
                                stroke="currentColor">
                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                                    d="M10 9v6m4-6v6m7-3a9 9 0 11-18 0 9 9 0 0118 0z" />
                            </svg>
                            {{ t('subscriptions.pause') }}
                        </button>

                        <button v-if="sub.status === 'paused'" @click="updateStatus(sub.id, 'active')"
                            class="px-3 py-2 rounded-lg bg-green-50 text-green-700 hover:bg-green-100 font-medium text-sm transition-colors flex items-center justify-center gap-1">
                            <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24"
                                stroke="currentColor">
                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                                    d="M14.752 11.168l-3.197-2.132A1 1 0 0010 9.87v4.263a1 1 0 001.555.832l3.197-2.132a1 1 0 000-1.664z" />
                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                                    d="M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                            </svg>
                            {{ t('subscriptions.activate') }}
                        </button>

                        <button v-if="sub.status !== 'cancelled'" @click="updateStatus(sub.id, 'cancelled')"
                            class="px-3 py-2 rounded-lg bg-red-50 text-red-700 hover:bg-red-100 font-medium text-sm transition-colors flex items-center justify-center gap-1 col-span-1">
                            <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24"
                                stroke="currentColor">
                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                                    d="M6 18L18 6M6 6l12 12" />
                            </svg>
                            {{ t('subscriptions.cancel') }}
                        </button>

                        <button v-if="sub.status === 'cancelled'" disabled
                            class="col-span-2 px-3 py-2 rounded-lg bg-slate-100 text-slate-400 cursor-not-allowed font-medium text-sm flex items-center justify-center gap-1">
                            {{ t('subscriptions.cancelled') }}
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

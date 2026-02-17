<script setup lang="ts">
import { ref, onMounted } from 'vue';
import { useI18n } from 'vue-i18n';
import { socket } from '../socket';
import MainLayout from './MainLayout.vue';

const { t } = useI18n();
const wallets = ref<any[]>([]);
const loading = ref(true);

const fetchWallets = () => {
    loading.value = true;
    socket.emit('wallet:list', (res: any) => {
        if (res.status === 'ok') {
            wallets.value = res.wallets;
        }
        loading.value = false;
    });
};

onMounted(() => {
    if (socket.connected) {
        fetchWallets();
    } else {
        socket.on('connect', fetchWallets);
    }
});

const getWalletTypeName = (type: string) => {
    switch (type) {
        case 'root': return t('wallet.rootWallet');
        case 'group': return t('wallet.groupWallet');
        case 'service': return t('wallet.serviceWallet');
        default: return type;
    }
};
</script>

<template>
    <MainLayout>
        <div class="flex flex-col gap-8 animate-fade-in">
            <header>
                <h1 class="text-3xl font-extrabold text-slate-800">{{ t('wallet.wallet') }}</h1>
                <p class="text-slate-500 text-sm mt-1">{{ t('wallet.walletDesc', 'Manage your wallets and transactions')
                }}</p>
            </header>

            <!-- Loading State -->
            <div v-if="loading" class="flex flex-col items-center justify-center p-20">
                <div class="animate-spin rounded-full h-12 w-12 border-b-2 border-primary-600"></div>
                <p class="mt-4 text-slate-500 font-medium">{{ t('common.status.loading') }}</p>
            </div>

            <!-- Wallet Cards -->
            <div v-else-if="wallets.length > 0" class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
                <div v-for="wallet in wallets" :key="wallet.id"
                    class="relative overflow-hidden rounded-2xl p-6 text-white shadow-lg group hover:-translate-y-1 transition-all duration-300"
                    style="background: var(--primary-gradient);">
                    <!-- Decorative circle -->
                    <div
                        class="absolute -right-6 -top-6 w-24 h-24 rounded-full bg-white/10 group-hover:scale-125 transition-transform duration-500">
                    </div>
                    <div class="absolute -right-2 -bottom-8 w-32 h-32 rounded-full bg-white/5"></div>

                    <div class="relative z-10">
                        <span class="text-xs font-bold uppercase tracking-widest text-white/70">
                            {{ getWalletTypeName(wallet.wallet_type) }}
                        </span>
                        <h2 class="text-3xl font-extrabold mt-2 tracking-tight">
                            {{ wallet.currency }} {{ parseFloat(wallet.balance).toFixed(2) }}
                        </h2>
                    </div>

                    <div class="relative z-10 flex gap-2 mt-6">
                        <button
                            class="flex-1 px-4 py-2.5 rounded-xl bg-white/20 hover:bg-white/30 text-white font-semibold text-sm transition-colors backdrop-blur-sm">
                            {{ t('wallet.deposit') }}
                        </button>
                        <button
                            class="flex-1 px-4 py-2.5 rounded-xl border border-white/30 hover:bg-white/10 text-white font-semibold text-sm transition-colors">
                            {{ t('wallet.transfer') }}
                        </button>
                    </div>
                </div>
            </div>

            <!-- Empty State -->
            <div v-else
                class="flex flex-col items-center justify-center p-16 glass-panel border-dashed border-2 border-slate-200">
                <div
                    class="w-20 h-20 bg-white rounded-2xl flex items-center justify-center mb-6 text-slate-300 shadow-sm">
                    <svg xmlns="http://www.w3.org/2000/svg" class="h-10 w-10" fill="none" viewBox="0 0 24 24"
                        stroke="currentColor">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5"
                            d="M3 10h18M7 15h1m4 0h1m-7 4h12a3 3 0 003-3V8a3 3 0 00-3-3H6a3 3 0 00-3 3v8a3 3 0 003 3z" />
                    </svg>
                </div>
                <h4 class="text-lg font-bold text-slate-700 mb-2">{{ t('wallet.noWallets', 'No wallets yet') }}</h4>
                <p class="text-slate-400 max-w-xs text-center text-sm">
                    {{ t('wallet.noWalletsDesc',
                        'Wallets will appear here once you join a group or create a subscription.') }}
                </p>
            </div>

            <!-- Transactions Section -->
            <section>
                <h3 class="text-xl font-bold text-slate-800 mb-4 flex items-center gap-2">
                    <div class="w-8 h-8 rounded-lg bg-primary-100 flex items-center justify-center text-primary-600">
                        <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" fill="none" viewBox="0 0 24 24"
                            stroke="currentColor">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                                d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2" />
                        </svg>
                    </div>
                    {{ t('wallet.transactions') }}
                </h3>
                <div class="glass-panel p-12 text-center border-2 border-dashed border-slate-200 bg-white/40">
                    <p class="text-slate-400">{{ t('wallet.noTransactions') }}</p>
                </div>
            </section>
        </div>
    </MainLayout>
</template>

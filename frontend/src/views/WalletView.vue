<script setup lang="ts">
import { ref, onMounted } from 'vue';
import { useI18n } from 'vue-i18n';
import { socket } from '../socket';
import MainLayout from './MainLayout.vue';
import WalletCard from '../components/WalletCard.vue';

const { t } = useI18n();
const wallets = ref<any[]>([]);
const transactions = ref<any[]>([]);
const loading = ref(true);
const selectedWalletId = ref<string>('');

const fetchWallets = () => {
    loading.value = true;
    socket.emit('wallet:list', (res: any) => {
        if (res.status === 'ok') {
            wallets.value = res.wallets;
            // Automatically select first wallet to show transactions if available
            if (wallets.value.length > 0 && !selectedWalletId.value) {
                selectWallet(wallets.value[0].id);
            } else {
                loading.value = false;
            }
        } else {
            loading.value = false;
        }
    });
};

const selectWallet = (id: string) => {
    selectedWalletId.value = id;
    socket.emit('wallet:details', { walletId: id }, (res: any) => {
        if (res.status === 'ok') {
            transactions.value = res.transactions;
        }
        loading.value = false;
    });
};

const handleDeposit = (wallet: any) => {
    const amount = prompt("Enter amount to deposit (TWD):");
    if (amount) {
        socket.emit('wallet:deposit', { amount: parseFloat(amount), currency: wallet.currency }, (res: any) => {
            if (res.status === 'ok') {
                alert('Deposit successful!');
                fetchWallets(); // Refresh balance
            } else {
                alert(res.message || 'Deposit failed');
            }
        });
    }
};

const handleTransfer = (wallet: any) => {
    alert("Transfer UI coming soon!");
};

onMounted(() => {
    if (socket.connected) {
        fetchWallets();
    } else {
        socket.on('connect', fetchWallets);
    }
});
</script>

<template>
    <MainLayout>
        <div class="flex flex-col gap-8 animate-fade-in relative z-10">
            <header>
                <h1 class="text-3xl font-extrabold text-slate-800">{{ t('wallet.wallet', 'My Wallets') }}</h1>
                <p class="text-slate-500 text-sm mt-1">{{ t('wallet.walletDesc', 'Manage your wallets and transactions')
                    }}</p>
            </header>

            <!-- Loading State -->
            <div v-if="loading && wallets.length === 0" class="flex flex-col items-center justify-center p-20">
                <div class="animate-spin rounded-full h-12 w-12 border-b-2 border-primary-600"></div>
                <p class="mt-4 text-slate-500 font-medium">{{ t('common.status.loading') }}</p>
            </div>

            <div v-else>
                <!-- Wallets Grid -->
                <div v-if="wallets.length > 0" class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6 mb-8">
                    <WalletCard v-for="wallet in wallets" :key="wallet.id" :wallet="wallet"
                        :class="{ 'ring-4 ring-primary-200': selectedWalletId === wallet.id }"
                        @click="selectWallet(wallet.id)" @deposit="handleDeposit" @transfer="handleTransfer" />
                </div>

                <!-- Empty State -->
                <div v-else
                    class="flex flex-col items-center justify-center p-16 glass-panel border-dashed border-2 border-slate-200 mb-8">
                    <div
                        class="w-20 h-20 bg-white rounded-2xl flex items-center justify-center mb-6 text-slate-300 shadow-sm">
                        <svg xmlns="http://www.w3.org/2000/svg" class="h-10 w-10" fill="none" viewBox="0 0 24 24"
                            stroke="currentColor">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5"
                                d="M3 10h18M7 15h1m4 0h1m-7 4h12a3 3 0 003-3V8a3 3 0 00-3-3H6a3 3 0 00-3 3v8a3 3 0 003 3z" />
                        </svg>
                    </div>
                    <h4 class="text-lg font-bold text-slate-700 mb-2">{{ t('wallet.noWallets', 'No wallets yet') }}</h4>
                </div>

                <!-- Transactions Section -->
                <section class="animate-fade-in-up" v-if="selectedWalletId">
                    <h3 class="text-xl font-bold text-slate-800 mb-4 flex items-center gap-2">
                        <div
                            class="w-8 h-8 rounded-lg bg-primary-100 flex items-center justify-center text-primary-600">
                            <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" fill="none" viewBox="0 0 24 24"
                                stroke="currentColor">
                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                                    d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2" />
                            </svg>
                        </div>
                        {{ t('wallet.transactions', 'Recent Transactions') }}
                    </h3>

                    <div class="glass-panel overflow-hidden">
                        <table class="w-full text-left text-sm">
                            <thead class="bg-slate-50 text-slate-500 font-bold uppercase border-b border-slate-200">
                                <tr>
                                    <th class="p-4 pl-6">Date</th>
                                    <th class="p-4">Type</th>
                                    <th class="p-4">Description</th>
                                    <th class="p-4 text-right pr-6">Amount</th>
                                </tr>
                            </thead>
                            <tbody class="divide-y divide-slate-100">
                                <tr v-for="tx in transactions" :key="tx.id" class="hover:bg-slate-50 transition-colors">
                                    <td class="p-4 pl-6 text-slate-500 whitespace-nowrap">
                                        {{ new Date(tx.created_at).toLocaleDateString() }}
                                        <span class="text-xs text-slate-400 block">{{ new
                                            Date(tx.created_at).toLocaleTimeString() }}</span>
                                    </td>
                                    <td class="p-4">
                                        <span class="px-2 py-1 rounded-md text-xs font-bold uppercase" :class="{
                                            'bg-green-100 text-green-700': tx.type === 'deposit' || tx.type === 'transfer_in', // Fixed: was tx.type.includes('in') which is risky
                                            'bg-slate-100 text-slate-700': tx.type === 'withdrawal' || tx.type === 'transfer_out' // Fixed logic
                                        }">
                                            {{ tx.type.replace('_', ' ') }}
                                        </span>
                                    </td>
                                    <td class="p-4 text-slate-700 font-medium">
                                        {{ tx.description || 'No description' }}
                                    </td>
                                    <td class="p-4 text-right pr-6 font-bold font-mono text-base"
                                        :class="tx.amount > 0 ? 'text-green-600' : 'text-slate-800'">
                                        {{ tx.amount > 0 ? '+' : '' }}{{ tx.amount }} {{ tx.currency }}
                                    </td>
                                </tr>
                                <tr v-if="transactions.length === 0">
                                    <td colspan="4" class="p-12 text-center text-slate-400">
                                        {{ t('wallet.noTransactions', 'No transactions found for this wallet.') }}
                                    </td>
                                </tr>
                            </tbody>
                        </table>
                    </div>
                </section>
            </div>
        </div>
    </MainLayout>
</template>

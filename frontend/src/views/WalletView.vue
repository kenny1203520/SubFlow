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
        <div class="wallet-container">
            <header class="page-header">
                <h1 class="page-title">{{ t('wallet.wallet') }}</h1>
            </header>

            <div v-if="loading" class="loading-state">
                <div class="spinner"></div>
            </div>

            <div v-else class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
                <div v-for="wallet in wallets" :key="wallet.id" class="card wallet-card">
                    <div class="wallet-info">
                        <span class="wallet-type">{{ getWalletTypeName(wallet.wallet_type) }}</span>
                        <h2 class="wallet-balance">{{ wallet.currency }} {{ parseFloat(wallet.balance).toFixed(2) }}</h2>
                    </div>
                    <div class="wallet-actions">
                        <button class="btn btn-primary btn-sm">{{ t('wallet.deposit') }}</button>
                        <button class="btn btn-outline btn-sm">{{ t('wallet.transfer') }}</button>
                    </div>
                </div>
            </div>

            <section class="mt-8">
                <h3 class="section-title">{{ t('wallet.transactions') }}</h3>
                <div class="card empty-state">
                    <p>{{ t('wallet.noTransactions') }}</p>
                </div>
            </section>
        </div>
    </MainLayout>
</template>

<style scoped>
.wallet-card {
    padding: 1.5rem;
    display: flex;
    flex-direction: column;
    justify-content: space-between;
    min-height: 160px;
    background: linear-gradient(135deg, var(--primary-600) 0%, var(--primary-800) 100%);
    color: white;
    border: none;
}

.wallet-type {
    font-size: 0.875rem;
    opacity: 0.8;
    text-transform: uppercase;
    letter-spacing: 0.05em;
}

.wallet-balance {
    font-size: 2rem;
    font-weight: 800;
    margin-top: 0.5rem;
}

.wallet-actions {
    display: flex;
    gap: 0.5rem;
    margin-top: 1.5rem;
}

.btn-outline {
    background: transparent;
    border: 1px solid rgba(255, 255, 255, 0.4);
    color: white;
}

.btn-outline:hover {
    background: rgba(255, 255, 255, 0.1);
}

.empty-state {
    padding: 3rem;
    text-align: center;
    color: var(--text-muted);
    border: 2px dashed var(--border-color);
    background: transparent;
    box-shadow: none;
}

.loading-state {
    display: flex;
    justify-content: center;
    padding: 3rem;
}
</style>

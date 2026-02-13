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
        <div class="header">
            <h1>{{ t('subscriptions.title') }}</h1>
        </div>

        <div v-if="loading" class="state">{{ t('subscriptions.loading') }}</div>

        <div v-else-if="subscriptions.length === 0" class="empty-state">
            <p>{{ t('subscriptions.noSubscriptions') }}</p>
        </div>

        <div v-else class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
            <div v-for="sub in subscriptions" :key="sub.id" class="sub-card card">
                <div class="sub-header">
                    <div class="sub-title">
                        <h3>{{ sub.name }}</h3>
                        <span :class="['status-badge', sub.status]">{{ t(`subscriptions.${sub.status}`) }}</span>
                    </div>
                </div>
                <div class="amount-display">
                    ${{ sub.amount }} <span class="cycle">/ {{ t(`subscriptions.cycle`) }}: {{ sub.cycle }}</span>
                </div>

                <div class="sub-details">
                    <p>{{ t('subscriptions.nextPayment') }}: <strong>{{ sub.next_payment_date ? new
                        Date(sub.next_payment_date).toLocaleDateString() : 'N/A' }}</strong></p>
                </div>

                <div class="sub-actions">
                    <button v-if="sub.status === 'active'" @click="updateStatus(sub.id, 'paused')"
                        class="btn btn-pause">{{ t('subscriptions.pause') }}</button>
                    <button v-if="sub.status === 'paused'" @click="updateStatus(sub.id, 'active')"
                        class="btn btn-activate">{{ t('subscriptions.activate') }}</button>
                    <button v-if="sub.status !== 'cancelled'" @click="updateStatus(sub.id, 'cancelled')"
                        class="btn btn-cancel">{{ t('subscriptions.cancel') }}</button>
                    <button v-if="sub.status === 'cancelled'" disabled class="btn btn-disabled">{{
                        t('subscriptions.cancelled') }}</button>
                </div>
            </div>
        </div>
    </MainLayout>
</template>

<style scoped>
.header {
    margin-bottom: 2rem;
}

.header h1 {
    font-size: 2rem;
    font-weight: 700;
    color: var(--text-main);
    margin: 0;
}

.sub-card {
    display: flex;
    flex-direction: column;
    height: 100%;
}

.sub-header {
    display: flex;
    justify-content: space-between;
    align-items: start;
    margin-bottom: 1rem;
}

.sub-title {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
}

.sub-header h3 {
    margin: 0;
    font-size: 1.25rem;
    font-weight: 600;
}

.status-badge {
    font-size: 0.75rem;
    padding: 0.25rem 0.6rem;
    border-radius: 999px;
    text-transform: uppercase;
    font-weight: 700;
    display: inline-block;
    width: fit-content;
}

.status-badge.active {
    background: #dcfce7;
    color: #166534;
}

.status-badge.paused {
    background: #fef9c3;
    color: #854d0e;
}

.status-badge.cancelled {
    background: #fee2e2;
    color: #991b1b;
}

.amount-display {
    font-size: 1.5rem;
    font-weight: 800;
    color: var(--text-main);
    margin-bottom: 1rem;
    display: flex;
    align-items: baseline;
    gap: 0.25rem;
}

.cycle {
    font-size: 0.875rem;
    color: var(--text-muted);
    font-weight: 500;
}

.sub-details {
    margin-bottom: auto;
    /* Pushes actions to bottom */
    padding-bottom: 1.5rem;
}

.sub-details p {
    margin: 0.25rem 0;
    color: var(--text-muted);
    font-size: 0.875rem;
}

.sub-actions {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 0.5rem;
    margin-top: 1rem;
}

.sub-actions button {
    width: 100%;
}

.btn-pause {
    background-color: var(--warning-color);
    color: white;
}

.btn-activate {
    background-color: var(--secondary-color);
    color: white;
}

.btn-cancel {
    background-color: var(--danger-color);
    color: white;
}

.btn-disabled {
    background-color: #e5e7eb;
    color: #9ca3af;
    cursor: not-allowed;
}

.state,
.empty-state {
    text-align: center;
    padding: 4rem;
    color: var(--text-muted);
    background: var(--bg-surface);
    border-radius: var(--radius-lg);
}
</style>

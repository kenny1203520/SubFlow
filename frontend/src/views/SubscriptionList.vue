<script setup lang="ts">
import { ref, onMounted } from 'vue';
import { socket } from '../socket';

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

onMounted(() => {
    if (socket.connected) {
        fetchData();
    } else {
        socket.once('connect', fetchData);
    }
});
</script>

<template>
  <div class="subscriptions-view">
    <div class="header">
        <h1>My Subscriptions</h1>
    </div>

    <div v-if="loading" class="state">Loading subscriptions...</div>
    
    <div v-else-if="subscriptions.length === 0" class="empty-state">
        <p>No active subscriptions found in your groups.</p>
    </div>

    <div v-else class="subscription-grid">
        <div v-for="sub in subscriptions" :key="sub.id" class="sub-card">
            <div class="sub-header">
                <h3>{{ sub.name }}</h3>
                <span class="amount">${{ sub.amount }}</span>
            </div>
            <div class="sub-details">
                <p>Cycle: <strong>{{ sub.billing_cycle }}</strong></p>
                <p>Next payment: <strong>{{ new Date(sub.start_date).toLocaleDateString() }}</strong></p>
            </div>
        </div>
    </div>
  </div>
</template>

<style scoped>
.subscriptions-view {
    padding: 2rem;
}
.header {
    margin-bottom: 2rem;
}
.subscription-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
    gap: 1.5rem;
}
.sub-card {
    background: white;
    padding: 1.5rem;
    border-radius: 8px;
    box-shadow: 0 2px 4px rgba(0,0,0,0.1);
}
.sub-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 1rem;
}
.sub-header h3 {
    margin: 0;
}
.amount {
    font-size: 1.25rem;
    font-weight: bold;
    color: #2c3e50;
}
.sub-details p {
    margin: 0.25rem 0;
    color: #666;
}
.state, .empty-state {
    text-align: center;
    padding: 4rem;
    color: #666;
}
</style>

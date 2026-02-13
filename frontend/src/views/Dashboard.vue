<script setup lang="ts">
import { ref, onMounted } from 'vue';
import http from '../http';
import { useRouter } from 'vue-router';
import { useAuthStore } from '../stores/auth';
import { socket } from '../socket';

const router = useRouter();
const authStore = useAuthStore();
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

const logout = async () => {
    await http.post('/auth/signout');
    authStore.clearUser();
    router.push('/auth');
};
</script>

<template>
    <div class="dashboard-container">
        <aside class="sidebar">
            <h2>SubFlow</h2>
            <nav>
                <router-link to="/dashboard" class="nav-item">Dashboard</router-link>
                <router-link to="/groups" class="nav-item">Groups</router-link>
                <router-link to="/subscriptions" class="nav-item">Subscriptions</router-link>
            </nav>
            <div class="user-info" v-if="authStore.user">
                <p>{{ (authStore.user as any).username }}</p>
                <button @click="logout" class="logout-btn">Logout</button>
            </div>
        </aside>
        <main class="content">
            <header>
                <h1>Dashboard</h1>
            </header>
            <div class="stats-grid">
                <div class="stat-card">
                    <h3>Total Owed</h3>
                    <p class="amount i-owe">${{ stats.totalIOwe.toFixed(2) }}</p>
                </div>
                <div class="stat-card">
                    <h3>Total Owed to You</h3>
                    <p class="amount owed-to-me">${{ stats.totalOwedToMe.toFixed(2) }}</p>
                </div>
                <div class="stat-card">
                    <h3>Active Subscriptions</h3>
                    <p class="amount">{{ stats.activeSubscriptions }}</p>
                </div>
            </div>
            <!-- Recent Activity Placeholder -->
            <section class="activity-section">
                <h3>Recent Activity</h3>
                <p>No recent activity.</p>
            </section>
        </main>
    </div>
</template>

<style scoped>
.dashboard-container {
    display: flex;
    height: 100vh;
}

.sidebar {
    width: 250px;
    background-color: #2c3e50;
    color: white;
    padding: 1rem;
    display: flex;
    flex-direction: column;
}

.sidebar h2 {
    margin-bottom: 2rem;
    text-align: center;
}

.nav-item {
    display: block;
    padding: 1rem;
    color: #ecf0f1;
    text-decoration: none;
    border-radius: 4px;
    margin-bottom: 0.5rem;
}

.nav-item:hover,
.router-link-active {
    background-color: #34495e;
}

.user-info {
    margin-top: auto;
    padding-top: 1rem;
    border-top: 1px solid #34495e;
}

.logout-btn {
    background: transparent;
    border: 1px solid #e74c3c;
    color: #e74c3c;
    padding: 0.5rem;
    width: 100%;
    margin-top: 0.5rem;
    cursor: pointer;
}

.content {
    flex: 1;
    padding: 2rem;
    background-color: #f9f9f9;
    overflow-y: auto;
}

.stats-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
    gap: 1.5rem;
    margin-bottom: 2rem;
}

.stat-card {
    background: white;
    padding: 1.5rem;
    border-radius: 8px;
    box-shadow: 0 2px 4px rgba(0, 0, 0, 0.05);
}

.amount {
    font-size: 1.5rem;
    font-weight: bold;
    color: #2c3e50;
    margin-top: 0.5rem;
}

.amount.i-owe {
    color: #e74c3c;
}

.amount.owed-to-me {
    color: #27ae60;
}
</style>

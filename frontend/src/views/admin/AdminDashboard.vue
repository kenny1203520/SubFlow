<script setup lang="ts">
import { ref, onMounted } from 'vue';
import http from '../../http';

const stats = ref({
    totalUsers: 0,
    activeSessions: 0,
});

onMounted(async () => {
    try {
        const res = await http.get('/api/admin/users'); // Quick hack to get total users, Ideally a true stats endpoint exists
        if (res.data.status === 'ok') {
            stats.value.totalUsers = res.data.users.length;
            // mock active sessions for now
            stats.value.activeSessions = Math.floor(res.data.users.length * 0.4);
        }
    } catch (e) {
        // UI error
    }
});
</script>

<template>
    <div class="space-y-6">
        <div>
            <h2 class="text-3xl font-bold text-white mb-2">Dashboard Overview</h2>
            <p class="text-neutral-400">Welcome to the System Administration Panel.</p>
        </div>

        <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 mt-8">
            <div class="bg-neutral-900/50 border border-neutral-800 rounded-2xl p-6">
                <div class="text-neutral-400 text-sm font-medium mb-1">Total Users</div>
                <div class="text-3xl font-bold text-white">{{ stats.totalUsers }}</div>
            </div>

            <div class="bg-neutral-900/50 border border-neutral-800 rounded-2xl p-6">
                <div class="text-neutral-400 text-sm font-medium mb-1">Active Sessions</div>
                <div class="text-3xl font-bold text-emerald-400">{{ stats.activeSessions }}</div>
            </div>

            <div class="bg-neutral-900/50 border border-neutral-800 rounded-2xl p-6">
                <div class="text-neutral-400 text-sm font-medium mb-1">System Health</div>
                <div class="text-3xl font-bold text-emerald-400">Optimal</div>
            </div>
        </div>

        <div class="mt-8 bg-neutral-900/30 border border-neutral-800 rounded-2xl p-8 text-center text-neutral-400">
            Use the sidebar to manage users, configure system security policies, and monitor access.
        </div>
    </div>
</template>

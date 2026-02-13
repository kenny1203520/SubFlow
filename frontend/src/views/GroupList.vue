<script setup lang="ts">
import { ref, onMounted } from 'vue';
import { socket } from '../socket';
import CreateGroupModal from '../components/CreateGroupModal.vue';
import MainLayout from './MainLayout.vue';

const groups = ref<any[]>([]);
const showCreateModal = ref(false);

const fetchGroups = () => {
    socket.emit('group:list', (res: any) => {
        if (res.status === 'ok') {
            groups.value = res.groups;
        }
    });
};

onMounted(() => {
    if (socket.connected) {
        fetchGroups();
    } else {
        socket.once('connect', fetchGroups);
    }
});

const handleGroupCreated = () => {
    showCreateModal.value = false;
    fetchGroups();
};
</script>

<template>
    <MainLayout>
        <div class="header">
            <h1>My Groups</h1>
            <button @click="showCreateModal = true" class="btn btn-primary">+ Create Group</button>
        </div>

        <div v-if="groups.length === 0" class="empty-state">
            <p>You are not in any groups yet.</p>
            <button @click="showCreateModal = true" class="btn btn-primary mt-4">Create your first group</button>
        </div>

        <div v-else class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
            <div v-for="group in groups" :key="group.id" class="group-card card"
                @click="$router.push(`/groups/${group.id}`)">
                <h3>{{ group.name }}</h3>
                <p class="date">Created on: {{ new Date(group.created_at).toLocaleDateString() }}</p>
                <div class="hover-indicator">View Details →</div>
            </div>
        </div>

        <CreateGroupModal v-if="showCreateModal" @close="showCreateModal = false" @created="handleGroupCreated" />
    </MainLayout>
</template>

<style scoped>
.header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 2rem;
}

.header h1 {
    font-size: 2rem;
    font-weight: 700;
    color: var(--text-main);
    margin: 0;
}

.group-card {
    cursor: pointer;
    transition: transform 0.2s, box-shadow 0.2s, border-color 0.2s;
    height: 100%;
    display: flex;
    flex-direction: column;
}

.group-card:hover {
    transform: translateY(-4px);
    box-shadow: var(--shadow-md);
    border-color: var(--primary-color);
}

.group-card h3 {
    margin-top: 0;
    color: var(--text-main);
    font-size: 1.25rem;
}

.date {
    color: var(--text-muted);
    font-size: 0.875rem;
    margin-bottom: 1rem;
}

.hover-indicator {
    margin-top: auto;
    color: var(--primary-color);
    font-weight: 500;
    font-size: 0.875rem;
    opacity: 0;
    transform: translateX(-10px);
    transition: all 0.2s;
}

.group-card:hover .hover-indicator {
    opacity: 1;
    transform: translateX(0);
}

.empty-state {
    text-align: center;
    margin-top: 4rem;
    color: var(--text-muted);
    background: var(--bg-surface);
    padding: 3rem;
    border-radius: var(--radius-lg);
    border: 1px dashed #ccc;
}

.mt-4 {
    margin-top: 1rem;
}
</style>

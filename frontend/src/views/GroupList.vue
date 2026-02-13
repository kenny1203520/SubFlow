<script setup lang="ts">
import { ref, onMounted } from 'vue';
import { socket } from '../socket';
import CreateGroupModal from '../components/CreateGroupModal.vue';
import MainLayout from './MainLayout.vue';
import { useI18n } from 'vue-i18n';

const { t } = useI18n();
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
            <h1>{{ t('groups.title') }}</h1>
            <button @click="showCreateModal = true" class="btn btn-primary">+ {{ t('groups.createGroup') }}</button>
        </div>

        <div v-if="groups.length === 0" class="empty-state">
            <p>{{ t('groups.noGroups') }}</p>
            <button @click="showCreateModal = true" class="btn btn-primary mt-4">{{ t('groups.createFirstGroup')
                }}</button>
        </div>

        <div v-else class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
            <div v-for="group in groups" :key="group.id" class="group-card card"
                @click="$router.push(`/groups/${group.id}`)">
                <h3>{{ group.name }}</h3>
                <p class="date">{{ t('groups.createdOn', { date: new Date(group.created_at).toLocaleDateString() }) }}
                </p>
                <div class="hover-indicator">
                    {{ t('groups.viewDetails') }}
                    <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4 inline-block ml-1" fill="none"
                        viewBox="0 0 24 24" stroke="currentColor">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                            d="M14 5l7 7m0 0l-7 7m7-7H3" />
                    </svg>
                </div>
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

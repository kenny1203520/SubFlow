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
        <div class="flex flex-col gap-8 animate-fade-in">
            <div
                class="flex justify-between items-center bg-white/40 sticky top-0 z-10 backdrop-blur-md p-4 rounded-2xl border border-white/50 shadow-sm">
                <div>
                    <h1 class="text-2xl font-bold text-slate-800">{{ t('groups.title') }}</h1>
                    <p class="text-slate-500 text-sm">{{ t('groups.subtitle') }}</p>
                </div>
                <button @click="showCreateModal = true"
                    class="btn btn-primary shadow-lg hover:shadow-xl transform hover:-translate-y-1 transition-all">
                    <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" fill="none" viewBox="0 0 24 24"
                        stroke="currentColor">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
                    </svg>
                    {{ t('groups.createGroup') }}
                </button>
            </div>

            <div v-if="groups.length === 0"
                class="flex flex-col items-center justify-center p-16 glass-panel border-dashed border-2 border-slate-300">
                <div class="w-16 h-16 bg-slate-100 rounded-full flex items-center justify-center mb-4 text-slate-400">
                    <svg xmlns="http://www.w3.org/2000/svg" class="h-8 w-8" fill="none" viewBox="0 0 24 24"
                        stroke="currentColor">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                            d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z" />
                    </svg>
                </div>
                <p class="text-slate-500 text-lg mb-4">{{ t('groups.noGroups') }}</p>
                <button @click="showCreateModal = true" class="btn btn-primary">
                    {{ t('groups.createFirstGroup') }}
                </button>
            </div>

            <div v-else class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-6">
                <div v-for="group in groups" :key="group.id"
                    class="glass-card group cursor-pointer p-6 flex flex-col h-full relative overflow-hidden transition-all duration-300 hover:border-primary-300"
                    @click="$router.push(`/groups/${group.id}`)">

                    <div class="absolute top-0 right-0 p-4 opacity-10 group-hover:opacity-20 transition-opacity">
                        <svg xmlns="http://www.w3.org/2000/svg" class="h-24 w-24 text-primary-500" fill="currentColor"
                            viewBox="0 0 24 24">
                            <path
                                d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z" />
                        </svg>
                    </div>

                    <div class="mb-4">
                        <div v-if="group.icon_url"
                            class="w-12 h-12 rounded-xl bg-white border border-slate-100 mb-3 shadow-sm overflow-hidden group-hover:scale-110 transition-transform">
                            <img :src="group.icon_url" alt="Icon" class="w-full h-full object-cover" />
                        </div>
                        <div v-else
                            class="w-12 h-12 rounded-xl bg-primary-100 text-primary-600 flex items-center justify-center font-bold text-xl mb-3 shadow-sm group-hover:scale-110 transition-transform">
                            {{ group.name[0].toUpperCase() }}
                        </div>
                        <h3
                            class="text-xl font-bold text-slate-800 line-clamp-1 group-hover:text-primary-600 transition-colors">
                            {{ group.name }}</h3>
                        <p class="text-slate-500 text-xs mt-1">
                            {{ t('groups.createdOn', { date: new Date(group.created_at).toLocaleDateString() }) }}
                        </p>
                    </div>

                    <div
                        class="mt-auto pt-4 flex items-center text-sm font-semibold text-primary-600 opacity-0 transform translate-x-[-10px] group-hover:opacity-100 group-hover:translate-x-0 transition-all">
                        {{ t('groups.viewDetails') }}
                        <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4 ml-1" fill="none" viewBox="0 0 24 24"
                            stroke="currentColor">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                                d="M14 5l7 7m0 0l-7 7m7-7H3" />
                        </svg>
                    </div>
                </div>
            </div>

            <CreateGroupModal v-if="showCreateModal" @close="showCreateModal = false" @created="handleGroupCreated" />
        </div>
    </MainLayout>
</template>

<style scoped>
/* Scoped styles minimal, relying on global utility classes */
</style>

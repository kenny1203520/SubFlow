<script setup lang="ts">
import { ref, onMounted } from 'vue';
import { socket } from '../socket';
import { useI18n } from 'vue-i18n';
import MainLayout from './MainLayout.vue';

const { t } = useI18n();
const services = ref<any[]>([]);
const loading = ref(true);
const searchQuery = ref('');
const showModal = ref(false);
const isSubmitting = ref(false);
const editingService = ref<any | null>(null);

const form = ref({
    name: '',
    domain: '',
    icon_url: ''
});

// Helper for icon URL
const getIconUrl = (service: any) => {
    if (service.icon_url) return service.icon_url;
    if (service.domain) return `https://unavatar.io/${service.domain}`;
    return null;
};

const fetchServices = () => {
    loading.value = true;

    // Safety timeout
    const timeout = setTimeout(() => {
        if (loading.value) {
            loading.value = false;
            // distinct error state could be added here
        }
    }, 5000);

    socket.emit('service:list', (res: any) => {
        clearTimeout(timeout);
        loading.value = false;
        if (res.status === 'ok') {
            services.value = res.data.services;
        }
    });
};

const openCreateModal = () => {
    editingService.value = null;
    form.value = { name: '', domain: '', icon_url: '' };
    showModal.value = true;
};

const openEditModal = (service: any) => {
    editingService.value = service;
    form.value = { ...service };
    showModal.value = true;
};

const deleteService = (id: string) => {
    if (!confirm(t('common.actions.deleteConfirm', 'Are you sure you want to delete this service?'))) return;

    socket.emit('service:delete', { id }, (res: any) => {
        if (res.status === 'ok') {
            fetchServices();
        } else {
            alert(res.message || 'Failed to delete service');
        }
    });
};

const handleSubmit = () => {
    if (!form.value.name.trim()) return;

    isSubmitting.value = true;
    const event = editingService.value ? 'service:update' : 'service:create';
    const payload = editingService.value ? { ...form.value, id: editingService.value.id } : form.value;

    socket.emit(event, payload, (res: any) => {
        isSubmitting.value = false;
        if (res.status === 'ok') {
            showModal.value = false;
            fetchServices();
        } else {
            alert(res.message || 'Operation failed');
        }
    });
};

onMounted(() => {
    fetchServices();
});
</script>

<template>
    <MainLayout>
        <div class="flex flex-col gap-8 animate-fade-in relative z-10">
            <header class="flex justify-between items-end">
                <div>
                    <h1 class="text-3xl font-extrabold text-slate-800">{{ t('services.title', 'Services Management') }}
                    </h1>
                    <p class="text-slate-500 text-sm mt-1">
                        {{ t('services.description', 'Manage available subscription services') }}
                    </p>
                </div>
                <button @click="openCreateModal" class="btn btn-primary">
                    <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5 mr-2" viewBox="0 0 20 20"
                        fill="currentColor">
                        <path fill-rule="evenodd"
                            d="M10 3a1 1 0 011 1v5h5a1 1 0 110 2h-5v5a1 1 0 11-2 0v-5H4a1 1 0 110-2h5V4a1 1 0 011-1z"
                            clip-rule="evenodd" />
                    </svg>
                    {{ t('common.actions.add', 'Add Service') }}
                </button>
            </header>

            <div class="glass-panel p-4">
                <input v-model="searchQuery" type="text" :placeholder="t('common.actions.search', 'Search services...')"
                    class="glass-input w-full" />
            </div>

            <div v-if="loading" class="flex justify-center p-12">
                <div class="animate-spin rounded-full h-12 w-12 border-b-2 border-primary-600"></div>
            </div>

            <div v-else class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
                <div v-for="service in services.filter(s => s.name.toLowerCase().includes(searchQuery.toLowerCase()))"
                    :key="service.id"
                    class="glass-panel p-4 flex flex-col gap-4 group hover:ring-2 ring-primary-200 transition-all cursor-pointer relative"
                    @click="openEditModal(service)">
                    <div class="flex items-center gap-4">
                        <img v-if="getIconUrl(service)" :src="getIconUrl(service)"
                            @error="(e: any) => e.target.style.display = 'none'"
                            class="w-12 h-12 rounded-xl object-cover bg-white shadow-sm" />
                        <div v-else
                            class="w-12 h-12 rounded-xl bg-slate-100 flex items-center justify-center text-slate-400 font-bold text-xl">
                            {{ service.name.charAt(0) }}
                        </div>
                        <div class="flex-1 min-w-0">
                            <h3 class="font-bold text-slate-800 truncate">{{ service.name }}</h3>
                            <p class="text-xs text-slate-500 truncate">{{ service.domain }}</p>
                        </div>
                    </div>
                </div>
            </div>

            <!-- Modal -->
            <Teleport to="body">
                <div v-if="showModal"
                    class="fixed inset-0 bg-black/50 backdrop-blur-sm z-50 flex items-center justify-center p-4 animate-fade-in"
                    @click.self="showModal = false">
                    <div
                        class="bg-white rounded-2xl shadow-2xl w-full max-w-md p-6 flex flex-col gap-6 transform transition-all scale-100">
                        <h2 class="text-xl font-bold text-slate-800">{{ editingService ? t('services.editService',
                            'EditService') :
                            t('services.newService', 'New Service') }}
                        </h2>

                        <form @submit.prevent="handleSubmit" class="space-y-4">
                            <div>
                                <label class="block text-sm font-medium text-slate-700 mb-1">{{ t('common.fields.name',
                                    'Name') }}</label>
                                <input v-model="form.name" type="text" class="glass-input w-full" required />
                            </div>
                            <div>
                                <label class="block text-sm font-medium text-slate-700 mb-1">{{ t('groups.website',
                                    'Domain') }} ({{ t('common.optional', 'Optional') }})</label>
                                <input v-model="form.domain" type="text" class="glass-input w-full"
                                    placeholder="netflix.com" />
                                <p class="text-xs text-slate-400 mt-1">{{ t('services.domainHint',
                                    'Used to fetch icon automatically') }}</p>
                            </div>
                            <div>
                                <label class="block text-sm font-medium text-slate-700 mb-1">{{ t('groups.iconUrl',
                                    'Icon URL') }} ({{ t('common.optional', 'Optional') }})</label>
                                <input v-model="form.icon_url" type="url" class="glass-input w-full" />
                            </div>

                            <div class="flex gap-3 pt-2">
                                <button type="button" @click="showModal = false"
                                    class="btn bg-slate-100 text-slate-600 hover:bg-slate-200 flex-1">{{
                                        t('common.actions.cancel', 'Cancel') }}</button>
                                <button v-if="editingService" type="button" @click="deleteService(editingService.id)"
                                    class="btn bg-red-50 text-red-600 hover:bg-red-100">{{ t('common.actions.delete',
                                        'Delete') }}</button>
                                <button type="submit" :disabled="isSubmitting" class="btn btn-primary flex-1">
                                    {{ isSubmitting ? t('common.status.saving', 'Saving...') : t('common.actions.save',
                                        'Save') }}
                                </button>
                            </div>
                        </form>
                    </div>
                </div>
            </Teleport>
        </div>
    </MainLayout>
</template>

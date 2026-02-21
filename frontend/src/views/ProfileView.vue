<script setup lang="ts">
import { ref, onMounted, computed } from 'vue';
import { useI18n } from 'vue-i18n';
import http from '../http';
import { useAuthStore } from '../stores/auth';
import MainLayout from './MainLayout.vue';
import { useUIStore } from '../stores/ui';

const { t } = useI18n();
const authStore = useAuthStore();
const ui = useUIStore();

const isEditing = ref(false);
const loading = ref(false);
const uploadLoading = ref(false);
const fileInput = ref<HTMLInputElement | null>(null);

interface UserProfile {
    first_name: string;
    middle_name: string;
    last_name: string;
    nickname: string;
    birthday: string;
    phone: string;
    id_number: string;
    passport_number: string;
    address: string;
    avatar_url: string;
}

const profile = ref<UserProfile>({
    first_name: '',
    middle_name: '',
    last_name: '',
    nickname: '',
    birthday: '',
    phone: '',
    id_number: '',
    passport_number: '',
    address: '',
    avatar_url: ''
});

// Backup for cancel functionality
const originalProfile = ref({ ...profile.value });

const fullName = computed(() => {
    const parts = [profile.value.first_name, profile.value.middle_name, profile.value.last_name];
    return parts.filter(Boolean).join(' ') || authStore.user?.username || 'User';
});

const fetchProfile = async () => {
    try {
        const res = await http.get('/api/user/profile');
        if (res.data.status === 'ok') {
            const p = res.data.profile;
            profile.value = {
                first_name: p.first_name || '',
                middle_name: p.middle_name || '',
                last_name: p.last_name || '',
                nickname: p.nickname || '',
                birthday: p.birthday ? new Date(p.birthday).toISOString().slice(0, 10) : '',
                phone: p.phone || '',
                id_number: p.id_number || '',
                passport_number: p.passport_number || '',
                address: p.address || '',
                avatar_url: p.avatar_url || ''
            };
            originalProfile.value = { ...profile.value };
        }
    } catch (err) {
        console.error('Failed to fetch profile', err);
    }
};

const handleAvatarClick = () => {
    if (isEditing.value) {
        fileInput.value?.click();
    }
};

const handleFileChange = async (event: Event) => {
    const target = event.target as HTMLInputElement;
    if (!target.files?.length) return;

    const file = target.files?.[0];
    if (!file) return;

    const formData = new FormData();
    formData.append('avatar', file);

    uploadLoading.value = true;
    try {
        const res = await http.post('/api/files/avatar', formData, {
            headers: { 'Content-Type': 'multipart/form-data' }
        });

        const avatarUrl = res.data.url;

        // Update user profile in DB immediately for avatar
        await http.patch('/api/user/profile', { avatar_url: avatarUrl });

        profile.value.avatar_url = avatarUrl;
        if (authStore.user) {
            authStore.user.avatar_url = avatarUrl;
        }
    } catch (err) {
        console.error(err);
        ui.alert(t('profile.messages.uploadFailed', 'Upload failed'));
    } finally {
        uploadLoading.value = false;
    }
};

const toggleEdit = () => {
    isEditing.value = true;
};

const cancelEdit = () => {
    profile.value = { ...originalProfile.value };
    isEditing.value = false;
};

const saveProfile = async () => {
    loading.value = true;
    try {
        await http.patch('/api/user/profile', profile.value);

        originalProfile.value = { ...profile.value };
        isEditing.value = false;

        // Update basic info in auth store if needed
        if (authStore.user) {
            authStore.user.avatar_url = profile.value.avatar_url;
        }

        ui.alert(t('profile.messages.updateSuccess', 'Profile updated successfully'));
    } catch (err) {
        ui.alert(t('profile.messages.updateFailed', 'Update failed'));
    } finally {
        loading.value = false;
    }
};

onMounted(fetchProfile);
</script>

<template>
    <MainLayout>
        <div class="profile-container max-w-4xl mx-auto animate-fade-in pb-12">
            <!-- Header -->
            <header class="flex justify-between items-center py-8 mb-4">
                <h1 class="page-title text-4xl font-extrabold text-slate-800">{{ t('profile.title', 'Profile') }}</h1>
                <button v-if="!isEditing" @click="toggleEdit" class="btn btn-primary px-6">
                    <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5 mr-2" fill="none" viewBox="0 0 24 24"
                        stroke="currentColor">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                            d="M15.232 5.232l3.536 3.536m-2.036-5.036a2.5 2.5 0 113.536 3.536L6.5 21.036H3v-3.572L16.732 3.732z" />
                    </svg>
                    {{ t('profile.actions.edit', 'Edit Profile') }}
                </button>
            </header>

            <div class="flex flex-col lg:flex-row gap-8">
                <!-- Left Column: Avatar & Quick Info -->
                <div class="lg:w-1/3 space-y-6">
                    <div class="glass-panel p-8 flex flex-col items-center text-center">
                        <div class="avatar-wrapper group relative mb-4" :class="{ 'cursor-pointer': isEditing }"
                            @click="handleAvatarClick">
                            <img :src="profile.avatar_url || 'https://www.gravatar.com/avatar/00000000000000000000000000000000?d=mp&f=y'"
                                class="avatar-image glass-card" alt="User Avatar">

                            <!-- Overlay only in edit mode -->
                            <div v-if="isEditing"
                                class="avatar-overlay glass-header opacity-0 group-hover:opacity-100 transition-opacity flex items-center justify-center">
                                <span class="text-xs font-bold">
                                    {{ uploadLoading ? '...' : t('profile.actions.changeAvatar', 'Change Avatar') }}
                                </span>
                            </div>
                            <input type="file" ref="fileInput" class="hidden" accept="image/*"
                                @change="handleFileChange">
                        </div>

                        <h2 class="text-xl font-bold text-slate-800">{{ fullName }}</h2>
                        <p class="text-slate-500 text-sm">@{{ authStore.user?.username }}</p>

                        <div class="mt-6 w-full pt-6 border-t border-slate-100">
                            <div class="flex justify-between items-center py-2">
                                <span class="text-slate-500 text-sm">{{ t('profile.fields.email') }}</span>
                            </div>
                            <div class="font-medium text-slate-700 break-all">{{ authStore.user?.email }}</div>
                        </div>
                    </div>
                </div>

                <!-- Right Column: Details Form -->
                <div class="lg:w-2/3">
                    <form @submit.prevent="saveProfile" class="space-y-8">

                        <!-- Basic Information -->
                        <section class="glass-panel p-6">
                            <h3 class="text-lg font-bold text-slate-800 mb-4 pb-2 border-b border-slate-100">
                                {{ t('profile.sections.basic', 'Basic Information') }}
                            </h3>
                            <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
                                <div class="form-group">
                                    <label class="form-label">{{ t('profile.fields.lastName') }}</label>
                                    <input v-if="isEditing" type="text" v-model="profile.last_name"
                                        class="glass-input w-full">
                                    <div v-else class="read-only-field">{{ profile.last_name || '-' }}</div>
                                </div>
                                <div class="form-group">
                                    <label class="form-label">{{ t('profile.fields.firstName') }}</label>
                                    <input v-if="isEditing" type="text" v-model="profile.first_name"
                                        class="glass-input w-full">
                                    <div v-else class="read-only-field">{{ profile.first_name || '-' }}</div>
                                </div>
                                <div class="form-group">
                                    <label class="form-label">{{ t('profile.fields.middleName') }}</label>
                                    <input v-if="isEditing" type="text" v-model="profile.middle_name"
                                        class="glass-input w-full">
                                    <div v-else class="read-only-field">{{ profile.middle_name || '-' }}</div>
                                </div>
                                <div class="form-group">
                                    <label class="form-label">{{ t('profile.fields.nickname') }}</label>
                                    <input v-if="isEditing" type="text" v-model="profile.nickname"
                                        class="glass-input w-full">
                                    <div v-else class="read-only-field">{{ profile.nickname || '-' }}</div>
                                </div>
                                <div class="form-group">
                                    <label class="form-label">{{ t('profile.fields.birthday') }}</label>
                                    <input v-if="isEditing" type="date" v-model="profile.birthday"
                                        class="glass-input w-full">
                                    <div v-else class="read-only-field">{{ profile.birthday || '-' }}</div>
                                </div>
                            </div>
                        </section>

                        <!-- Contact Information -->
                        <section class="glass-panel p-6">
                            <h3 class="text-lg font-bold text-slate-800 mb-4 pb-2 border-b border-slate-100">
                                {{ t('profile.sections.contact', 'Contact Information') }}
                            </h3>
                            <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
                                <div class="form-group">
                                    <label class="form-label">{{ t('profile.fields.mobilePhone') }}</label>
                                    <input v-if="isEditing" type="tel" v-model="profile.phone"
                                        class="glass-input w-full">
                                    <div v-else class="read-only-field">{{ profile.phone || '-' }}</div>
                                </div>
                                <div class="form-group md:col-span-2">
                                    <label class="form-label">{{ t('profile.fields.address') }}</label>
                                    <input v-if="isEditing" type="text" v-model="profile.address"
                                        class="glass-input w-full">
                                    <div v-else class="read-only-field">{{ profile.address || '-' }}</div>
                                </div>
                            </div>
                        </section>

                        <!-- Identity Documents -->
                        <section class="glass-panel p-6">
                            <h3 class="text-lg font-bold text-slate-800 mb-4 pb-2 border-b border-slate-100">
                                {{ t('profile.sections.identity', 'Identity Documents') }}
                            </h3>
                            <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
                                <div class="form-group">
                                    <label class="form-label">{{ t('profile.fields.idNumber') }}</label>
                                    <input v-if="isEditing" type="text" v-model="profile.id_number"
                                        class="glass-input w-full">
                                    <div v-else class="read-only-field">{{ profile.id_number || '-' }}</div>
                                </div>
                                <div class="form-group">
                                    <label class="form-label">{{ t('profile.fields.passportNumber') }}</label>
                                    <input v-if="isEditing" type="text" v-model="profile.passport_number"
                                        class="glass-input w-full">
                                    <div v-else class="read-only-field">{{ profile.passport_number || '-' }}</div>
                                </div>
                            </div>
                        </section>

                        <!-- Actions -->
                        <div v-if="isEditing" class="flex justify-end gap-4 pt-4">
                            <button type="button" @click="cancelEdit"
                                class="btn bg-slate-100 text-slate-600 hover:bg-slate-200 px-6">
                                {{ t('profile.actions.cancel', 'Cancel') }}
                            </button>
                            <button type="submit" class="btn btn-primary px-8" :disabled="loading">
                                {{ loading ? '...' : t('profile.actions.save', 'Save Changes') }}
                            </button>
                        </div>
                    </form>
                </div>
            </div>
        </div>
    </MainLayout>
</template>

<style scoped>
.avatar-wrapper {
    width: 150px;
    height: 150px;
    border-radius: 50%;
    overflow: hidden;
    border: 4px solid var(--primary-200);
    box-shadow: 0 8px 32px 0 rgba(31, 38, 135, 0.12);
    transition: all 0.3s ease;
}

.avatar-wrapper.cursor-pointer:hover {
    border-color: var(--primary-400);
    box-shadow: 0 8px 32px 0 rgba(99, 102, 241, 0.25);
}

.avatar-image {
    width: 100%;
    height: 100%;
    object-fit: cover;
}

.avatar-overlay {
    position: absolute;
    top: 0;
    left: 0;
    width: 100%;
    height: 100%;
    background: rgba(0, 0, 0, 0.4);
    display: flex;
    align-items: center;
    justify-content: center;
    color: white;
    opacity: 0;
    transition: opacity 0.3s ease;
}

.read-only-field {
    padding: 0.75rem 0 0.75rem 0;
    font-size: 1rem;
    color: #334155;
    border-bottom: 1px dashed #e2e8f0;
}

.form-label {
    display: block;
    font-size: 0.875rem;
    font-weight: 500;
    color: #64748b;
    margin-bottom: 0.5rem;
}
</style>
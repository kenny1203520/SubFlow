<script setup lang="ts">
import { ref, onMounted } from 'vue';
import { useI18n } from 'vue-i18n';
import http from '../http';
import { useAuthStore } from '../stores/auth';
import MainLayout from './MainLayout.vue';

const { t } = useI18n();
const authStore = useAuthStore();

const profile = ref<any>({
    real_name: '',
    birthday: '',
    phone: '',
    avatar_url: ''
});
const loading = ref(false);
const uploadLoading = ref(false);
const fileInput = ref<HTMLInputElement | null>(null);

const fetchProfile = async () => {
    try {
        const res = await http.get('/api/user/profile');
        if (res.data.status === 'ok') {
            const p = res.data.profile;
            profile.value = {
                real_name: p.real_name || '',
                birthday: p.birthday ? new Date(p.birthday).toISOString().split('T')[0] : '',
                phone: p.phone || '',
                avatar_url: p.avatar_url || ''
            };
        }
    } catch (err) { }
};

const handleAvatarClick = () => {
    fileInput.value?.click();
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

        // Update user profile in DB
        await http.patch('/api/user/profile', { avatar_url: avatarUrl });

        // Update local state
        profile.value.avatar_url = avatarUrl;
        if (authStore.user) {
            authStore.user.avatar_url = avatarUrl;
        }
    } catch (err) {
        console.error(err);
        alert('Upload failed');
    } finally {
        uploadLoading.value = false;
    }
};

const saveProfile = async () => {
    loading.value = true;
    try {
        await http.patch('/api/user/profile', {
            real_name: profile.value.real_name,
            birthday: profile.value.birthday,
            phone: profile.value.phone
        });
        alert(t('profile.updateSuccess', 'Profile updated successfully'));
    } catch (err) {
        alert(t('profile.updateFailed', 'Update failed'));
    } finally {
        loading.value = false;
    }
};

onMounted(fetchProfile);
</script>

<template>
    <MainLayout>
        <div class="profile-container max-w-2xl mx-auto animate-fade-in">
            <header class="page-header py-8">
                <h1 class="page-title text-4xl font-extrabold">{{ t('profile.profile') }}</h1>
            </header>

            <div class="glass-panel p-8 space-y-8">
                <!-- Avatar Section -->
                <div class="flex flex-col items-center">
                    <div class="avatar-wrapper group relative" @click="handleAvatarClick">
                        <img :src="profile.avatar_url || 'https://www.gravatar.com/avatar/00000000000000000000000000000000?d=mp&f=y'"
                            class="avatar-image glass-card" alt="User Avatar">
                        <div
                            class="avatar-overlay glass-header opacity-0 group-hover:opacity-100 transition-opacity flex items-center justify-center">
                            <span class="text-xs font-bold">
                                {{ uploadLoading ? '...' : t('profile.changeAvatar', 'Change Avatar') }}
                            </span>
                        </div>
                        <input type="file" ref="fileInput" class="hidden" accept="image/*" @change="handleFileChange">
                    </div>
                </div>

                <form @submit.prevent="saveProfile" class="space-y-6">
                    <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
                        <div class="form-group">
                            <label class="form-label">{{ t('profile.username') }}</label>
                            <input type="text" :value="authStore.user?.username" disabled
                                class="glass-input bg-white/5 cursor-not-allowed">
                        </div>

                        <div class="form-group">
                            <label class="form-label">{{ t('profile.email') }}</label>
                            <input type="email" :value="authStore.user?.email" disabled
                                class="glass-input bg-white/5 cursor-not-allowed">
                        </div>

                        <div class="form-group">
                            <label class="form-label">{{ t('profile.realName') }}</label>
                            <input type="text" v-model="profile.real_name" class="glass-input"
                                :placeholder="t('profile.realName')">
                        </div>

                        <div class="form-group">
                            <label class="form-label">{{ t('profile.birthday') }}</label>
                            <input type="date" v-model="profile.birthday" class="glass-input">
                        </div>

                        <div class="form-group">
                            <label class="form-label">{{ t('profile.phone') }}</label>
                            <input type="tel" v-model="profile.phone" class="glass-input"
                                :placeholder="t('profile.phone')">
                        </div>
                    </div>

                    <div class="pt-6">
                        <button type="submit" class="btn btn-primary w-full py-4 text-lg font-bold"
                            :disabled="loading || uploadLoading">
                            {{ loading ? '...' : t('profile.save') }}
                        </button>
                    </div>
                </form>
            </div>
        </div>
    </MainLayout>
</template>

<style scoped>
.max-w-2xl {
    max-width: 48rem;
}

.avatar-wrapper {
    width: 150px;
    height: 150px;
    border-radius: 50%;
    overflow: hidden;
    cursor: pointer;
    border: 4px solid rgba(255, 255, 255, 0.2);
    box-shadow: 0 8px 32px 0 rgba(31, 38, 135, 0.37);
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
    opacity: 0;
    transition: opacity 0.3s ease;
}

.avatar-wrapper:hover .avatar-overlay {
    opacity: 1;
}

.glass-input {
    width: 100%;
    background: rgba(255, 255, 255, 0.05);
    backdrop-filter: blur(4px);
    -webkit-backdrop-filter: blur(4px);
    border: 1px solid rgba(255, 255, 255, 0.1);
    border-radius: 12px;
    padding: 0.75rem 1rem;
    color: white;
    transition: all 0.3s ease;
}

.glass-input:focus {
    background: rgba(255, 255, 255, 0.1);
    border-color: rgba(255, 255, 255, 0.3);
    outline: none;
    box-shadow: 0 0 0 4px rgba(255, 255, 255, 0.05);
}

.form-label {
    display: block;
    font-size: 0.875rem;
    font-weight: 600;
    color: rgba(255, 255, 255, 0.7);
    margin-bottom: 0.5rem;
}
</style>
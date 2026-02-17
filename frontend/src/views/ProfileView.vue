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
    phone: ''
});
const loading = ref(false);

const fetchProfile = async () => {
    try {
        const res = await http.get('/auth/user'); // 這裡暫時共用此 API，或新增 profile API
        if (res.data) {
            // 實作上應從 profile 表讀取
        }
    } catch (err) {}
};

const saveProfile = async () => {
    loading.value = true;
    try {
        // 實作 API 呼叫
        alert(t('profile.updateSuccess'));
    } catch (err) {
        alert(t('profile.updateFailed'));
    } finally {
        loading.value = false;
    }
};

onMounted(fetchProfile);
</script>

<template>
    <MainLayout>
        <div class="profile-container max-w-2xl mx-auto">
            <header class="page-header">
                <h1 class="page-title">{{ t('profile.profile') }}</h1>
            </header>

            <form @submit.prevent="saveProfile" class="card space-y-4">
                <div class="form-group">
                    <label class="form-label">{{ t('profile.username') }}</label>
                    <input type="text" :value="authStore.user?.username" disabled class="form-input bg-muted">
                </div>

                <div class="form-group">
                    <label class="form-label">{{ t('profile.email') }}</label>
                    <input type="email" :value="authStore.user?.email" disabled class="form-input bg-muted">
                </div>

                <div class="form-group">
                    <label class="form-label">{{ t('profile.realName') }}</label>
                    <input type="text" v-model="profile.real_name" class="form-input">
                </div>

                <div class="form-group">
                    <label class="form-label">{{ t('profile.birthday') }}</label>
                    <input type="date" v-model="profile.birthday" class="form-input">
                </div>

                <div class="mt-6">
                    <button type="submit" class="btn btn-primary w-full" :disabled="loading">
                        {{ loading ? '...' : t('profile.save') }}
                    </button>
                </div>
            </form>
        </div>
    </MainLayout>
</template>

<style scoped>
.max-w-2xl { max-width: 42rem; }
.mx-auto { margin-left: auto; margin-right: auto; }
.space-y-4 > * + * { margin-top: 1rem; }
.bg-muted { background-color: var(--bg-surface-soft); opacity: 0.7; }
</style>

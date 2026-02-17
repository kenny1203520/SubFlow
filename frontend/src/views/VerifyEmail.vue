<script setup lang="ts">
import { ref, onMounted } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { useI18n } from 'vue-i18n';
import api from '../http';

const { t } = useI18n();
const route = useRoute();
const router = useRouter();
const token = route.params.token as string;

const status = ref<'loading' | 'success' | 'error'>('loading');
const message = ref('');

onMounted(async () => {
    try {
        const res = await api.get(`/auth/verify-email/${token}`);
        status.value = 'success';
        message.value = res.data.message;
        setTimeout(() => {
            router.push('/dashboard');
        }, 3000);
    } catch (err: any) {
        status.value = 'error';
        message.value = err.response?.data || 'Verification failed';
    }
});
</script>

<template>
    <div class="verify-page">
        <div class="verify-card glass-panel animate-fade-in-up">
            <h1 class="title">{{ t('auth.emailVerification', 'Email Verification') }}</h1>

            <div class="content">
                <div v-if="status === 'loading'" class="loading-state">
                    <div class="spinner"></div>
                    <p>{{ t('auth.verifying', 'Verifying your email...') }}</p>
                </div>

                <div v-else-if="status === 'success'" class="success-state">
                    <div class="icon-success">✓</div>
                    <p>{{ message }}</p>
                    <p class="hint">{{ t('auth.redirecting', 'Redirecting you to dashboard...') }}</p>
                </div>

                <div v-else class="error-state">
                    <div class="icon-error">!</div>
                    <p>{{ message }}</p>
                    <button @click="router.push('/auth')" class="btn btn-primary mt-6">
                        {{ t('auth.backToLogin') }}
                    </button>
                </div>
            </div>
        </div>
    </div>
</template>

<style scoped>
.verify-page {
    min-height: 100vh;
    display: flex;
    align-items: center;
    justify-content: center;
    background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
    padding: 1rem;
}

.verify-card {
    width: 100%;
    max-width: 450px;
    padding: 3rem;
    text-align: center;
    color: white;
}

.title {
    font-size: 2rem;
    font-weight: 800;
    margin-bottom: 2rem;
}

.content {
    min-height: 150px;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
}

.loading-state p {
    margin-top: 1rem;
    opacity: 0.8;
}

.spinner {
    width: 40px;
    height: 40px;
    border: 4px solid rgba(255, 255, 255, 0.3);
    border-top: 4px solid white;
    border-radius: 50%;
    animation: spin 1s linear infinite;
}

@keyframes spin {
    to {
        transform: rotate(360deg);
    }
}

.icon-success {
    font-size: 4rem;
    color: #10b981;
    margin-bottom: 1rem;
}

.icon-error {
    font-size: 4rem;
    color: #ef4444;
    margin-bottom: 1rem;
}

.hint {
    font-size: 0.875rem;
    opacity: 0.7;
    margin-top: 0.5rem;
}
</style>

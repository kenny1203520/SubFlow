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
    <div class="min-h-screen w-full flex items-center justify-center p-4 relative overflow-hidden bg-body">
        <!-- Dynamic Background -->
        <div class="absolute top-0 left-0 w-full h-full overflow-hidden -z-10">
            <div class="blob blob-1"></div>
            <div class="blob blob-2"></div>
        </div>

        <div class="verify-card glass-panel animate-fade-in-up">
            <h1 class="text-3xl font-extrabold text-slate-800 mb-6">{{ t('auth.emailVerification', 'Email Verification')
                }}</h1>

            <div class="content">
                <div v-if="status === 'loading'" class="loading-state">
                    <div class="spinner"></div>
                    <p class="text-slate-500 mt-4">{{ t('auth.verifying', 'Verifying your email...') }}</p>
                </div>

                <div v-else-if="status === 'success'" class="success-state">
                    <div class="text-6xl text-green-500 mb-4">✓</div>
                    <p class="text-lg font-semibold text-slate-800">{{ message }}</p>
                    <p class="text-sm text-slate-400 mt-2">{{ t('auth.redirecting', 'Redirecting you to dashboard...')
                        }}</p>
                </div>

                <div v-else class="error-state">
                    <div class="text-6xl text-red-500 mb-4">!</div>
                    <p class="text-lg font-semibold text-slate-800">{{ message }}</p>
                    <button @click="router.push('/auth')" class="btn btn-primary mt-8 w-full">
                        {{ t('auth.backToLogin') }}
                    </button>
                </div>
            </div>
        </div>
    </div>
</template>

<style scoped>
.verify-card {
    width: 100%;
    max-width: 450px;
    padding: 3rem;
    text-align: center;
}

.content {
    min-height: 150px;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
}

.spinner {
    width: 40px;
    height: 40px;
    border: 4px solid rgba(0, 0, 0, 0.1);
    border-top: 4px solid var(--primary-600);
    border-radius: 50%;
    animation: spin 1s linear infinite;
}

@keyframes spin {
    to {
        transform: rotate(360deg);
    }
}

.blob {
    position: absolute;
    border-radius: 50%;
    filter: blur(80px);
    opacity: 0.6;
    animation: float 10s infinite ease-in-out;
}

.blob-1 {
    top: -10%;
    right: -10%;
    width: 500px;
    height: 500px;
    background: var(--primary-300);
    animation-delay: 0s;
}

.blob-2 {
    bottom: -10%;
    left: -10%;
    width: 400px;
    height: 400px;
    background: var(--accent-color);
    animation-delay: -5s;
    opacity: 0.4;
}

@keyframes float {

    0%,
    100% {
        transform: translateY(0) scale(1);
    }

    50% {
        transform: translateY(-20px) scale(1.05);
    }
}
</style>

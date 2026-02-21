<script setup lang="ts">
import { ref } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRouter } from 'vue-router';
import api from '../http';

const { t } = useI18n();
const router = useRouter();
const email = ref('');
const loading = ref(false);
const message = ref('');
const error = ref('');

const handleSubmit = async () => {
    if (!email.value) return;
    loading.value = true;
    message.value = '';
    error.value = '';

    try {
        await api.post('/auth/password-reset', { email: email.value });
        message.value = t('auth.resetLinkSent', 'Reset link sent to your email.');
    } catch (err: any) {
        const msg = err.response?.data?.message || 'auth.errors.unknownError';
        error.value = t(msg, msg);
    } finally {
        loading.value = false;
    }
};
</script>

<template>
    <div class="min-h-screen w-full flex items-center justify-center p-4 relative overflow-hidden bg-body">
        <!-- Dynamic Background -->
        <div class="absolute top-0 left-0 w-full h-full overflow-hidden -z-10">
            <div class="blob blob-1"></div>
            <div class="blob blob-2 opacity-50"></div>
        </div>

        <div class="w-full max-w-md glass-panel p-8 shadow-2xl animate-fade-in-up">
            <h1 class="text-3xl font-extrabold text-center mb-2 text-gradient">
                {{ t('auth.forgotPassword', 'Forgot Password') }}
            </h1>
            <p class="text-center text-slate-500 mb-8">
                {{ t('auth.forgotSubtitle', 'Enter your email to receive a reset link') }}
            </p>

            <form @submit.prevent="handleSubmit" class="space-y-6">
                <div class="form-group">
                    <label class="form-label">{{ t('auth.email') }}</label>
                    <input v-model="email" type="email" :placeholder="t('auth.email')" required class="glass-input" />
                </div>

                <div v-if="message" class="p-3 rounded-lg bg-green-50 text-green-700 text-sm border border-green-200">
                    {{ message }}
                </div>
                <div v-if="error" class="p-3 rounded-lg bg-red-50 text-red-600 text-sm border border-red-200">
                    {{ error }}
                </div>

                <button type="submit" :disabled="loading" class="btn btn-primary w-full shadow-lg">
                    {{ loading ? t('common.status.loading') : t('auth.sendResetLink', 'Send Reset Link') }}
                </button>

                <div class="text-center mt-6">
                    <span @click="router.push('/auth')"
                        class="text-sm text-slate-500 hover:text-primary-600 cursor-pointer font-medium hover:underline transition-colors">
                        ← {{ t('auth.backToLogin', 'Back to Login') }}
                    </span>
                </div>
            </form>
        </div>
    </div>
</template>

<style scoped>
/* Blob Animations (Copied from Auth.vue for consistency) */
.blob {
    position: absolute;
    border-radius: 50%;
    filter: blur(80px);
    opacity: 0.5;
    animation: float 10s infinite ease-in-out;
}

.blob-1 {
    top: -10%;
    right: -10%;
    width: 400px;
    height: 400px;
    background: var(--primary-300);
}

.blob-2 {
    bottom: -10%;
    left: -10%;
    width: 300px;
    height: 300px;
    background: var(--accent-color);
    animation-delay: -5s;
}

@keyframes float {

    0%,
    100% {
        transform: translateY(0);
    }

    50% {
        transform: translateY(-20px);
    }
}
</style>

<script setup lang="ts">
import { ref } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRoute, useRouter } from 'vue-router';
import api from '../http';

const { t } = useI18n();
const route = useRoute();
const router = useRouter();
const token = route.params.token as string;

const password = ref('');
const confirmPassword = ref('');
const loading = ref(false);
const message = ref('');
const error = ref('');
const showPassword = ref(false);
const showConfirmPassword = ref(false);

const handleSubmit = async () => {
    if (password.value !== confirmPassword.value) {
        error.value = t('auth.passwordMismatch', 'Passwords do not match');
        return;
    }

    loading.value = true;
    message.value = '';
    error.value = '';

    try {
        await api.post(`/auth/password-reset/${token}`, { password: password.value });
        message.value = t('auth.passwordResetSuccess', 'Password has been reset successfully.');
        setTimeout(() => {
            router.push('/auth');
        }, 2000);
    } catch (err: any) {
        const msg = err.response?.data?.message || 'auth.errors.unknownError';
        error.value = t(msg, msg);
    } finally {
        loading.value = false;
    }
};
</script>

<template>
    <div class="auth-page">
        <div class="auth-card glass-panel animate-fade-in-up">
            <h1 class="auth-title">{{ t('auth.resetPassword', 'Reset Password') }}</h1>
            <p class="auth-subtitle">{{ t('auth.resetSubtitle', 'Enter your new password below') }}</p>

            <form @submit.prevent="handleSubmit" class="auth-form">
                <div class="form-group">
                    <label>{{ t('auth.newPassword', 'New Password') }}</label>
                    <div class="relative">
                        <input v-model="password" :type="showPassword ? 'text' : 'password'" required
                            class="glass-input pr-12" />
                        <button type="button" @click="showPassword = !showPassword"
                            class="absolute right-3 top-1/2 -translate-y-1/2 p-2 text-white/50 hover:text-white transition-colors">
                            <svg v-if="showPassword" xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" fill="none"
                                viewBox="0 0 24 24" stroke="currentColor">
                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                                    d="M13.875 18.825A10.05 10.05 0 0112 19c-4.478 0-8.268-2.943-9.543-7a9.97 9.97 0 011.563-3.029m5.858.908a3 3 0 114.243 4.243M9.878 9.878l4.242 4.242M9.88 9.88l-3.29-3.29m7.532 7.532l3.29 3.29M3 3l18 18" />
                            </svg>
                            <svg v-else xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" fill="none"
                                viewBox="0 0 24 24" stroke="currentColor">
                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                                    d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                                    d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
                            </svg>
                        </button>
                    </div>
                </div>

                <div class="form-group">
                    <label>{{ t('auth.confirmPassword', 'Confirm Password') }}</label>
                    <div class="relative">
                        <input v-model="confirmPassword" :type="showConfirmPassword ? 'text' : 'password'" required
                            class="glass-input pr-12" />
                        <button type="button" @click="showConfirmPassword = !showConfirmPassword"
                            class="absolute right-3 top-1/2 -translate-y-1/2 p-2 text-white/50 hover:text-white transition-colors">
                            <svg v-if="showConfirmPassword" xmlns="http://www.w3.org/2000/svg" class="h-5 w-5"
                                fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                                    d="M13.875 18.825A10.05 10.05 0 0112 19c-4.478 0-8.268-2.943-9.543-7a9.97 9.97 0 011.563-3.029m5.858.908a3 3 0 114.243 4.243M9.878 9.878l4.242 4.242M9.88 9.88l-3.29-3.29m7.532 7.532l3.29 3.29M3 3l18 18" />
                            </svg>
                            <svg v-else xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" fill="none"
                                viewBox="0 0 24 24" stroke="currentColor">
                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                                    d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                                    d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
                            </svg>
                        </button>
                    </div>
                </div>

                <div v-if="message" class="alert alert-success mt-4">
                    {{ message }}
                </div>
                <div v-if="error" class="alert alert-danger mt-4">
                    {{ error }}
                </div>

                <button type="submit" :disabled="loading" class="btn btn-primary w-full mt-6">
                    {{ loading ? t('common.status.loading') : t('auth.resetBtn', 'Reset Password') }}
                </button>
            </form>
        </div>
    </div>
</template>

<style scoped>
.auth-page {
    min-height: 100vh;
    display: flex;
    align-items: center;
    justify-content: center;
    background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
    padding: 1rem;
}

.auth-card {
    width: 100%;
    max-width: 450px;
    padding: 2.5rem;
    color: white;
}

.auth-title {
    font-size: 2rem;
    font-weight: 800;
    margin-bottom: 0.5rem;
    text-align: center;
}

.auth-subtitle {
    font-size: 1rem;
    color: rgba(255, 255, 255, 0.7);
    margin-bottom: 2rem;
    text-align: center;
}

.form-group {
    margin-bottom: 1.5rem;
}

.form-group label {
    display: block;
    font-size: 0.875rem;
    font-weight: 600;
    margin-bottom: 0.5rem;
}

.glass-input {
    width: 100%;
    background: rgba(255, 255, 255, 0.1);
    border: 1px solid rgba(255, 255, 255, 0.2);
    border-radius: var(--radius-lg);
    padding: 0.75rem 1rem;
    color: white;
    outline: none;
    transition: all 0.3s ease;
}

.glass-input:focus {
    background: rgba(255, 255, 255, 0.15);
    border-color: rgba(255, 255, 255, 0.4);
    box-shadow: 0 0 0 4px rgba(255, 255, 255, 0.05);
}

.auth-footer {
    text-align: center;
}
</style>

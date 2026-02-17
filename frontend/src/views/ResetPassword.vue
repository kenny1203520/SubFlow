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
        error.value = err.response?.data || 'Error resetting password';
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
                    <input v-model="password" type="password" required class="glass-input" />
                </div>

                <div class="form-group">
                    <label>{{ t('auth.confirmPassword', 'Confirm Password') }}</label>
                    <input v-model="confirmPassword" type="password" required class="glass-input" />
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

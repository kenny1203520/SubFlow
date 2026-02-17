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
        const res = await api.post('/auth/password-reset', { email: email.value });
        message.value = t('auth.resetLinkSent', 'Reset link sent to your email.');
    } catch (err: any) {
        error.value = err.response?.data || 'Error sending reset link';
    } finally {
        loading.value = false;
    }
};
</script>

<template>
    <div class="auth-page">
        <div class="auth-card glass-panel animate-fade-in-up">
            <h1 class="auth-title">{{ t('auth.forgotPassword', 'Forgot Password') }}</h1>
            <p class="auth-subtitle">{{ t('auth.forgotSubtitle', 'Enter your email to receive a reset link') }}</p>

            <form @submit.prevent="handleSubmit" class="auth-form">
                <div class="form-group">
                    <label>{{ t('auth.email') }}</label>
                    <input v-model="email" type="email" :placeholder="t('auth.email')" required class="glass-input" />
                </div>

                <div v-if="message" class="alert alert-success mt-4">
                    {{ message }}
                </div>
                <div v-if="error" class="alert alert-danger mt-4">
                    {{ error }}_{{ t('auth.emailNotFound', 'If account exists, email sent.') }}
                </div>

                <button type="submit" :disabled="loading" class="btn btn-primary w-full mt-6">
                    {{ loading ? t('common.status.loading') : t('auth.sendResetLink', 'Send Reset Link') }}
                </button>

                <div class="auth-footer mt-6">
                    <button @click="router.push('/auth')" class="btn-link">
                        {{ t('auth.backToLogin', 'Back to Login') }}
                    </button>
                </div>
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

.btn-link {
    background: none;
    border: none;
    color: white;
    font-size: 0.875rem;
    cursor: pointer;
    text-decoration: underline;
    opacity: 0.8;
}

.btn-link:hover {
    opacity: 1;
}

.auth-footer {
    text-align: center;
}
</style>

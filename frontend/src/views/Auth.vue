<script setup lang="ts">
import { ref } from 'vue';
import http from '../http';
import { useRouter } from 'vue-router';
import { useAuthStore } from '../stores/auth';
import { useI18n } from 'vue-i18n';

const router = useRouter();
const authStore = useAuthStore();
const { t, locale } = useI18n();

const isLogin = ref(true);
const username = ref('');
const email = ref('');
const password = ref('');
const errorMsg = ref('');
const isLoading = ref(false);

const toggleMode = () => {
    isLogin.value = !isLogin.value;
    errorMsg.value = '';
};

const toggleLanguage = () => {
    locale.value = locale.value === 'en' ? 'zh' : 'en';
};

const handleSubmit = async () => {
    errorMsg.value = '';
    isLoading.value = true;
    try {
        if (isLogin.value) {
            const res = await http.post('/auth/signin', {
                username: username.value,
                password: password.value
            });
            authStore.setUser(res.data.user);
            router.push('/dashboard');
        } else {
            const res = await http.post('/auth/signup', {
                username: username.value,
                email: email.value,
                password: password.value
            });
            authStore.setUser(res.data.user || res.data);
            router.push('/dashboard');
        }
    } catch (err: any) {
        errorMsg.value = err.response?.data?.message || err.response?.data || t('common.status.error');
    } finally {
        isLoading.value = false;
    }
};
</script>

<template>
    <div class="auth-container">
        <div class="auth-card">
            <div class="auth-header">
                <h1 class="logo">SubFlow</h1>
                <p class="subtitle">{{ isLogin ? t('auth.loginTitle') : t('auth.signupTitle') }}</p>
            </div>

            <form @submit.prevent="handleSubmit" class="auth-form">
                <div class="form-group">
                    <label>{{ t('auth.username') }}</label>
                    <input type="text" v-model="username" required minlength="3" class="input-field"
                        :placeholder="t('auth.username')" />
                </div>

                <div v-if="!isLogin" class="form-group">
                    <label>{{ t('auth.email') }}</label>
                    <input type="email" v-model="email" required class="input-field" :placeholder="t('auth.email')" />
                </div>

                <div class="form-group">
                    <label>{{ t('auth.password') }}</label>
                    <input type="password" v-model="password" required minlength="6" class="input-field"
                        :placeholder="t('auth.password')" />
                </div>

                <div v-if="errorMsg" class="error-banner">
                    {{ errorMsg }}
                </div>

                <button type="submit" class="btn-submit" :disabled="isLoading">
                    {{ isLoading ? t('common.status.loading') : (isLogin ? t('auth.loginBtn') : t('auth.signupBtn')) }}
                </button>
            </form>

            <div class="auth-footer">
                <p @click="toggleMode" class="toggle-link">
                    {{ isLogin ? t('auth.needAccount') : t('auth.haveAccount') }}
                </p>
                <p v-if="isLogin" @click="router.push('/auth/forgot-password')" class="toggle-link">
                    {{ t('auth.forgotPasswordLink', 'Forgot Password?') }}
                </p>
                <button @click="toggleLanguage" class="lang-toggle">
                    {{ locale === 'en' ? '中文' : 'English' }}
                </button>
            </div>
        </div>
    </div>
</template>

<style scoped>
.auth-container {
    display: flex;
    justify-content: center;
    align-items: center;
    min-height: 100vh;
    background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
    padding: 1rem;
}

.auth-card {
    background: white;
    padding: 2.5rem;
    border-radius: var(--radius-lg);
    box-shadow: 0 10px 25px rgba(0, 0, 0, 0.2);
    width: 100%;
    max-width: 420px;
    animation: slideUp 0.5s ease-out;
}

@keyframes slideUp {
    from {
        opacity: 0;
        transform: translateY(20px);
    }

    to {
        opacity: 1;
        transform: translateY(0);
    }
}

.auth-header {
    text-align: center;
    margin-bottom: 2rem;
}

.logo {
    font-size: 2.5rem;
    font-weight: 800;
    margin: 0;
    background: linear-gradient(to right, #667eea, #764ba2);
    -webkit-background-clip: text;
    background-clip: text;
    -webkit-text-fill-color: transparent;
}

.subtitle {
    color: var(--text-muted);
    font-size: 1.1rem;
    margin-top: 0.5rem;
}

.form-group {
    margin-bottom: 1.25rem;
}

.form-group label {
    display: block;
    margin-bottom: 0.5rem;
    color: var(--text-main);
    font-weight: 500;
    font-size: 0.9rem;
}

.input-field {
    width: 100%;
    padding: 0.75rem 1rem;
    border: 2px solid #e2e8f0;
    border-radius: var(--radius-md);
    font-size: 1rem;
    transition: all 0.2s;
    outline: none;
}

.input-field:focus {
    border-color: var(--primary-color);
    box-shadow: 0 0 0 3px rgba(79, 70, 229, 0.1);
}

.btn-submit {
    width: 100%;
    padding: 0.875rem;
    background: linear-gradient(to right, #667eea, #764ba2);
    color: white;
    border: none;
    border-radius: var(--radius-md);
    font-size: 1rem;
    font-weight: 600;
    cursor: pointer;
    transition: transform 0.1s, opacity 0.2s;
    margin-top: 1rem;
}

.btn-submit:hover {
    opacity: 0.9;
    transform: translateY(-1px);
}

.btn-submit:active {
    transform: translateY(0);
}

.btn-submit:disabled {
    opacity: 0.7;
    cursor: not-allowed;
}

.error-banner {
    background-color: #fee2e2;
    color: #991b1b;
    padding: 0.75rem;
    border-radius: var(--radius-sm);
    font-size: 0.9rem;
    margin-bottom: 1rem;
    text-align: center;
}

.auth-footer {
    margin-top: 2rem;
    text-align: center;
    border-top: 1px solid #f1f5f9;
    padding-top: 1.5rem;
}

.toggle-link {
    color: var(--primary-color);
    cursor: pointer;
    font-size: 0.95rem;
    font-weight: 500;
    margin-bottom: 1rem;
}

.toggle-link:hover {
    text-decoration: underline;
}

.lang-toggle {
    background: none;
    border: none;
    color: var(--text-muted);
    font-size: 0.85rem;
    cursor: pointer;
    padding: 0.5rem;
}

.lang-toggle:hover {
    color: var(--text-main);
}
</style>

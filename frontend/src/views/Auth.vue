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
const requires2FA = ref(false);
const twoFactorCode = ref('');
const pendingUserId = ref('');

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
        if (requires2FA.value) {
            const res = await http.post('/auth/signin/2fa', {
                userId: pendingUserId.value,
                code: twoFactorCode.value
            });
            authStore.setUser(res.data.user);
            router.push('/dashboard');
            return;
        }

        if (isLogin.value) {
            const res = await http.post('/auth/signin', {
                username: username.value,
                password: password.value
            });
            
            if (res.data.requires2FA) {
                requires2FA.value = true;
                pendingUserId.value = res.data.userId;
                return;
            }

            authStore.setUser(res.data.user);
            router.push('/dashboard');
        } else {
            const res = await http.post('/auth/signup', {
                username: username.value,
                email: email.value,
                password: password.value
            });
            authStore.setUser(res.data.user);
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
    <div class="min-h-screen w-full flex items-center justify-center p-4 relative overflow-hidden bg-body">
        <!-- Dynamic Background -->
        <div class="absolute top-0 left-0 w-full h-full overflow-hidden -z-10">
            <div class="blob blob-1"></div>
            <div class="blob blob-2"></div>
            <div class="blob blob-3"></div>
        </div>

        <div
            class="w-full max-w-4xl grid md:grid-cols-2 shadow-2xl rounded-3xl overflow-hidden glass-panel animate-fade-in">
            <!-- Left Side: Brand Area -->
            <div
                class="hidden md:flex flex-col justify-center items-center p-12 bg-primary-gradient relative text-white">
                <div class="absolute inset-0 bg-pattern opacity-10"></div>
                <div class="relative z-10 text-center">
                    <div
                        class="w-20 h-20 bg-white/20 backdrop-blur-md rounded-2xl flex items-center justify-center mb-6 mx-auto shadow-glow border border-white/30">
                        <img src="/favicon.svg" class="h-12 w-12" alt="SubFlow Logo" />
                    </div>
                    <h1 class="text-4xl font-extrabold mb-2 tracking-tight">SubFlow</h1>
                    <p class="text-primary-100 text-lg opacity-90">Manage your subscriptions with elegance.</p>
                </div>

                <!-- Decoration Cycles -->
                <div class="absolute bottom-0 left-0 w-full h-1/3 bg-gradient-to-t from-black/20 to-transparent"></div>
            </div>

            <!-- Right Side: Form Area -->
            <div class="p-8 md:p-12 bg-white/80 backdrop-blur-xl flex flex-col justify-center relative">
                <div class="absolute top-4 right-4">
                    <button @click="toggleLanguage"
                        class="px-3 py-1 rounded-full text-xs font-medium bg-slate-100 hover:bg-slate-200 text-slate-600 transition-colors">
                        {{ locale === 'en' ? '中文' : 'EN' }}
                    </button>
                </div>

                <div class="mb-8">
                    <h2 class="text-2xl font-bold text-gray-800 mb-2">
                        {{ isLogin ? t('auth.loginTitle') : t('auth.signupTitle') }}
                    </h2>
                    <p class="text-slate-500 text-sm">
                        {{ isLogin ? t('auth.welcomeBack', 'Welcome back! Please enter your details.') :
                            t('auth.createAccount', 'Start your journey with us today.') }}
                    </p>
                </div>

                <form @submit.prevent="handleSubmit" class="space-y-5">
                    <div class="space-y-4">
                        <div class="form-group">
                            <label class="form-label text-sm">{{ t('auth.username') }}</label>
                            <input type="text" v-model="username" required minlength="3"
                                class="glass-input bg-slate-50 focus:bg-white" :placeholder="t('auth.username')" />
                        </div>

                        <div v-if="!isLogin" class="form-group animate-slide-in">
                            <label class="form-label text-sm">{{ t('auth.email') }}</label>
                            <input type="email" v-model="email" required class="glass-input bg-slate-50 focus:bg-white"
                                :placeholder="t('auth.email')" />
                        </div>

                        <div v-if="!requires2FA" class="form-group">
                            <label class="form-label text-sm">{{ t('auth.password') }}</label>
                            <input type="password" v-model="password" required minlength="8"
                                class="glass-input bg-slate-50 focus:bg-white" :placeholder="t('auth.password')" />

                            <div v-if="isLogin" class="flex justify-end mt-1">
                                <span @click="router.push('/auth/forgot-password')"
                                    class="text-xs text-primary-600 hover:text-primary-700 cursor-pointer font-medium">
                                    {{ t('auth.forgotPasswordLink', 'Forgot Password?') }}
                                </span>
                            </div>
                        </div>

                        <!-- 2FA Input -->
                        <div v-if="requires2FA" class="form-group animate-slide-in">
                            <label class="form-label text-sm">{{ t('security.twoFactorAuth', 'Two-Factor Authentication') }}</label>
                            <input type="text" v-model="twoFactorCode" required maxlength="6"
                                class="glass-input bg-slate-50 focus:bg-white text-center text-2xl tracking-[1rem] uppercase" 
                                placeholder="000000" />
                            <p class="text-xs text-slate-500 mt-2 text-center">{{ t('auth.enterTotp', 'Please enter your 6-digit TOTP code.') }}</p>
                        </div>
                    </div>

                    <div v-if="errorMsg"
                        class="p-3 rounded-lg bg-red-50 text-red-600 text-sm flex items-center gap-2 animate-pulse">
                        <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24"
                            stroke="currentColor">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                                d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                        </svg>
                        {{ errorMsg }}
                    </div>

                    <button type="submit"
                        class="btn btn-primary w-full shadow-lg hover:shadow-xl transform transition-all duration-200"
                        :disabled="isLoading">
                        <svg v-if="isLoading" class="animate-spin -ml-1 mr-2 h-4 w-4 text-white"
                            xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
                            <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4">
                            </circle>
                            <path class="opacity-75" fill="currentColor"
                                d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z">
                            </path>
                        </svg>
                        {{ isLoading ? t('common.status.loading') : (requires2FA ? t('common.actions.verify', 'Verify') : (isLogin ? t('auth.loginBtn') : t('auth.signupBtn')))
                        }}
                    </button>

                    <div class="relative flex items-center justify-center mt-6">
                        <div class="h-px bg-slate-200 w-full absolute"></div>
                        <span class="bg-white px-2 z-10 text-xs text-slate-400 font-medium">OR</span>
                    </div>

                    <p class="text-center text-sm text-slate-600 mt-4">
                        {{ isLogin ? t('auth.needAccount') : t('auth.haveAccount') }}
                        <span @click="toggleMode"
                            class="text-primary-600 font-bold cursor-pointer hover:underline ml-1">
                            {{ isLogin ? t('auth.signupLink', 'Sign up') : t('auth.loginLink', 'Log in') }}
                        </span>
                    </p>
                </form>
            </div>
        </div>
    </div>
</template>

<style scoped>
/* Blob Animations */
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

.blob-3 {
    top: 40%;
    left: 40%;
    width: 300px;
    height: 300px;
    background: #a7f3d0;
    filter: blur(60px);
    animation-delay: -2s;
    opacity: 0.3;
}

.bg-pattern {
    background-image: radial-gradient(circle, #ffffff 1px, transparent 1px);
    background-size: 20px 20px;
}

/* Tailwind-ish Utilities for Scoped Style */
.bg-primary-gradient {
    background: var(--primary-gradient);
}

.shadow-glow {
    box-shadow: 0 0 20px rgba(255, 255, 255, 0.3);
}

/* Responsive Adjustments */
@media (max-width: 768px) {
    .blob {
        opacity: 0.4;
    }

    .glass-panel {
        margin: 1rem;
    }
}
</style>

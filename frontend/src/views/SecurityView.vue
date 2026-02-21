<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed } from 'vue';
import { useI18n } from 'vue-i18n';
import { socket } from '../socket';
import http from '../http';
import MainLayout from './MainLayout.vue';
import PasswordRequirements from '../components/PasswordRequirements.vue';

const { t } = useI18n();
const settings = ref<any>(null);
const sessions = ref<any[]>([]);
const loading = ref(true);

const isSettingUp2FA = ref(false);
const showDisable2FAModal = ref(false);
const setupSecret = ref('');
const qrCodeUrl = ref('');
const setupCode = ref('');
const errorMsg = ref('');
const showOldPassword = ref(false);
const showNewPassword = ref(false);
const showConfirmPassword = ref(false);

const backupCodes = ref<string[]>([]);
const showBackupCodesModal = ref(false);

const fetchData = () => {
    loading.value = true;
    socket.emit('security:get_settings', (res: any) => {
        if (res.status === 'ok') settings.value = res.settings;
    });
    // Fetch active sessions instead of devices
    socket.emit('security:sessions', (res: any) => {
        if (res.status === 'ok') {
            sessions.value = res.sessions.sort((a: any, b: any) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime());
        }
        loading.value = false;
    });
};

const toggle2FA = () => {
    if (settings.value?.two_factor_enabled) {
        showDisable2FAModal.value = true;
    } else {
        socket.emit('security:generate_2fa_secret', (res: any) => {
            if (res.status === 'ok') {
                setupSecret.value = res.secret;
                qrCodeUrl.value = res.qrDataUrl;
                isSettingUp2FA.value = true;
            } else {
                alert(res.message || "Failed to initiate 2FA setup");
            }
        });
    }
};

const confirmDisable2FA = () => {
    socket.emit('security:disable_2fa', (res: any) => {
        if (res.status === 'ok') {
            settings.value.two_factor_enabled = false;
        }
        showDisable2FAModal.value = false;
    });
};

const cancelDisable2FA = () => {
    showDisable2FAModal.value = false;
};

const verifyAndEnable2FA = () => {
    errorMsg.value = '';
    socket.emit('security:enable_2fa', { secret: setupSecret.value, code: setupCode.value }, (res: any) => {
        if (res.status === 'ok') {
            settings.value.two_factor_enabled = true;
            isSettingUp2FA.value = false;
            setupCode.value = '';
            setupSecret.value = '';
            backupCodes.value = res.backupCodes || [];
            showBackupCodesModal.value = true;
        } else {
            errorMsg.value = res.message || "Verification failed";
        }
    });
};

const regenerateBackupCodes = () => {
    if (confirm(t('security.confirmRegenBackupCodes', 'Are you sure you want to regenerate backup codes? Your old codes will immediately stop working.'))) {
        socket.emit('security:regenerate_backup_codes', (res: any) => {
            if (res.status === 'ok') {
                backupCodes.value = res.backupCodes || [];
                showBackupCodesModal.value = true;
            } else {
                alert(res.message || "Failed to regenerate backup codes");
            }
        });
    }
};

const copyBackupCodes = () => {
    navigator.clipboard.writeText(backupCodes.value.join('\n'))
        .then(() => alert(t('security.copiedBackupCodes', 'Backup codes copied to clipboard!')));
};

const cancelSetup = () => {
    isSettingUp2FA.value = false;
    setupCode.value = '';
    setupSecret.value = '';
    qrCodeUrl.value = '';
};

const revokeSession = (sessionId: string) => {
    if (confirm(t('security.confirmRevokeSession', 'Are you sure you want to revoke this session?'))) {
        socket.emit('security:revoke_session', { sessionId }, (res: any) => {
            if (res.status === 'ok') {
                sessions.value = sessions.value.filter(s => s.id !== sessionId);
            }
        });
    }
};

const formatUserAgent = (uaString: string) => {
    if (!uaString) return 'Unknown Device';

    let browser = 'Unknown Browser';
    if (uaString.indexOf("Firefox") > -1) browser = "Firefox";
    else if (uaString.indexOf("SamsungBrowser") > -1) browser = "Samsung Internet";
    else if (uaString.indexOf("Opera") > -1 || uaString.indexOf("OPR") > -1) browser = "Opera";
    else if (uaString.indexOf("Trident") > -1) browser = "Internet Explorer";
    else if (uaString.indexOf("Edge") > -1) browser = "Edge";
    else if (uaString.indexOf("Chrome") > -1) browser = "Chrome";
    else if (uaString.indexOf("Safari") > -1) browser = "Safari";

    let os = 'Unknown OS';
    if (uaString.indexOf("Win") > -1) os = "Windows";
    else if (uaString.indexOf("Mac") > -1) os = "MacOS";
    else if (uaString.indexOf("Linux") > -1) os = "Linux";
    else if (uaString.indexOf("Android") > -1) os = "Android";
    else if (uaString.indexOf("like Mac") > -1) os = "iOS";

    return `${browser} on ${os}`;
};

// isCurrentSession logic is now handled by backend (session.is_current)

// Formatted Secret for readability
const formattedSecret = computed(() => {
    if (!setupSecret.value) return '';
    return setupSecret.value.match(/.{1,4}/g)?.join(' ') || setupSecret.value;
});

// Password Change Logic
const passwordForm = ref({
    oldPassword: '',
    newPassword: '',
    confirmPassword: ''
});
const passwordMsg = ref({ type: '', text: '' });
const isChangingPassword = ref(false);

const changePassword = async () => {
    passwordMsg.value = { type: '', text: '' };

    // Client-side validation
    if (passwordForm.value.newPassword !== passwordForm.value.confirmPassword) {
        passwordMsg.value = { type: 'error', text: t('auth.passwordMismatch', 'Passwords do not match') };
        return;
    }

    const passwordRegex = /^(?=.*[A-Z])(?=.*[0-9])(?=.*[^A-Za-z0-9]).{8,}$/;
    if (!passwordRegex.test(passwordForm.value.newPassword)) {
        passwordMsg.value = { type: 'error', text: t('auth.passwordComplexity', 'Password must be at least 8 characters and include uppercase, numbers, and symbols') };
        return;
    }

    isChangingPassword.value = true;
    try {
        await http.post('/auth/change-password', {
            oldPassword: passwordForm.value.oldPassword,
            newPassword: passwordForm.value.newPassword
        });
        passwordMsg.value = { type: 'success', text: t('auth.passwordResetSuccess', 'Password changed successfully') };
        passwordForm.value = { oldPassword: '', newPassword: '', confirmPassword: '' };
    } catch (err: any) {
        const errorData = err.response?.data;
        if (errorData?.error && Array.isArray(errorData.error)) {
            passwordMsg.value = { type: 'error', text: errorData.error[0].message };
        } else {
            const msg = errorData?.message || errorData || 'auth.errors.unknownError';
            passwordMsg.value = { type: 'error', text: t(msg, msg) };
        }
    } finally {
        isChangingPassword.value = false;
    }
};

onMounted(() => {
    if (socket.connected) {
        fetchData();
    } else {
        socket.on('connect', fetchData);
    }
});

onUnmounted(() => {
    socket.off('connect', fetchData);
});
</script>

<template>
    <MainLayout>
        <div class="security-view max-w-6xl mx-auto px-4 py-8 animate-fade-in">
            <!-- Header Section -->
            <div class="mb-10 text-center md:text-left">
                <h1 class="text-4xl font-extrabold text-slate-800 tracking-tight mb-2">
                    {{ t('security.security') }}
                </h1>
                <p class="text-slate-500 max-w-2xl">
                    {{ t('security.subtitle', 'Manage your account security, 2FA settings and active sessions.') }}
                </p>
            </div>

            <div class="grid grid-cols-1 lg:grid-cols-2 gap-8">
                <!-- Left Column: 2FA & Password -->
                <div class="space-y-8 h-full flex flex-col">
                    <!-- 2FA Section -->
                    <section class="glass-panel p-8 relative overflow-hidden flex-none">
                        <div class="flex items-center justify-between mb-8">
                            <div class="flex items-center gap-4">
                                <div
                                    class="w-12 h-12 rounded-2xl bg-primary-100 flex items-center justify-center text-primary-600 shadow-sm">
                                    <svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6" fill="none"
                                        viewBox="0 0 24 24" stroke="currentColor">
                                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                                            d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" />
                                    </svg>
                                </div>
                                <div>
                                    <h3 class="text-xl font-bold text-slate-800">{{ t('security.twoFactorAuth') }}</h3>
                                    <div class="flex items-center gap-2 mt-1">
                                        <span
                                            :class="['status-dot', settings?.two_factor_enabled ? 'bg-success' : 'bg-slate-300 animate-pulse']"></span>
                                        <span
                                            :class="['text-xs font-bold uppercase tracking-wider', settings?.two_factor_enabled ? 'text-success' : 'text-slate-500']">
                                            {{ settings?.two_factor_enabled ? t('security.enabled') :
                                                t('security.disabled') }}
                                        </span>
                                    </div>
                                </div>
                            </div>
                            <button @click="toggle2FA"
                                :class="['btn px-6 py-2 rounded-full text-sm', settings?.two_factor_enabled ? 'bg-slate-100 text-slate-600 hover:bg-danger-color hover:text-white' : 'btn-primary']">
                                {{ settings?.two_factor_enabled ? t('security.disable2FA') : t('security.enable2FA') }}
                            </button>
                        </div>

                        <p class="text-slate-500 text-sm mb-6 leading-relaxed">
                            {{ t('security.2faDesc',
                                'Two-factor authentication adds an extra layer of security to your account. \
                            To log in, you will also need to provide a 6 - digit code from your authenticator app.') }}
                        </p>

                        <!-- Regenerate Backup Codes Action -->
                        <div v-if="settings?.two_factor_enabled && !isSettingUp2FA" class="mb-6">
                            <button @click="regenerateBackupCodes"
                                class="text-sm font-semibold text-primary-600 hover:text-primary-700 underline underline-offset-2 transition-colors">
                                {{ t('security.regenerateBackupCodes', 'Regenerate Backup Codes') }}
                            </button>
                        </div>

                        <!-- 2FA Setup Flow -->
                        <div v-if="isSettingUp2FA" class="animate-slide-in mt-auto">
                            <div class="p-6 rounded-2xl bg-primary-50/50 border border-primary-100">
                                <h4 class="font-bold text-primary-800 mb-4 flex items-center gap-2">
                                    <span
                                        class="w-6 h-6 rounded-full bg-primary-600 text-white flex-center text-xs">1</span>
                                    {{ t('security.setupTitle', 'Scan QR Code or enter code manually') }}
                                </h4>
                                <p class="text-xs text-slate-600 mb-4 leading-relaxed">
                                    {{ t('security.setupDesc', 'Scan the QR code below using your authenticator app.')
                                    }}
                                </p>

                                <div v-if="qrCodeUrl" class="flex justify-center mb-6">
                                    <img :src="qrCodeUrl" alt="2FA QR Code"
                                        class="rounded-xl border border-slate-200 shadow-sm w-48 h-48" />
                                </div>

                                <div class="bg-white/80 p-4 rounded-xl border-2 border-dashed border-primary-200 font-mono text-center text-xl tracking-widest text-primary-700 shadow-inner mb-6 transition-all hover:bg-white select-all cursor-pointer"
                                    title="Click to select">
                                    {{ formattedSecret }}
                                </div>

                                <h4 class="font-bold text-primary-800 mb-4 flex items-center gap-2">
                                    <span
                                        class="w-6 h-6 rounded-full bg-primary-600 text-white flex-center text-xs">2</span>
                                    {{ t('security.verifyCode') }}
                                </h4>
                                <div class="space-y-4">
                                    <div class="relative">
                                        <input type="text" v-model="setupCode"
                                            class="glass-input h-14 text-center text-2xl tracking-[0.5em] font-bold"
                                            placeholder="000000" maxlength="6" />
                                        <div
                                            class="absolute inset-y-0 right-4 flex items-center pointer-events-none text-slate-300">
                                            <svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6" fill="none"
                                                viewBox="0 0 24 24" stroke="currentColor">
                                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                                                    d="M12 11c0 3.517-1.009 6.799-2.753 9.571m-4.44-2.03c.52-1.282.909-2.607 1.155-3.974m2.812-3.162l3.414-3.414m0 0a2 2 0 10-2.828-2.828l-3.414 3.414m0 0L5.95 10.122m3.162-3.162l3.414-3.414m0 0a2 2 0 10-2.828-2.828l-3.414 3.414m0 0L5.95 10.122" />
                                            </svg>
                                        </div>
                                    </div>

                                    <div v-if="errorMsg"
                                        class="flex items-center gap-2 p-3 rounded-lg bg-danger/10 text-danger text-xs font-bold animate-pulse">
                                        <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" viewBox="0 0 20 20"
                                            fill="currentColor">
                                            <path fill-rule="evenodd"
                                                d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7 4a1 1 0 11-2 0 1 1 0 012 0zm-1-9a1 1 0 00-1 1v4a1 1 0 102 0V6a1 1 0 00-1-1z"
                                                clip-rule="evenodd" />
                                        </svg>
                                        {{ errorMsg }}
                                    </div>

                                    <div class="flex gap-3">
                                        <button @click="verifyAndEnable2FA"
                                            class="btn btn-primary flex-1 shadow-lg shadow-primary-500/20">
                                            {{ t('common.actions.verify', 'Verify & Enable') }}
                                        </button>
                                        <button @click="cancelSetup"
                                            class="btn bg-white hover:bg-slate-50 text-slate-500 border border-slate-200">
                                            {{ t('common.actions.cancel', 'Cancel') }}
                                        </button>
                                    </div>
                                </div>
                            </div>
                        </div>
                    </section>

                    <!-- Change Password Section -->
                    <section class="glass-panel p-8 flex-none bg-white/60">
                        <div class="flex items-center gap-4 mb-8">
                            <div
                                class="w-12 h-12 rounded-2xl bg-indigo-100 flex items-center justify-center text-indigo-600 shadow-sm">
                                <svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6" fill="none" viewBox="0 0 24 24"
                                    stroke="currentColor">
                                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                                        d="M15 7a2 2 0 012 2m4 0a6 6 0 01-7.743 5.743L11 17H9v2H7v2H4a1 1 0 01-1-1v-2.586a1 1 0 01.293-.707l5.964-5.964A6 6 0 1121 9z" />
                                </svg>
                            </div>
                            <h3 class="text-xl font-bold text-slate-800">{{ t('security.changePassword') }}</h3>
                        </div>

                        <form @submit.prevent="changePassword" class="space-y-5">
                            <div class="space-y-4">
                                <div class="form-group">
                                    <label class="form-label text-sm font-semibold text-slate-600 block mb-1.5">{{
                                        t('auth.oldPassword') }}</label>
                                    <div class="relative">
                                        <input :type="showOldPassword ? 'text' : 'password'"
                                            v-model="passwordForm.oldPassword" required class="glass-input w-full pr-12"
                                            :placeholder="t('auth.oldPassword')" />
                                        <button type="button" @click="showOldPassword = !showOldPassword"
                                            class="absolute right-3 top-1/2 -translate-y-1/2 p-2 text-slate-400 hover:text-slate-600 transition-colors">
                                            <svg v-if="showOldPassword" xmlns="http://www.w3.org/2000/svg"
                                                class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
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
                                    <label class="form-label text-sm font-semibold text-slate-600 block mb-1.5">{{
                                        t('auth.newPassword') }}</label>
                                    <div class="relative">
                                        <input :type="showNewPassword ? 'text' : 'password'"
                                            v-model="passwordForm.newPassword" required class="glass-input w-full pr-12"
                                            :placeholder="t('auth.newPassword')" />
                                        <button type="button" @click="showNewPassword = !showNewPassword"
                                            class="absolute right-3 top-1/2 -translate-y-1/2 p-2 text-slate-400 hover:text-slate-600 transition-colors">
                                            <svg v-if="showNewPassword" xmlns="http://www.w3.org/2000/svg"
                                                class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
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
                                    <PasswordRequirements v-if="passwordForm.newPassword"
                                        :password="passwordForm.newPassword" />
                                </div>

                                <div class="form-group">
                                    <label class="form-label text-sm font-semibold text-slate-600 block mb-1.5">{{
                                        t('auth.confirmPassword') }}</label>
                                    <div class="relative">
                                        <input :type="showConfirmPassword ? 'text' : 'password'"
                                            v-model="passwordForm.confirmPassword" required
                                            class="glass-input w-full pr-12" :placeholder="t('auth.confirmPassword')" />
                                        <button type="button" @click="showConfirmPassword = !showConfirmPassword"
                                            class="absolute right-3 top-1/2 -translate-y-1/2 p-2 text-slate-400 hover:text-slate-600 transition-colors">
                                            <svg v-if="showConfirmPassword" xmlns="http://www.w3.org/2000/svg"
                                                class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
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
                            </div>

                            <div v-if="passwordMsg.text"
                                :class="['flex items-center gap-2 p-4 rounded-xl text-sm font-bold animate-fade-in', passwordMsg.type === 'error' ? 'bg-danger/10 text-danger' : 'bg-success/10 text-success']">
                                <svg v-if="passwordMsg.type === 'error'" xmlns="http://www.w3.org/2000/svg"
                                    class="h-5 w-5" viewBox="0 0 20 20" fill="currentColor">
                                    <path fill-rule="evenodd"
                                        d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z"
                                        clip-rule="evenodd" />
                                </svg>
                                <svg v-else xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" viewBox="0 0 20 20"
                                    fill="currentColor">
                                    <path fill-rule="evenodd"
                                        d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z"
                                        clip-rule="evenodd" />
                                </svg>
                                {{ passwordMsg.text }}
                            </div>

                            <button type="submit" :disabled="isChangingPassword"
                                class="btn btn-primary w-full shadow-lg shadow-primary-500/20 h-12">
                                <svg v-if="isChangingPassword" class="animate-spin h-5 w-5 text-white"
                                    xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
                                    <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor"
                                        stroke-width="4"></circle>
                                    <path class="opacity-75" fill="currentColor"
                                        d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z">
                                    </path>
                                </svg>
                                <span>{{ isChangingPassword ? t('common.status.saving') : t('common.actions.update')
                                }}</span>
                            </button>
                        </form>
                    </section>
                </div>

                <!-- Right Column: Session Management -->
                <div class="space-y-8 flex flex-col">
                    <section class="glass-panel p-8 flex flex-col flex-1 h-full">
                        <div class="flex items-center gap-4 mb-8">
                            <div
                                class="w-12 h-12 rounded-2xl bg-amber-100 flex items-center justify-center text-amber-600 shadow-sm">
                                <svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6" fill="none" viewBox="0 0 24 24"
                                    stroke="currentColor">
                                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                                        d="M9.75 17L9 20l-1 1h8l-1-1-.75-3M3 13h18M5 17h14a2 2 0 002-2V5a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
                                </svg>
                            </div>
                            <h3 class="text-xl font-bold text-slate-800">{{ t('security.devices', 'Active Sessions') }}
                            </h3>
                        </div>

                        <div v-if="loading" class="flex-center flex-col gap-4 py-20 flex-1">
                            <div
                                class="animate-spin rounded-full h-10 w-10 border-4 border-primary-200 border-t-primary-600">
                            </div>
                            <p class="text-slate-400 font-medium text-sm">{{ t('common.status.loading') }}</p>
                        </div>

                        <div v-else-if="sessions.length === 0"
                            class="flex-center flex-col gap-4 py-20 flex-1 opacity-50">
                            <svg xmlns="http://www.w3.org/2000/svg" class="h-16 w-16 text-slate-300" fill="none"
                                viewBox="0 0 24 24" stroke="currentColor">
                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1"
                                    d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
                            </svg>
                            <p class="text-slate-400 font-bold uppercase tracking-widest text-xs">{{ t('common.noData')
                            }}</p>
                        </div>

                        <div v-else class="space-y-4 flex-1">
                            <div v-for="session in sessions" :key="session.id"
                                class="glass-card p-5 group transition-all hover:scale-[1.02] flex items-center justify-between">
                                <div class="flex items-center gap-5 min-w-0">
                                    <div
                                        class="w-12 h-12 rounded-xl flex items-center justify-center transition-colors bg-slate-100 text-slate-400 group-hover:bg-primary-50 group-hover:text-primary-400">
                                        <svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6" fill="none"
                                            viewBox="0 0 24 24" stroke="currentColor">
                                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                                                d="M9.75 17L9 20l-1 1h8l-1-1-.75-3M3 13h18M5 17h14a2 2 0 002-2V5a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
                                        </svg>
                                    </div>
                                    <div class="min-w-0 flex-1">
                                        <div class="flex items-center gap-2">
                                            <p class="font-bold text-slate-700 truncate" :title="session.user_agent">
                                                {{ formatUserAgent(session.user_agent) }}
                                            </p>
                                        </div>
                                        <p class="text-xs text-slate-500 font-mono mt-1 w-full truncate"
                                            :title="session.user_agent">
                                            {{ session.user_agent || 'Unknown User-Agent' }}
                                        </p>
                                        <p class="text-xs text-slate-400 mt-1 flex items-center gap-1">
                                            <svg xmlns="http://www.w3.org/2000/svg" class="h-3 w-3" fill="none"
                                                viewBox="0 0 24 24" stroke="currentColor">
                                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                                                    d="M17.657 16.657L13.414 20.9a1.998 1.998 0 01-2.827 0l-4.244-4.243a8 8 0 1111.314 0z" />
                                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                                                    d="M15 11a3 3 0 11-6 0 3 3 0 016 0z" />
                                            </svg>
                                            {{ session.ip_address }}
                                        </p>
                                        <p class="text-[10px] text-slate-400 mt-0.5 opacity-75">
                                            {{ t('security.lastUsed') }}: {{ new Date(session.created_at ||
                                                Date.now()).toLocaleString() }}
                                        </p>
                                    </div>
                                </div>
                                <button @click="revokeSession(session.id)"
                                    class="p-3 rounded-xl bg-red-50 text-red-500 hover:bg-red-600 hover:text-white transition-all shadow-sm opacity-100 flex items-center justify-center border border-red-100"
                                    :title="t('security.revoke')">
                                    <svg xmlns="http://www.w3.org/2000/svg" class="h-10 w-10" fill="none"
                                        viewBox="0 0 24 24" stroke="currentColor">
                                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                                            d="M17 16l4-4m0 0l-4-4m4 4H7m6 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h4a3 3 0 013 3v1" />
                                    </svg>
                                </button>
                            </div>
                        </div>

                        <div class="mt-8 p-4 rounded-xl bg-amber-50 border border-amber-100 flex gap-3">
                            <div class="text-amber-500 pt-0.5">
                                <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" viewBox="0 0 20 20"
                                    fill="currentColor">
                                    <path fill-rule="evenodd"
                                        d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a1 1 0 000 2v3a1 1 0 001 1h1a1 1 0 100-2v-3a1 1 0 00-1-1H9z"
                                        clip-rule="evenodd" />
                                </svg>
                            </div>
                            <p class="text-xs text-amber-700 leading-normal">
                                <strong>{{ t('common.optional', 'Recommendation') }}:</strong>
                                {{ t('security.revokeHint',
                                    'Regularly audit your active sessions. \
                                If you see something suspicious, revoke access immediately.') }}
                            </p>
                        </div>
                    </section>
                </div>
            </div>
        </div>

        <!-- Disable 2FA Modal -->
        <div v-if="showDisable2FAModal"
            class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-900/50 backdrop-blur-sm animate-fade-in">
            <div class="bg-white rounded-2xl shadow-xl w-full max-w-md overflow-hidden animate-slide-up">
                <div class="p-6">
                    <div class="w-12 h-12 rounded-full bg-red-100 flex items-center justify-center text-red-600 mb-4">
                        <svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6" fill="none" viewBox="0 0 24 24"
                            stroke="currentColor">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                                d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
                        </svg>
                    </div>
                    <h3 class="text-xl font-bold text-slate-800 mb-2">
                        {{ t('security.disable2FATitle',
                            'Disable Two - Factor Authentication') }}
                    </h3>
                    <p class="text-slate-500 text-sm leading-relaxed mb-6">
                        {{ t('security.confirmDisable2FA',
                            'Are you sure you want to disable 2FA? \
                        Disabling 2FA will make your account less secure.') }}
                    </p>
                    <div class="flex gap-3">
                        <button @click="confirmDisable2FA"
                            class="btn bg-red-500 hover:bg-red-600 text-white flex-1 shadow-lg shadow-red-500/20">
                            {{ t('common.actions.confirm', 'Confirm') }}
                        </button>
                        <button @click="cancelDisable2FA"
                            class="btn bg-white hover:bg-slate-50 text-slate-500 border border-slate-200 flex-1">
                            {{ t('common.actions.cancel', 'Cancel') }}
                        </button>
                    </div>
                </div>
            </div>
        </div>
        <!-- Backup Codes Modal -->
        <div v-if="showBackupCodesModal"
            class="fixed inset-0 bg-slate-900/40 backdrop-blur-sm flex items-center justify-center p-4 z-50 animate-fade-in">
            <div class="bg-white rounded-2xl shadow-xl max-w-md w-full p-8 border border-slate-100 relative">
                <button @click="showBackupCodesModal = false"
                    class="absolute top-4 right-4 text-slate-400 hover:text-slate-600 transition-colors">
                    <svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6" fill="none" viewBox="0 0 24 24"
                        stroke="currentColor">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                            d="M6 18L18 6M6 6l12 12" />
                    </svg>
                </button>
                <div class="mb-6 text-center">
                    <div
                        class="w-16 h-16 rounded-full bg-emerald-100 flex items-center justify-center text-emerald-600 mx-auto mb-4">
                        <svg xmlns="http://www.w3.org/2000/svg" class="h-8 w-8" fill="none" viewBox="0 0 24 24"
                            stroke="currentColor">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                                d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z" />
                        </svg>
                    </div>
                    <h3 class="text-2xl font-bold text-slate-800 mb-2">
                        {{ t("security.backupCodesTitle", "Your Backup Codes") }}
                    </h3>
                    <p class="text-slate-500 text-sm">
                        {{ t("security.backupCodesDesc",
                            "Save these codes in a secure place. \
                        If you lose access to your authenticator app, \
                        you can use these 8 - character codes to sign in. \
                        Each code can only be used once.") }}
                    </p>
                </div>

                <div class="grid grid-cols-2 gap-3 mb-6 bg-slate-50 p-4 rounded-xl border border-slate-200">
                    <div v-for="code in backupCodes" :key="code"
                        class="text-center font-mono font-bold text-slate-700 tracking-wider">
                        {{ code }}
                    </div>
                </div>

                <div class="flex gap-4">
                    <button @click="copyBackupCodes"
                        class="btn bg-slate-800 text-white hover:bg-slate-700 flex-1 shadow-md">
                        {{ t("security.copyCodes", "Copy Codes") }}
                    </button>
                    <button @click="showBackupCodesModal = false"
                        class="btn bg-white hover:bg-slate-50 text-slate-600 border border-slate-200 flex-1">
                        {{ t('common.actions.done', 'Done') }}
                    </button>
                </div>
            </div>
        </div>
    </MainLayout>
</template>

<style scoped>
.security-view {
    perspective: 1000px;
}

.status-dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
}

.bg-danger-color {
    background-color: var(--danger-color);
}

.text-success {
    color: #10b981;
}

.bg-success {
    background-color: #10b981;
}

.text-danger {
    color: #ef4444;
}

.bg-danger {
    background-color: #ef4444;
}

/* Custom Font logic matches layout */
:deep(.page-title) {
    background: var(--primary-gradient);
    -webkit-background-clip: text;
    background-clip: text;
    -webkit-text-fill-color: transparent;
}

.active-glow {
    box-shadow: 0 0 20px rgba(99, 102, 241, 0.4);
}

.select-all {
    user-select: all;
    -webkit-user-select: all;
}
</style>

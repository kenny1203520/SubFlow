<script setup lang="ts">
import { ref, onMounted } from 'vue';
import { useI18n } from 'vue-i18n';
import http from '../../http';
import { useUIStore } from '../../stores/ui';

const { t } = useI18n();
const ui = useUIStore();

interface Setting {
    key: string;
    value: any;
    description: string;
    updated_at: string;
}

const settingsRaw = ref<Setting[]>([]);
const loading = ref(true);
const saving = ref(false);

const captchaConfig = ref({
    enabled: false,
    provider: 'none',
    siteKey: '',
    secretKey: ''
});

const passwordPolicy = ref({
    minLength: 8,
    requireUppercase: true,
    requireLowercase: true,
    requireNumbers: true,
    requireSymbols: true
});

const twoFactor = ref({
    enabled: false
});

const securityConfig = ref({
    maxFailedAttempts: 5,
    lockoutDurationMins: 720,
    authWindowMs: 900000,
    authMax: 5,
    apiWindowMs: 900000,
    apiMax: 100
});

const fetchSettings = async () => {
    loading.value = true;
    try {
        const res = await http.get('/api/admin/settings');
        if (res.data.status === 'ok') {
            settingsRaw.value = res.data.list;
            const settingsMap = res.data.settings;

            if (settingsMap['auth.captcha']) captchaConfig.value = { ...captchaConfig.value, ...settingsMap['auth.captcha'] };
            if (settingsMap['auth.password_policy']) passwordPolicy.value = { ...passwordPolicy.value, ...settingsMap['auth.password_policy'] };
            if (settingsMap['auth.require_2fa']) twoFactor.value = { ...twoFactor.value, ...settingsMap['auth.require_2fa'] };
            if (settingsMap['security.auth_lockout']) {
                securityConfig.value.maxFailedAttempts = settingsMap['security.auth_lockout'].maxFailedAttempts ?? 5;
                securityConfig.value.lockoutDurationMins = settingsMap['security.auth_lockout'].lockoutDurationMins ?? 720;
            }
            if (settingsMap['security.rate_limit']) {
                securityConfig.value.authWindowMs = settingsMap['security.rate_limit'].authWindowMs ?? 900000;
                securityConfig.value.authMax = settingsMap['security.rate_limit'].authMax ?? 5;
                securityConfig.value.apiWindowMs = settingsMap['security.rate_limit'].apiWindowMs ?? 900000;
                securityConfig.value.apiMax = settingsMap['security.rate_limit'].apiMax ?? 100;
            }
        }
    } catch (e) {
        ui.alert(t('admin.settings.loadFailed'));
    } finally {
        loading.value = false;
    }
};

const saveSetting = async (key: string, value: any) => {
    saving.value = true;
    try {
        const res = await http.put('/api/admin/settings', { key, value });
        if (res.data.status === 'ok') {
            ui.alert(t('admin.settings.settingsSaved'));
            fetchSettings();
        }
    } catch (e) {
        ui.alert(t('admin.settings.saveSettingFailed', { key }));
    } finally {
        saving.value = false;
    }
};

const saveSecurityConfig = async () => {
    await saveSetting('security.auth_lockout', {
        maxFailedAttempts: securityConfig.value.maxFailedAttempts,
        lockoutDurationMins: securityConfig.value.lockoutDurationMins
    });
    await saveSetting('security.rate_limit', {
        authWindowMs: securityConfig.value.authWindowMs,
        authMax: securityConfig.value.authMax,
        apiWindowMs: securityConfig.value.apiWindowMs,
        apiMax: securityConfig.value.apiMax
    });
};

onMounted(() => {
    fetchSettings();
});

</script>

<template>
    <div class="space-y-8 max-w-4xl">
        <div>
            <h2 class="text-2xl font-bold bg-gradient-to-r from-red-400 to-rose-600 bg-clip-text text-transparent">
                {{ t('admin.settings.title') }}</h2>
            <p class="text-neutral-400 mt-1">{{ t('admin.settings.subtitle') }}</p>
        </div>

        <div v-if="loading" class="text-center py-10">
            <div class="w-8 h-8 border-4 border-red-500/20 border-t-red-500 rounded-full animate-spin mx-auto mb-4">
            </div>
            <p class="text-neutral-400">{{ t('admin.loadingConfigs') }}</p>
        </div>

        <div v-else class="space-y-8">

            <!-- Global 2FA Settings -->
            <section class="bg-neutral-900/50 border border-neutral-800 rounded-2xl p-6">
                <h3 class="text-lg font-semibold text-white mb-4">{{ t('admin.settings.twoFactor') }}</h3>
                <div class="space-y-4">
                    <label class="flex items-center space-x-3 text-sm text-neutral-300">
                        <input type="checkbox" v-model="twoFactor.enabled"
                            class="w-4 h-4 rounded border-neutral-700 bg-neutral-800 text-red-500 focus:ring-red-500 focus:ring-offset-neutral-900">
                        <span>{{ t('admin.settings.enforce2FA') }}</span>
                    </label>
                    <div class="flex justify-end mt-4">
                        <button @click="saveSetting('auth.require_2fa', twoFactor)" :disabled="saving"
                            class="px-4 py-2 bg-red-500/10 text-red-400 border border-red-500/20 rounded-xl hover:bg-red-500/20 font-medium transition disabled:opacity-50">
                            {{ t('admin.settings.save2FA') }}
                        </button>
                    </div>
                </div>
            </section>

            <!-- Password Policy -->
            <section class="bg-neutral-900/50 border border-neutral-800 rounded-2xl p-6">
                <h3 class="text-lg font-semibold text-white mb-4">{{ t('admin.settings.passwordPolicy') }}</h3>
                <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
                    <div>
                        <label class="block text-sm font-medium text-neutral-400 mb-2">{{ t('admin.settings.minLength')
                            }}</label>
                        <input type="number" v-model="passwordPolicy.minLength" min="4" max="64"
                            class="w-full bg-neutral-950/50 border border-neutral-800 rounded-xl px-4 py-2.5 text-neutral-200 focus:ring-2 focus:ring-red-500/50 focus:border-red-500 outline-none transition">
                    </div>
                    <div class="space-y-3 pt-8">
                        <label class="flex items-center space-x-3 text-sm text-neutral-300">
                            <input type="checkbox" v-model="passwordPolicy.requireUppercase"
                                class="w-4 h-4 rounded border-neutral-700 bg-neutral-800 text-red-500 focus:ring-red-500 focus:ring-offset-neutral-900">
                            <span>{{ t('admin.settings.requireUppercase') }}</span>
                        </label>
                        <label class="flex items-center space-x-3 text-sm text-neutral-300">
                            <input type="checkbox" v-model="passwordPolicy.requireLowercase"
                                class="w-4 h-4 rounded border-neutral-700 bg-neutral-800 text-red-500 focus:ring-red-500 focus:ring-offset-neutral-900">
                            <span>{{ t('admin.settings.requireLowercase') }}</span>
                        </label>
                        <label class="flex items-center space-x-3 text-sm text-neutral-300">
                            <input type="checkbox" v-model="passwordPolicy.requireNumbers"
                                class="w-4 h-4 rounded border-neutral-700 bg-neutral-800 text-red-500 focus:ring-red-500 focus:ring-offset-neutral-900">
                            <span>{{ t('admin.settings.requireNumbers') }}</span>
                        </label>
                        <label class="flex items-center space-x-3 text-sm text-neutral-300">
                            <input type="checkbox" v-model="passwordPolicy.requireSymbols"
                                class="w-4 h-4 rounded border-neutral-700 bg-neutral-800 text-red-500 focus:ring-red-500 focus:ring-offset-neutral-900">
                            <span>{{ t('admin.settings.requireSymbols') }}</span>
                        </label>
                    </div>
                </div>
                <div class="flex justify-end mt-6">
                    <button @click="saveSetting('auth.password_policy', passwordPolicy)" :disabled="saving"
                        class="px-4 py-2 bg-red-500/10 text-red-400 border border-red-500/20 rounded-xl hover:bg-red-500/20 font-medium transition disabled:opacity-50">
                        {{ t('admin.settings.savePasswordPolicy') }}
                    </button>
                </div>
            </section>

            <!-- Security & Lockout Policy -->
            <section class="bg-neutral-900/50 border border-neutral-800 rounded-2xl p-6">
                <h3 class="text-lg font-semibold text-white mb-1">{{ t('admin.settings.security') }}</h3>
                <p class="text-sm text-neutral-500 mb-4">{{ t('admin.settings.securitySubtitle') }}</p>
                <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
                    <div>
                        <label class="block text-sm font-medium text-neutral-400 mb-2">{{
                            t('admin.settings.maxFailedAttempts') }}</label>
                        <input type="number" v-model="securityConfig.maxFailedAttempts" min="1" max="100"
                            class="w-full bg-neutral-950/50 border border-neutral-800 rounded-xl px-4 py-2.5 text-neutral-200 focus:ring-2 focus:ring-red-500/50 focus:border-red-500 outline-none transition">
                    </div>
                    <div>
                        <label class="block text-sm font-medium text-neutral-400 mb-2">{{
                            t('admin.settings.lockoutDurationMins') }}</label>
                        <input type="number" v-model="securityConfig.lockoutDurationMins" min="1"
                            class="w-full bg-neutral-950/50 border border-neutral-800 rounded-xl px-4 py-2.5 text-neutral-200 focus:ring-2 focus:ring-red-500/50 focus:border-red-500 outline-none transition">
                    </div>
                    <div>
                        <label class="block text-sm font-medium text-neutral-400 mb-2">{{ t('admin.settings.authMax')
                            }}</label>
                        <input type="number" v-model="securityConfig.authMax" min="1"
                            class="w-full bg-neutral-950/50 border border-neutral-800 rounded-xl px-4 py-2.5 text-neutral-200 focus:ring-2 focus:ring-red-500/50 focus:border-red-500 outline-none transition">
                    </div>
                    <div>
                        <label class="block text-sm font-medium text-neutral-400 mb-2">{{ t('admin.settings.apiMax')
                            }}</label>
                        <input type="number" v-model="securityConfig.apiMax" min="1"
                            class="w-full bg-neutral-950/50 border border-neutral-800 rounded-xl px-4 py-2.5 text-neutral-200 focus:ring-2 focus:ring-red-500/50 focus:border-red-500 outline-none transition">
                    </div>
                </div>
                <div class="flex justify-end mt-6">
                    <button @click="saveSecurityConfig()" :disabled="saving"
                        class="px-4 py-2 bg-red-500/10 text-red-400 border border-red-500/20 rounded-xl hover:bg-red-500/20 font-medium transition disabled:opacity-50">
                        {{ t('admin.settings.saveSecurity') }}
                    </button>
                </div>
            </section>

            <!-- Captcha Settings -->
            <section class="bg-neutral-900/50 border border-neutral-800 rounded-2xl p-6">
                <h3 class="text-lg font-semibold text-white mb-4">{{ t('admin.settings.captcha') }}</h3>
                <div class="space-y-6">
                    <label class="flex items-center space-x-3 text-sm text-neutral-300">
                        <input type="checkbox" v-model="captchaConfig.enabled"
                            class="w-4 h-4 rounded border-neutral-700 bg-neutral-800 text-red-500 focus:ring-red-500 focus:ring-offset-neutral-900">
                        <span>{{ t('admin.settings.enableCaptcha') }}</span>
                    </label>

                    <div v-if="captchaConfig.enabled"
                        class="grid grid-cols-1 gap-6 p-4 rounded-xl bg-neutral-950/30 border border-neutral-800/50">
                        <div>
                            <label class="block text-sm font-medium text-neutral-400 mb-2">{{
                                t('admin.settings.provider') }}</label>
                            <select v-model="captchaConfig.provider"
                                class="w-full bg-neutral-900/50 border border-neutral-800 rounded-xl px-4 py-2.5 text-neutral-200 focus:ring-2 focus:ring-red-500/50 outline-none">
                                <option value="none">None</option>
                                <option value="turnstile">Cloudflare Turnstile</option>
                                <option value="recaptcha">Google reCAPTCHA v2</option>
                            </select>
                        </div>

                        <div v-if="captchaConfig.provider !== 'none'">
                            <label class="block text-sm font-medium text-neutral-400 mb-2">{{
                                t('admin.settings.siteKey') }}</label>
                            <input type="text" v-model="captchaConfig.siteKey"
                                class="w-full bg-neutral-900/50 border border-neutral-800 rounded-xl px-4 py-2.5 text-neutral-200 focus:ring-2 focus:ring-red-500/50 outline-none">
                        </div>
                        <div v-if="captchaConfig.provider !== 'none'">
                            <label class="block text-sm font-medium text-neutral-400 mb-2">{{
                                t('admin.settings.secretKey') }}</label>
                            <input type="password" v-model="captchaConfig.secretKey"
                                class="w-full bg-neutral-900/50 border border-neutral-800 rounded-xl px-4 py-2.5 text-neutral-200 focus:ring-2 focus:ring-red-500/50 outline-none">
                        </div>
                    </div>
                </div>

                <div class="flex justify-end mt-6">
                    <button @click="saveSetting('auth.captcha', captchaConfig)" :disabled="saving"
                        class="px-4 py-2 bg-red-500/10 text-red-400 border border-red-500/20 rounded-xl hover:bg-red-500/20 font-medium transition disabled:opacity-50">
                        {{ t('admin.settings.saveCaptcha') }}
                    </button>
                </div>
            </section>

        </div>
    </div>
</template>

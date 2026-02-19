<script setup lang="ts">
import { ref, onMounted } from 'vue';
import { useI18n } from 'vue-i18n';
import { socket } from '../socket';
import MainLayout from './MainLayout.vue';

const { t } = useI18n();
const settings = ref<any>(null);
const devices = ref<any[]>([]);
const loading = ref(true);

const isSettingUp2FA = ref(false);
const setupSecret = ref('');
const setupCode = ref('');
const errorMsg = ref('');

const fetchData = () => {
    loading.value = true;
    socket.emit('security:settings', (res: any) => {
        if (res.status === 'ok') settings.value = res.settings;
    });
    socket.emit('security:devices', (res: any) => {
        if (res.status === 'ok') devices.value = res.devices;
        loading.value = false;
    });
};

const toggle2FA = () => {
    if (settings.value?.two_factor_enabled) {
        if (confirm("Are you sure you want to disable 2FA?")) {
            socket.emit('security:disable_2fa', (res: any) => {
                if (res.status === 'ok') {
                    settings.value.two_factor_enabled = false;
                }
            });
        }
    } else {
        socket.emit('security:generate_2fa_secret', (res: any) => {
            if (res.status === 'ok') {
                setupSecret.value = res.secret;
                isSettingUp2FA.value = true;
            }
        });
    }
};

const verifyAndEnable2FA = () => {
    errorMsg.value = '';
    socket.emit('security:enable_2fa', { secret: setupSecret.value, code: setupCode.value }, (res: any) => {
        if (res.status === 'ok') {
            settings.value.two_factor_enabled = true;
            isSettingUp2FA.value = false;
            setupCode.value = '';
            setupSecret.value = '';
        } else {
            errorMsg.value = res.message || "Verification failed";
        }
    });
};

const cancelSetup = () => {
    isSettingUp2FA.value = false;
    setupCode.value = '';
    setupSecret.value = '';
};

const revokeDevice = (deviceId: string) => {
    socket.emit('security:revoke_device', { deviceId }, (res: any) => {
        if (res.status === 'ok') {
            devices.value = devices.value.filter(d => d.id !== deviceId);
        }
    });
};

onMounted(() => {
    if (socket.connected) {
        fetchData();
    } else {
        socket.on('connect', fetchData);
    }
});
</script>

<template>
    <MainLayout>
        <div class="security-container">
            <header class="page-header">
                <h1 class="page-title">{{ t('security.security') }}</h1>
            </header>

            <section class="card mb-8">
                <h3 class="section-title">{{ t('security.twoFactorAuth') }}</h3>
                <div class="flex items-center justify-between">
                    <div>
                        <p class="text-muted">{{ t('security.status') }}: 
                            <span :class="settings?.two_factor_enabled ? 'text-success' : 'text-danger'">
                                {{ settings?.two_factor_enabled ? t('security.enabled') : t('security.disabled') }}
                            </span>
                        </p>
                    </div>
                    <button @click="toggle2FA" class="btn" :class="settings?.two_factor_enabled ? 'btn-danger' : 'btn-primary'">
                        {{ settings?.two_factor_enabled ? t('security.disable2FA') : t('security.enable2FA') }}
                    </button>
                </div>

                <!-- 2FA Setup UI -->
                <div v-if="isSettingUp2FA" class="setup-2fa mt-6 p-6 border-2 border-primary-100 rounded-2xl bg-primary-50/30 animate-fade-in">
                    <h4 class="font-bold mb-4">{{ t('security.setupTitle', 'Setup 2FA') }}</h4>
                    <p class="text-sm text-slate-600 mb-4">{{ t('security.setupDesc', 'Enter this secret manually in your Authenticator app:') }}</p>
                    
                    <div class="bg-white p-3 rounded-lg border border-slate-200 font-mono text-center text-lg tracking-widest mb-6">
                        {{ setupSecret }}
                    </div>

                    <div class="space-y-4">
                        <div class="form-group">
                            <label class="text-xs font-bold text-slate-500 uppercase">{{ t('security.verifyCode', 'Verification Code') }}</label>
                            <input type="text" v-model="setupCode" class="glass-input h-12 text-center text-xl" placeholder="000000" maxlength="6" />
                        </div>

                        <div v-if="errorMsg" class="text-danger text-sm text-center font-medium">{{ errorMsg }}</div>

                        <div class="flex gap-2">
                            <button @click="verifyAndEnable2FA" class="btn btn-primary flex-1">{{ t('common.actions.verify', 'Verify & Enable') }}</button>
                            <button @click="cancelSetup" class="btn btn-ghost">{{ t('common.actions.cancel', 'Cancel') }}</button>
                        </div>
                    </div>
                </div>
            </section>

            <section class="card">
                <h3 class="section-title">{{ t('security.devices') }}</h3>
                <div v-if="devices.length === 0" class="empty-state">
                    <p>{{ t('common.noData') }}</p>
                </div>
                <div v-else class="device-list">
                    <div v-for="device in devices" :key="device.id" class="device-item flex justify-between items-center py-4 border-b last:border-0">
                        <div>
                            <p class="font-bold">{{ device.device_name || 'Unknown Device' }}</p>
                            <p class="text-xs text-muted">{{ t('security.lastUsed') }}: {{ new Date(device.last_active).toLocaleString() }}</p>
                        </div>
                        <button v-if="!device.is_current" @click="revokeDevice(device.id)" class="btn btn-sm btn-icon-only text-danger">
                            {{ t('security.revoke') }}
                        </button>
                        <span v-else class="badge badge-success">{{ t('security.currentDevice') }}</span>
                    </div>
                </div>
            </section>
        </div>
    </MainLayout>
</template>

<style scoped>
.mb-8 { margin-bottom: 2rem; }
.text-success { color: var(--success-color); }
.text-danger { color: var(--danger-color); }
.device-item:hover { background-color: var(--bg-hover); }
</style>

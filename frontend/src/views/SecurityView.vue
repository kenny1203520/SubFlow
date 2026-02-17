<script setup lang="ts">
import { ref, onMounted } from 'vue';
import { useI18n } from 'vue-i18n';
import { socket } from '../socket';
import MainLayout from './MainLayout.vue';

const { t } = useI18n();
const settings = ref<any>(null);
const devices = ref<any[]>([]);
const loading = ref(true);

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
                    <button class="btn" :class="settings?.two_factor_enabled ? 'btn-danger' : 'btn-primary'">
                        {{ settings?.two_factor_enabled ? t('security.disable2FA') : t('security.enable2FA') }}
                    </button>
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

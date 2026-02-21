<script setup lang="ts">
import { ref, onMounted } from 'vue';
import { useI18n } from 'vue-i18n';
import http from '../../http';
import { useUIStore } from '../../stores/ui';

const { t } = useI18n();
const ui = useUIStore();

interface IpBlock {
    ip_address: string;
    reason: string | null;
    expires_at: string | null;
    created_at: string;
    created_by_name: string | null;
}

const blocks = ref<IpBlock[]>([]);
const loading = ref(true);

const newIp = ref('');
const newReason = ref('');
const adding = ref(false);

const fetchBlocks = async () => {
    loading.value = true;
    try {
        const res = await http.get('/api/admin/ip-blocks');
        if (res.data.status === 'ok') {
            blocks.value = res.data.blocks;
        }
    } catch (e) {
        ui.alert(t('admin.ip.fetchFailed'));
    } finally {
        loading.value = false;
    }
};

const blockIp = async () => {
    if (!newIp.value.trim()) return;
    adding.value = true;
    try {
        await http.post('/api/admin/ip-blocks', {
            ip: newIp.value.trim(),
            reason: newReason.value.trim() || null
        });
        newIp.value = '';
        newReason.value = '';
        fetchBlocks();
    } catch (e) {
        ui.alert(t('admin.ip.blockFailed'));
    } finally {
        adding.value = false;
    }
};

const unblockIp = async (ip: string) => {
    if (!await ui.confirm(t('admin.ip.confirmUnblock', { ip }))) return;
    try {
        await http.delete(`/api/admin/ip-blocks/${encodeURIComponent(ip)}`);
        fetchBlocks();
    } catch (e) {
        ui.alert(t('admin.ip.unblockFailed'));
    }
};

onMounted(fetchBlocks);
</script>

<template>
    <div class="space-y-6">
        <div>
            <h2 class="text-2xl font-bold bg-gradient-to-r from-red-400 to-rose-600 bg-clip-text text-transparent">
                {{ t('admin.ipBlocks') }}</h2>
            <p class="text-neutral-400 mt-1">{{ t('admin.ip.subtitle') }}</p>
        </div>

        <!-- Add IP Block -->
        <section class="bg-neutral-900/50 border border-neutral-800 rounded-2xl p-6">
            <h3 class="text-lg font-semibold text-white mb-4">{{ t('admin.ip.blockNew') }}</h3>
            <div class="flex flex-wrap gap-4">
                <input type="text" v-model="newIp" :placeholder="t('admin.ip.ipPlaceholder')"
                    class="flex-1 min-w-[200px] bg-neutral-950/50 border border-neutral-800 rounded-xl px-4 py-2.5 text-neutral-200 focus:ring-2 focus:ring-red-500/50 outline-none transition" />
                <input type="text" v-model="newReason" :placeholder="t('admin.ip.reasonPlaceholder')"
                    class="flex-1 min-w-[200px] bg-neutral-950/50 border border-neutral-800 rounded-xl px-4 py-2.5 text-neutral-200 focus:ring-2 focus:ring-red-500/50 outline-none transition" />
                <button @click="blockIp" :disabled="adding || !newIp.trim()"
                    class="px-6 py-2.5 bg-red-500/10 text-red-400 border border-red-500/20 rounded-xl hover:bg-red-500/20 font-medium transition disabled:opacity-50">
                    {{ t('admin.ip.block') }}
                </button>
            </div>
        </section>

        <!-- Block List -->
        <div v-if="loading" class="text-center py-10">
            <div class="w-8 h-8 border-4 border-red-500/20 border-t-red-500 rounded-full animate-spin mx-auto mb-4">
            </div>
        </div>

        <div v-else-if="blocks.length === 0" class="text-center py-10 text-neutral-400">
            {{ t('admin.ip.noBlocks') }}
        </div>

        <div v-else class="bg-neutral-950/50 rounded-2xl border border-neutral-800/50 overflow-hidden">
            <table class="w-full text-left">
                <thead>
                    <tr class="bg-neutral-900/80 border-b border-neutral-800/80 text-sm font-semibold text-neutral-300">
                        <th class="p-4">{{ t('admin.ip.ipAddress') }}</th>
                        <th class="p-4">{{ t('admin.ip.reason') }}</th>
                        <th class="p-4">{{ t('admin.ip.blockedBy') }}</th>
                        <th class="p-4">{{ t('admin.ip.blockedAt') }}</th>
                        <th class="p-4 text-right">{{ t('admin.actions') }}</th>
                    </tr>
                </thead>
                <tbody class="divide-y divide-neutral-800/50">
                    <tr v-for="block in blocks" :key="block.ip_address"
                        class="hover:bg-neutral-800/20 transition-colors">
                        <td class="p-4 font-mono text-white">{{ block.ip_address }}</td>
                        <td class="p-4 text-neutral-400">{{ block.reason || '—' }}</td>
                        <td class="p-4 text-neutral-400">{{ block.created_by_name || '—' }}</td>
                        <td class="p-4 text-sm text-neutral-500">{{ new Date(block.created_at).toLocaleString() }}</td>
                        <td class="p-4 text-right">
                            <button @click="unblockIp(block.ip_address)"
                                class="px-3 py-1.5 text-sm font-medium rounded-lg bg-emerald-500/10 text-emerald-400 hover:bg-emerald-500/20 transition">
                                {{ t('admin.ip.unblock') }}
                            </button>
                        </td>
                    </tr>
                </tbody>
            </table>
        </div>
    </div>
</template>

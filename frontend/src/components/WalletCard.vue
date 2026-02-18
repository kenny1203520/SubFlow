<script setup lang="ts">
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';

const { t } = useI18n();

const props = defineProps<{ wallet: any; loading?: boolean; }>();

const emit = defineEmits(['deposit', 'transfer']);

const formattedBalance = computed(() => {
    return new Intl.NumberFormat('en-US', {
        style: 'currency',
        currency: props.wallet?.currency || 'TWD'
    }).format(props.wallet?.balance || 0);
});

const bgGradient = computed(() => {
    // Different gradients based on currency or type could go here
    return 'bg-gradient-to-br from-slate-800 to-slate-900';
});
</script>

<template>
    <div :class="['rounded-2xl p-6 text-white shadow-xl relative overflow-hidden group', bgGradient]">
        <!-- Decorative blobs -->
        <div
            class="absolute -top-10 -right-10 w-32 h-32 bg-white/10 rounded-full blur-2xl group-hover:scale-110 transition-transform duration-700">
        </div>
        <div class="absolute bottom-0 left-0 w-24 h-24 bg-primary-500/20 rounded-full blur-xl"></div>

        <div class="relative z-10 flex flex-col h-full justify-between min-h-[160px]">
            <div class="flex justify-between items-start">
                <div>
                    <h3 class="text-slate-300 text-sm font-medium uppercase tracking-wider">
                        {{ wallet?.group_id ? 'Group Wallet' : 'Personal Wallet' }}
                    </h3>
                    <p class="text-xs text-slate-400 mt-1 font-mono opacity-70">
                        {{ wallet?.id?.slice(0, 8) }}...
                    </p>
                </div>
                <!-- Currency Badge -->
                <div class="px-2 py-1 bg-white/10 rounded-lg text-xs font-bold border border-white/10 backdrop-blur-sm">
                    {{ wallet?.currency }}
                </div>
            </div>

            <div>
                <div v-if="loading" class="h-10 w-32 bg-white/10 rounded animate-pulse mb-2"></div>
                <h2 v-else class="text-4xl font-bold tracking-tight mb-4">
                    {{ formattedBalance }}
                </h2>

                <div class="flex gap-2 mt-2">
                    <button @click="$emit('deposit', wallet)"
                        class="flex-1 py-2 px-3 bg-white/10 hover:bg-white/20 rounded-lg text-sm font-medium transition-colors flex items-center justify-center gap-2 backdrop-blur-sm border border-white/5">
                        <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24"
                            stroke="currentColor">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
                        </svg>
                        {{ t('wallet.deposit') }}
                    </button>
                    <button @click="$emit('transfer', wallet)"
                        class="flex-1 py-2 px-3 bg-white/10 hover:bg-white/20 rounded-lg text-sm font-medium transition-colors flex items-center justify-center gap-2 backdrop-blur-sm border border-white/5">
                        <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24"
                            stroke="currentColor">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                                d="M8 7h12m0 0l-4-4m4 4l-4 4m0 6H4m0 0l4 4m-4-4l4-4" />
                        </svg>
                        {{ t('wallet.transfer') }}
                    </button>
                </div>
            </div>
        </div>
    </div>
</template>

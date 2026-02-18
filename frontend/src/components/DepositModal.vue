<script setup lang="ts">
import { ref } from 'vue';
import { useI18n } from 'vue-i18n';

const props = defineProps<{
    wallet: any;
}>();

const emit = defineEmits(['close', 'submit']);
const { t } = useI18n();

const amount = ref('');
const isSubmitting = ref(false);

const handleSubmit = () => {
    if (!amount.value || parseFloat(amount.value) <= 0) return;
    isSubmitting.value = true;
    emit('submit', parseFloat(amount.value));
};
</script>

<template>
    <div class="fixed inset-0 bg-black/50 backdrop-blur-sm z-50 flex items-center justify-center p-4 animate-fade-in"
        @click.self="$emit('close')">
        <div
            class="bg-white rounded-2xl shadow-2xl w-full max-w-sm p-6 flex flex-col gap-6 transform transition-all scale-100">
            <header>
                <h2 class="text-xl font-bold text-slate-800">{{ t('wallet.depositTitle', 'Deposit Funds') }}</h2>
                <p class="text-sm text-slate-500">{{ t('wallet.depositDesc', 'Add funds to your wallet') }}</p>
            </header>

            <form @submit.prevent="handleSubmit" class="space-y-4">
                <div>
                    <label class="form-label">{{ t('wallet.amount') }} ({{ wallet.currency }})</label>
                    <div class="relative">
                        <span class="absolute left-3 top-1/2 -translate-y-1/2 text-slate-400 font-bold">$</span>
                        <input v-model="amount" type="number" step="0.01" min="0.01" class="glass-input pl-8"
                            placeholder="0.00" autofocus required />
                    </div>
                </div>

                <div class="flex gap-3 pt-2">
                    <button type="button" @click="$emit('close')"
                        class="btn bg-slate-100 text-slate-600 hover:bg-slate-200 flex-1">
                        {{ t('common.actions.cancel') }}
                    </button>
                    <button type="submit" :disabled="isSubmitting" class="btn btn-primary flex-1">
                        {{ isSubmitting ? t('common.status.saving') : t('wallet.confirmDeposit', 'Confirm') }}
                    </button>
                </div>
            </form>
        </div>
    </div>
</template>

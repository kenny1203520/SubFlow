<script setup lang="ts">
import { ref } from 'vue';
import { socket } from '../socket';
import { useI18n } from 'vue-i18n';

const { t } = useI18n();
const props = defineProps<{
    groupId: string;
    members: any[];
}>();

const emit = defineEmits(['close', 'added']);

const description = ref('');
const totalAmount = ref<number | null>(null);
const isSubmitting = ref(false);
const error = ref('');

// Simplified split: Equal split among all members
const handleSubmit = () => {
    if (!description.value || !totalAmount.value || totalAmount.value <= 0) return;

    isSubmitting.value = true;
    error.value = '';

    const splitAmount = totalAmount.value / props.members.length;
    const splits = props.members.map(m => ({
        memberId: m.member_id,
        amount: splitAmount
    }));

    socket.emit('expense:add', {
        groupId: props.groupId,
        amount: totalAmount.value,
        description: description.value,
        splits
    }, (res: any) => {
        isSubmitting.value = false;
        if (res.status === 'ok') {
            emit('added', res.expense);
        } else {
            error.value = res.message || t('common.status.error');
        }
    });
};
</script>

<template>
    <Transition name="fade" appear>
        <div class="modal-overlay" @click.self="$emit('close')">
            <div class="modal-content glass-panel">
                <h2 class="text-xl font-bold text-slate-800 mb-6">{{ t('groups.addExpense') }}</h2>
                <form @submit.prevent="handleSubmit" class="space-y-6">
                    <div class="form-group">
                        <label class="block text-sm font-medium text-slate-700 mb-1">{{ t('groups.groupDesc') }}</label>
                        <input v-model="description" type="text" :placeholder="t('common.placeholders.description')"
                            required :disabled="isSubmitting" class="glass-input w-full" />
                    </div>
                    <div class="form-group">
                        <label class="block text-sm font-medium text-slate-700 mb-1">{{ t('groups.expenses') }}</label>
                        <div class="relative">
                            <span class="absolute left-3 top-2.5 text-slate-400 font-bold">$</span>
                            <input v-model="totalAmount" type="number" step="0.01"
                                :placeholder="t('common.placeholders.amount')" required :disabled="isSubmitting"
                                class="glass-input w-full pl-8" />
                        </div>
                    </div>

                    <div class="bg-slate-50 p-4 rounded-xl border border-slate-200">
                        <p class="text-sm text-slate-600 mb-1">{{ t('groups.splitEqually', { count: members.length }) }}
                        </p>
                        <p v-if="totalAmount" class="font-bold text-primary-600">
                            {{ t('groups.eachOwes', { amount: '$' + (totalAmount / members.length).toFixed(2) }) }}
                        </p>
                    </div>

                    <p v-if="error" class="text-red-500 text-sm">{{ error }}</p>

                    <div class="flex justify-end gap-3">
                        <button type="button" @click="$emit('close')" :disabled="isSubmitting"
                            class="btn bg-slate-100 text-slate-600 hover:bg-slate-200">
                            {{ t('common.actions.cancel') }}
                        </button>
                        <button type="submit" :disabled="isSubmitting || !totalAmount" class="btn btn-primary">
                            {{ isSubmitting ? t('common.status.loading') : t('groups.addExpense') }}
                        </button>
                    </div>
                </form>
            </div>
        </div>
    </Transition>
</template>

<style scoped>
.modal-overlay {
    position: fixed;
    top: 0;
    left: 0;
    width: 100%;
    height: 100%;
    background: rgba(0, 0, 0, 0.4);
    backdrop-filter: blur(8px);
    display: flex;
    justify-content: center;
    align-items: center;
    z-index: 1000;
}

.modal-content {
    background: white;
    /* Fallback */
    background: var(--glass-bg, rgba(255, 255, 255, 0.9));
    /* Use glass variable if available */
    padding: 2rem;
    width: 100%;
    max-width: 450px;
}

/* Transition Styles */
.fade-enter-active,
.fade-leave-active {
    transition: opacity 0.3s ease;
}

.fade-enter-from,
.fade-leave-to {
    opacity: 0;
}
</style>

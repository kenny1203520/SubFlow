<script setup lang="ts">
import { ref } from 'vue';
import { socket } from '../socket';
import { useI18n } from 'vue-i18n';

const { t } = useI18n();
const emit = defineEmits(['close', 'created']);

const isSubmitting = ref(false);
const error = ref('');

// Form state
const form = ref({
    name: '',
    service_name: '',
    website: '',
    plan_name: '',
    amount: 0,
    service_currency: 'TWD',
    payment_currency: 'TWD',
    billing_type: 'recurring',
    interval_unit: 'month',
    interval_value: 1,
    max_members: 1,
    billing_method: 'equal',
});

const initialMembers = ref<{ type: 'email' | 'name', value: string }[]>([]);
const nextMemberValue = ref('');
const nextMemberType = ref<'email' | 'name'>('name');

const addMember = () => {
    if (!nextMemberValue.value.trim()) return;
    initialMembers.value.push({
        type: nextMemberType.value,
        value: nextMemberValue.value.trim()
    });
    nextMemberValue.value = '';
};

const removeMember = (index: number) => {
    initialMembers.value.splice(index, 1);
};

const handleSubmit = () => {
    if (!form.value.name.trim()) return;

    isSubmitting.value = true;
    error.value = '';

    const payload = {
        ...form.value,
        initial_members: initialMembers.value.map(m => ({
            [m.type]: m.value
        }))
    };

    socket.emit('group:create', payload, (res: any) => {
        isSubmitting.value = false;
        if (res.status === 'ok') {
            emit('created', res.group);
        } else {
            error.value = res.message || t('common.status.error');
        }
    });
};
</script>

<template>
    <div class="modal-overlay animate-fade-in" @click.self="$emit('close')">
        <div class="modal-content glass-panel">
            <h2 class="text-2xl font-bold text-slate-800 mb-6">{{ t('groups.createGroup') }}</h2>
            <form @submit.prevent="handleSubmit" class="scrollable-form space-y-8">
                <!-- Basic Group Info -->
                <div class="section">
                    <h3 class="text-sm font-bold text-slate-500 uppercase tracking-wider mb-4">{{ t('groups.basicInfo')
                    }}</h3>
                    <div class="form-group">
                        <label for="groupName" class="block text-sm font-medium text-slate-700 mb-1">{{
                            t('groups.groupName') }} *</label>
                        <input id="groupName" v-model="form.name" type="text"
                            :placeholder="t('common.placeholders.name')" required :disabled="isSubmitting"
                            class="glass-input w-full" />
                    </div>
                </div>

                <!-- Service Details -->
                <div class="section">
                    <h3 class="text-sm font-bold text-slate-500 uppercase tracking-wider mb-4">{{
                        t('groups.serviceDetails') }}</h3>
                    <div class="grid grid-cols-2 gap-4">
                        <div class="form-group">
                            <label class="block text-sm font-medium text-slate-700 mb-1">{{ t('groups.serviceName')
                            }}</label>
                            <input v-model="form.service_name" type="text"
                                :placeholder="t('common.placeholders.service')" class="glass-input w-full" />
                        </div>
                        <div class="form-group">
                            <label class="block text-sm font-medium text-slate-700 mb-1">{{ t('groups.website')
                            }}</label>
                            <input v-model="form.website" type="url" placeholder="https://..."
                                class="glass-input w-full" />
                        </div>
                    </div>
                </div>

                <!-- Plan Details -->
                <div class="section">
                    <h3 class="text-sm font-bold text-slate-500 uppercase tracking-wider mb-4">{{
                        t('groups.planAndBilling') }}</h3>
                    <div class="grid grid-cols-2 gap-4 mb-4">
                        <div class="form-group">
                            <label class="block text-sm font-medium text-slate-700 mb-1">{{ t('groups.planName')
                            }}</label>
                            <input v-model="form.plan_name" type="text" :placeholder="t('common.placeholders.plan')"
                                class="glass-input w-full" />
                        </div>
                        <div class="form-group">
                            <label class="block text-sm font-medium text-slate-700 mb-1">{{ t('groups.billingCycle')
                            }}</label>
                            <select v-model="form.interval_unit" class="glass-input w-full">
                                <option value="month">{{ t('common.cycles.month') }}</option>
                                <option value="year">{{ t('common.cycles.year') }}</option>
                                <option value="day">{{ t('common.cycles.day') }}</option>
                            </select>
                        </div>
                    </div>
                    <div class="grid grid-cols-3 gap-4">
                        <div class="form-group col-span-2">
                            <label class="block text-sm font-medium text-slate-700 mb-1">{{ t('groups.amount')
                            }}</label>
                            <input v-model.number="form.amount" type="number" step="0.01" class="glass-input w-full" />
                        </div>
                        <div class="form-group">
                            <label class="block text-sm font-medium text-slate-700 mb-1">{{ t('groups.currency')
                            }}</label>
                            <input v-model="form.service_currency" type="text" maxlength="3"
                                class="glass-input w-full uppercase" />
                        </div>
                    </div>
                </div>

                <!-- Group Settings -->
                <div class="section">
                    <h3 class="text-sm font-bold text-slate-500 uppercase tracking-wider mb-4">{{ t('groups.settings')
                    }}</h3>
                    <div class="grid grid-cols-2 gap-4">
                        <div class="form-group">
                            <label class="block text-sm font-medium text-slate-700 mb-1">{{ t('groups.maxMembers')
                            }}</label>
                            <input v-model.number="form.max_members" type="number" min="1" class="glass-input w-full" />
                        </div>
                        <div class="form-group">
                            <label class="block text-sm font-medium text-slate-700 mb-1">{{ t('groups.billingMethod')
                            }}</label>
                            <select v-model="form.billing_method" class="glass-input w-full">
                                <option value="equal">{{ t('groups.methods.equal') }}</option>
                                <option value="fixed">{{ t('groups.methods.fixed') }}</option>
                                <option value="percentage">{{ t('groups.methods.percentage') }}</option>
                            </select>
                        </div>
                    </div>
                </div>

                <!-- Initial Members -->
                <div class="section w-full">
                    <h3 class="text-sm font-bold text-slate-500 uppercase tracking-wider mb-4">{{
                        t('groups.initialMembers') }}</h3>
                    <div class="flex gap-2 mb-3 w-full">
                        <select v-model="nextMemberType" class="glass-input w-32">
                            <option value="name">{{ t('common.fields.name') }}</option>
                            <option value="email">{{ t('common.fields.email') }}</option>
                        </select>
                        <input v-model="nextMemberValue" type="text" :placeholder="t('common.placeholders.member')"
                            @keydown.enter.prevent="addMember" class="glass-input flex-1" />
                        <button type="button" @click="addMember" class="btn btn-secondary px-4">+</button>
                    </div>
                    <div class="flex flex-wrap gap-2">
                        <div v-for="(m, idx) in initialMembers" :key="idx"
                            class="px-3 py-1 rounded-full bg-primary-50 text-primary-700 border border-primary-100 text-sm flex items-center gap-2">
                            <span>{{ m.value }}</span>
                            <button @click="removeMember(idx)" type="button"
                                class="text-primary-400 hover:text-primary-700">&times;</button>
                        </div>
                    </div>
                </div>

                <p v-if="error" class="text-red-500 text-sm">{{ error }}</p>

                <div class="flex justify-end gap-3 pt-4 border-t border-slate-200">
                    <button type="button" @click="$emit('close')" :disabled="isSubmitting"
                        class="btn bg-slate-100 text-slate-600 hover:bg-slate-200">
                        {{ t('common.actions.cancel') }}
                    </button>
                    <button type="submit" :disabled="isSubmitting || !form.name.trim()" class="btn btn-primary">
                        {{ isSubmitting ? t('common.status.loading') : t('groups.createGroup') }}
                    </button>
                </div>
            </form>
        </div>
    </div>
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
    padding: 2.5rem;
    width: 90%;
    max-width: 650px;
    max-height: 90vh;
    display: flex;
    flex-direction: column;
}

.scrollable-form {
    overflow-y: auto;
    padding-right: 0.5rem;
}

/* Custom Scrollbar for form */
.scrollable-form::-webkit-scrollbar {
    width: 6px;
}

.scrollable-form::-webkit-scrollbar-track {
    background: transparent;
}

.scrollable-form::-webkit-scrollbar-thumb {
    background-color: rgba(0, 0, 0, 0.1);
    border-radius: 3px;
}

.section {
    border-bottom: 1px dashed var(--border-color);
    padding-bottom: 2rem;
}

.section:last-of-type {
    border-bottom: none;
    padding-bottom: 0;
}
</style>

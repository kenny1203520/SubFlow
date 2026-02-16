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
    currency: 'TWD',
    billing_cycle: 'monthly',
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
    <div class="modal-overlay" @click.self="$emit('close')">
        <div class="modal-content card">
            <h2>{{ t('groups.createGroup') }}</h2>
            <form @submit.prevent="handleSubmit" class="scrollable-form">
                <!-- Basic Group Info -->
                <div class="section">
                    <h3>{{ t('groups.basicInfo') }}</h3>
                    <div class="form-group">
                        <label for="groupName">{{ t('groups.groupName') }} *</label>
                        <input id="groupName" v-model="form.name" type="text"
                            :placeholder="t('common.placeholders.name')" required :disabled="isSubmitting" />
                    </div>
                </div>

                <!-- Service Details -->
                <div class="section">
                    <h3>{{ t('groups.serviceDetails') }}</h3>
                    <div class="grid grid-cols-2 gap-4">
                        <div class="form-group">
                            <label>{{ t('groups.serviceName') }}</label>
                            <input v-model="form.service_name" type="text"
                                :placeholder="t('common.placeholders.service')" />
                        </div>
                        <div class="form-group">
                            <label>{{ t('groups.website') }}</label>
                            <input v-model="form.website" type="url" placeholder="https://..." />
                        </div>
                    </div>
                </div>

                <!-- Plan Details -->
                <div class="section">
                    <h3>{{ t('groups.planAndBilling') }}</h3>
                    <div class="grid grid-cols-2 gap-4">
                        <div class="form-group">
                            <label>{{ t('groups.planName') }}</label>
                            <input v-model="form.plan_name" type="text" :placeholder="t('common.placeholders.plan')" />
                        </div>
                        <div class="form-group">
                            <label>{{ t('groups.billingCycle') }}</label>
                            <select v-model="form.billing_cycle">
                                <option value="monthly">{{ t('common.cycles.monthly') }}</option>
                                <option value="yearly">{{ t('common.cycles.yearly') }}</option>
                                <option value="one-time">{{ t('common.cycles.one-time') }}</option>
                            </select>
                        </div>
                    </div>
                    <div class="grid grid-cols-3 gap-4">
                        <div class="form-group col-span-2">
                            <label>{{ t('groups.amount') }}</label>
                            <input v-model.number="form.amount" type="number" step="0.01" />
                        </div>
                        <div class="form-group">
                            <label>{{ t('groups.currency') }}</label>
                            <input v-model="form.currency" type="text" maxlength="3" />
                        </div>
                    </div>
                </div>

                <!-- Group Settings -->
                <div class="section">
                    <h3>{{ t('groups.settings') }}</h3>
                    <div class="grid grid-cols-2 gap-4">
                        <div class="form-group">
                            <label>{{ t('groups.maxMembers') }}</label>
                            <input v-model.number="form.max_members" type="number" min="1" />
                        </div>
                        <div class="form-group">
                            <label>{{ t('groups.billingMethod') }}</label>
                            <select v-model="form.billing_method">
                                <option value="equal">{{ t('groups.methods.equal') }}</option>
                                <option value="fixed">{{ t('groups.methods.fixed') }}</option>
                                <option value="percentage">{{ t('groups.methods.percentage') }}</option>
                            </select>
                        </div>
                    </div>
                </div>

                <!-- Initial Members -->
                <div class="section">
                    <h3>{{ t('groups.initialMembers') }}</h3>
                    <div class="member-input-row">
                        <select v-model="nextMemberType" class="type-select">
                            <option value="name">{{ t('common.fields.name') }}</option>
                            <option value="email">{{ t('common.fields.email') }}</option>
                        </select>
                        <input v-model="nextMemberValue" type="text" :placeholder="t('common.placeholders.member')"
                            @keydown.enter.prevent="addMember" />
                        <button type="button" @click="addMember" class="btn btn-sm btn-secondary">+</button>
                    </div>
                    <div class="member-tags">
                        <div v-for="(m, idx) in initialMembers" :key="idx" class="member-tag">
                            <span>{{ m.value }}</span>
                            <button @click="removeMember(idx)" type="button">&times;</button>
                        </div>
                    </div>
                </div>

                <p v-if="error" class="error-msg">{{ error }}</p>

                <div class="actions">
                    <button type="button" @click="$emit('close')" :disabled="isSubmitting" class="btn btn-secondary">
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
    background: rgba(0, 0, 0, 0.6);
    backdrop-filter: blur(4px);
    display: flex;
    justify-content: center;
    align-items: center;
    z-index: 1000;
}

.modal-content {
    background: var(--bg-surface);
    padding: 2rem;
    border-radius: var(--radius-lg);
    width: 100%;
    max-width: 600px;
    max-height: 90vh;
    display: flex;
    flex-direction: column;
    box-shadow: 0 20px 25px -5px rgba(0, 0, 0, 0.1), 0 10px 10px -5px rgba(0, 0, 0, 0.04);
}

.scrollable-form {
    overflow-y: auto;
    padding-right: 0.5rem;
}

h2 {
    margin-top: 0;
    margin-bottom: 1.5rem;
    font-size: 1.5rem;
    font-weight: 700;
}

.section {
    margin-bottom: 2rem;
    padding-bottom: 1rem;
    border-bottom: 1px solid var(--border-color);
}

.section:last-of-type {
    border-bottom: none;
}

h3 {
    font-size: 0.875rem;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: var(--text-muted);
    margin-bottom: 1rem;
}

.form-group {
    margin-bottom: 1.25rem;
}

label {
    display: block;
    margin-bottom: 0.375rem;
    font-size: 0.875rem;
    font-weight: 500;
    color: var(--text-main);
}

input,
select {
    width: 100%;
    padding: 0.625rem;
    border: 1px solid var(--border-color);
    border-radius: var(--radius-md);
    background-color: var(--bg-body);
    color: var(--text-main);
    font-size: 0.9375rem;
    transition: border-color 0.2s;
}

input:focus,
select:focus {
    outline: none;
    border-color: var(--primary-color);
    box-shadow: 0 0 0 3px rgba(99, 102, 241, 0.1);
}

.member-input-row {
    display: flex;
    gap: 0.5rem;
    margin-bottom: 1rem;
}

.type-select {
    width: 100px;
}

.member-tags {
    display: flex;
    flex-wrap: wrap;
    gap: 0.5rem;
}

.member-tag {
    background-color: var(--primary-50);
    color: var(--primary-700);
    padding: 0.25rem 0.75rem;
    border-radius: 9999px;
    display: flex;
    align-items: center;
    gap: 0.5rem;
    font-size: 0.8125rem;
    font-weight: 500;
    border: 1px solid var(--primary-100);
}

.member-tag button {
    background: none;
    border: none;
    color: inherit;
    cursor: pointer;
    font-size: 1.1rem;
    line-height: 1;
    padding: 0;
}

.error-msg {
    color: var(--danger-color);
    margin-bottom: 1rem;
    font-size: 0.875rem;
}

.actions {
    display: flex;
    justify-content: flex-end;
    gap: 1rem;
    margin-top: 1rem;
    position: sticky;
    bottom: 0;
    background: var(--bg-surface);
    padding-top: 1rem;
}

.grid {
    display: grid;
}

.grid-cols-2 {
    grid-template-columns: repeat(2, minmax(0, 1fr));
}

.grid-cols-3 {
    grid-template-columns: repeat(3, minmax(0, 1fr));
}

.col-span-2 {
    grid-column: span 2 / span 2;
}

.gap-4 {
    gap: 1rem;
}
</style>

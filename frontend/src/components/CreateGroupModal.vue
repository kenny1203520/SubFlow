<script setup lang="ts">
import { ref, computed } from 'vue';
import { socket } from '../socket';
import { useI18n } from 'vue-i18n';
import debounce from 'lodash.debounce';

const { t } = useI18n();
const emit = defineEmits(['close', 'created']);

const isSubmitting = ref(false);
const error = ref('');

// Service Search State
const serviceQuery = ref('');
const foundServices = ref<any[]>([]);
const showServiceDropdown = ref(false);
const isSearching = ref(false);

// Form state
const form = ref({
    name: '',
    service_name: '',
    service_id: '',
    website: '',
    icon_url: '',
    plan_name: '',
    amount: 0,
    service_currency: 'TWD',
    payment_currency: 'TWD',
    billing_type: 'recurring',
    interval_unit: 'month',
    interval_value: 1,
    max_members: 1,
    billing_method: 'equal',
    start_date: '',
    end_condition: 'indefinite',
    end_value: ''
});

const initialMembers = ref<{ type: 'email' | 'name', value: string }[]>([]);
const nextMemberValue = ref('');
const nextMemberType = ref<'email' | 'name'>('name');

// Service Search Logic
const searchServices = debounce((query: string) => {
    if (!query.trim()) {
        foundServices.value = [];
        return;
    }
    isSearching.value = true;
    socket.emit('service:search', { query }, (res: any) => {
        isSearching.value = false;
        if (res.status === 'ok') {
            foundServices.value = res.data.services;
        }
    });
}, 300);

const onServiceInput = () => {
    form.value.service_name = serviceQuery.value;
    form.value.service_id = ''; // Reset ID if typing manually
    showServiceDropdown.value = true;
    searchServices(serviceQuery.value);
};

const selectService = (service: any) => {
    serviceQuery.value = service.name;
    form.value.service_name = service.name;
    form.value.service_id = service.id;
    form.value.website = service.domain ? `https://${service.domain}` : '';
    form.value.icon_url = service.icon_url || '';
    showServiceDropdown.value = false;
};

const handleServiceBlur = () => {
    setTimeout(() => {
        showServiceDropdown.value = false;
    }, 200);
};

// Icon Logic
const iconPreviewUrl = computed(() => {
    if (form.value.icon_url) return form.value.icon_url;
    // Fallback or generic icon could go here
    return null;
});

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
        service_name: serviceQuery.value || form.value.service_name, // Ensure service name is captured
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
            <div class="flex justify-between items-center mb-6">
                <h2 class="text-2xl font-bold text-slate-800">{{ t('groups.createGroup') }}</h2>
                <button @click="$emit('close')" class="text-slate-400 hover:text-slate-600 transition-colors">
                    <svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6" fill="none" viewBox="0 0 24 24"
                        stroke="currentColor">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                            d="M6 18L18 6M6 6l12 12" />
                    </svg>
                </button>
            </div>

            <form @submit.prevent="handleSubmit" class="scrollable-form space-y-8 pr-2">
                <!-- 1. Service & Identity -->
                <div class="section">
                    <h3 class="section-title">{{ t('groups.serviceDetails') }}</h3>

                    <!-- Service Search & Icon -->
                    <div class="flex gap-4 items-start mb-4">
                        <div class="flex-shrink-0">
                            <div
                                class="w-16 h-16 rounded-xl bg-slate-100 border border-slate-200 flex items-center justify-center overflow-hidden shadow-sm relative group">
                                <img v-if="iconPreviewUrl" :src="iconPreviewUrl" alt="Icon"
                                    class="w-full h-full object-cover" />
                                <span v-else class="text-2xl text-slate-300">#</span>

                                <!-- Hover to edit icon URL -->
                                <div
                                    class="absolute inset-0 bg-black/50 opacity-0 group-hover:opacity-100 transition-opacity flex items-center justify-center">
                                    <span class="text-xs text-white font-medium">Edit</span>
                                </div>
                            </div>
                        </div>

                        <div class="flex-1 space-y-4">
                            <!-- Service Search -->
                            <div class="relative">
                                <label class="field-label">{{ t('groups.serviceName') }} *</label>
                                <input v-model="serviceQuery" @input="onServiceInput"
                                    @focus="showServiceDropdown = true" @blur="handleServiceBlur" type="text"
                                    class="glass-input w-full" :placeholder="t('groups.serviceSearchPlaceholder')"
                                    required />

                                <!-- Dropdown -->
                                <div v-if="showServiceDropdown && (foundServices.length > 0 || isSearching)"
                                    class="absolute z-50 w-full mt-1 bg-white border border-slate-200 rounded-lg shadow-lg max-h-60 overflow-y-auto">
                                    <div v-if="isSearching" class="p-3 text-sm text-slate-400">Searching...</div>
                                    <ul v-else>
                                        <li v-for="svc in foundServices" :key="svc.id" @click="selectService(svc)"
                                            class="px-4 py-2 hover:bg-primary-50 cursor-pointer flex items-center gap-3 transition-colors">
                                            <img v-if="svc.icon_url" :src="svc.icon_url"
                                                class="w-6 h-6 rounded object-cover" />
                                            <span class="text-slate-700">{{ svc.name }}</span>
                                        </li>
                                    </ul>
                                </div>
                            </div>

                            <!-- Website & Custom Icon URL (Collapsible or always visible?) -->
                            <div class="grid grid-cols-2 gap-4">
                                <div>
                                    <label class="field-label">{{ t('groups.website') }}</label>
                                    <input v-model="form.website" type="url" placeholder="https://..."
                                        class="glass-input w-full" />
                                </div>
                                <div>
                                    <label class="field-label">{{ t('groups.iconUrl') }}</label>
                                    <input v-model="form.icon_url" type="url" placeholder="https://..."
                                        class="glass-input w-full" />
                                </div>
                            </div>
                        </div>
                    </div>

                    <!-- Group Name -->
                    <div class="form-group">
                        <label class="field-label">{{ t('groups.groupName') }} *</label>
                        <input v-model="form.name" type="text" :placeholder="t('common.placeholders.name')" required
                            class="glass-input w-full" />
                    </div>
                </div>

                <!-- 2. Plan & Billing -->
                <div class="section">
                    <h3 class="section-title">{{ t('groups.planAndBilling') }}</h3>
                    <div class="grid grid-cols-2 gap-4 mb-4">
                        <div>
                            <label class="field-label">{{ t('groups.planName') }}</label>
                            <input v-model="form.plan_name" type="text" :placeholder="t('common.placeholders.plan')"
                                class="glass-input w-full" />
                        </div>
                        <div class="grid grid-cols-2 gap-2">
                            <div>
                                <label class="field-label">{{ t('groups.amount') }}</label>
                                <input v-model.number="form.amount" type="number" step="0.01"
                                    class="glass-input w-full" />
                            </div>
                            <div>
                                <label class="field-label">{{ t('groups.currency') }}</label>
                                <input v-model="form.service_currency" type="text" maxlength="3"
                                    class="glass-input w-full uppercase" />
                            </div>
                        </div>
                    </div>

                    <div class="grid grid-cols-2 gap-4">
                        <div>
                            <label class="field-label">{{ t('groups.billingCycle') }}</label>
                            <div class="flex gap-2">
                                <input v-model.number="form.interval_value" type="number" min="1"
                                    class="glass-input w-20 text-center" />
                                <select v-model="form.interval_unit" class="glass-input flex-1">
                                    <option value="month">{{ t('common.cycles.month') }}</option>
                                    <option value="year">{{ t('common.cycles.year') }}</option>
                                    <option value="day">{{ t('common.cycles.day') }}</option>
                                </select>
                            </div>
                        </div>
                        <div>
                            <label class="field-label">{{ t('groups.billingMethod') }}</label>
                            <select v-model="form.billing_method" class="glass-input w-full">
                                <option value="equal">{{ t('groups.methods.equal') }}</option>
                                <option value="fixed">{{ t('groups.methods.fixed') }}</option>
                                <option value="percentage">{{ t('groups.methods.percentage') }}</option>
                            </select>
                        </div>
                    </div>
                </div>

                <!-- 3. Duration & Dates (NEW) -->
                <div class="section">
                    <h3 class="section-title">Duration & Limits</h3>
                    <div class="grid grid-cols-2 gap-4">
                        <div>
                            <label class="field-label">{{ t('groups.startDate') }}</label>
                            <input v-model="form.start_date" type="date" class="glass-input w-full" />
                        </div>
                        <div>
                            <label class="field-label">{{ t('groups.maxMembers') }}</label>
                            <input v-model.number="form.max_members" type="number" min="1" class="glass-input w-full" />
                        </div>
                    </div>
                    <div class="grid grid-cols-2 gap-4 mt-4">
                        <div>
                            <label class="field-label">{{ t('groups.endCondition') }}</label>
                            <select v-model="form.end_condition" class="glass-input w-full">
                                <option value="indefinite">{{ t('groups.endConditions.indefinite') }}</option>
                                <option value="date">{{ t('groups.endConditions.date') }}</option>
                                <option value="total_amount">{{ t('groups.endConditions.total_amount') }}</option>
                            </select>
                        </div>
                        <div v-if="form.end_condition !== 'indefinite'">
                            <label class="field-label">{{ t('groups.endValue') }}</label>
                            <input v-if="form.end_condition === 'date'" v-model="form.end_value" type="date"
                                class="glass-input w-full" />
                            <input v-else v-model="form.end_value" type="number" class="glass-input w-full"
                                placeholder="Total Amount" />
                        </div>
                    </div>
                </div>

                <!-- 4. Initial Members -->
                <div class="section border-none">
                    <h3 class="section-title">{{ t('groups.initialMembers') }}</h3>
                    <div class="flex gap-2 mb-3">
                        <select v-model="nextMemberType" class="glass-input w-32">
                            <option value="name">{{ t('common.fields.name') }}</option>
                            <option value="email">{{ t('common.fields.email') }}</option>
                        </select>
                        <input v-model="nextMemberValue" type="text" :placeholder="t('common.placeholders.member')"
                            @keydown.enter.prevent="addMember" class="glass-input flex-1" />
                        <button type="button" @click="addMember" class="btn btn-secondary px-4">
                            <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" viewBox="0 0 20 20"
                                fill="currentColor">
                                <path fill-rule="evenodd"
                                    d="M10 3a1 1 0 011 1v5h5a1 1 0 110 2h-5v5a1 1 0 11-2 0v-5H4a1 1 0 110-2h5V4a1 1 0 011-1z"
                                    clip-rule="evenodd" />
                            </svg>
                        </button>
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

                <p v-if="error" class="text-red-500 text-sm bg-red-50 p-2 rounded">{{ error }}</p>

                <div class="flex justify-end gap-3 pt-6 border-t border-slate-200">
                    <button type="button" @click="$emit('close')" :disabled="isSubmitting"
                        class="btn bg-slate-100 text-slate-600 hover:bg-slate-200">
                        {{ t('common.actions.cancel') }}
                    </button>
                    <button type="submit" :disabled="isSubmitting || !form.name.trim()"
                        class="btn btn-primary min-w-[120px]">
                        <span v-if="isSubmitting" class="flex items-center gap-2">
                            <svg class="animate-spin h-4 w-4 text-white" xmlns="http://www.w3.org/2000/svg" fill="none"
                                viewBox="0 0 24 24">
                                <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor"
                                    stroke-width="4"></circle>
                                <path class="opacity-75" fill="currentColor"
                                    d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z">
                                </path>
                            </svg>
                            {{ t('common.status.loading') }}
                        </span>
                        <span v-else>{{ t('groups.createGroup') }}</span>
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
    background: rgba(15, 23, 42, 0.4);
    /* Slate-900 with opacity */
    backdrop-filter: blur(8px);
    display: flex;
    justify-content: center;
    align-items: center;
    z-index: 1000;
}

.modal-content {
    width: 95%;
    max-width: 700px;
    max-height: 90vh;
    display: flex;
    flex-direction: column;
    padding: 2rem;
    border-radius: 1.5rem;
    box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.25);
}

.section {
    border-bottom: 1px dashed var(--border-color);
    padding-bottom: 1.5rem;
}

.section.border-none {
    border-bottom: none;
    padding-bottom: 0;
}

.section-title {
    font-size: 0.75rem;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: var(--text-secondary);
    margin-bottom: 1rem;
}

.field-label {
    display: block;
    font-size: 0.875rem;
    font-weight: 500;
    color: var(--text-secondary);
    margin-bottom: 0.25rem;
}

.scrollable-form {
    overflow-y: auto;
    padding-right: 0.5rem;
}

/* Custom Scrollbar */
.scrollable-form::-webkit-scrollbar {
    width: 6px;
}

.scrollable-form::-webkit-scrollbar-track {
    background: transparent;
}

.scrollable-form::-webkit-scrollbar-thumb {
    background-color: var(--border-color);
    border-radius: 3px;
}
</style>

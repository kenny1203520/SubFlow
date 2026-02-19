<script setup lang="ts">
import { ref, computed } from 'vue';
import { socket } from '../socket';
import { useI18n } from 'vue-i18n';
import debounce from 'lodash.debounce';

const { t } = useI18n();
const emit = defineEmits(['close', 'created']);

const isSubmitting = ref(false);
const error = ref('');
const step = ref(1); // 1: Basic Info, 2: Service & Billing

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
    end_condition: 'never',
    end_value: ''
});

const initialMembers = ref<{ type: 'email' | 'name' | 'username', value: string }[]>([]);
const nextMemberValue = ref('');
const nextMemberType = ref<'email' | 'name' | 'username'>('name');

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
    if (form.value.website) {
        return `https://unavatar.io/${form.value.website}`;
    }
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

const nextStep = () => {
    if (!form.value.name.trim()) {
        error.value = t('common.validation.required', { field: t('groups.groupName') });
        return;
    }
    error.value = '';
    step.value = 2;
};

const handleSubmit = () => {
    isSubmitting.value = true;
    error.value = '';

    const payload = {
        ...form.value,
        service_name: serviceQuery.value || form.value.service_name,
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
    <Teleport to="body">
        <Transition name="fade" appear>
            <div class="modal-overlay" @click.self="$emit('close')">
                <div class="modal-content glass-panel">
                    <div class="flex justify-between items-center mb-6">
                        <h2 class="text-2xl font-bold text-slate-800">
                            {{ step === 1 ? t('groups.createGroup') : t('groups.serviceDetails') }}
                        </h2>
                        <button @click="$emit('close')" class="text-slate-400 hover:text-slate-600 transition-colors">
                            <svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6" fill="none" viewBox="0 0 24 24"
                                stroke="currentColor">
                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                                    d="M6 18L18 6M6 6l12 12" />
                            </svg>
                        </button>
                    </div>

                    <div class="mb-6 flex gap-2">
                        <div class="h-1 flex-1 rounded-full transition-colors duration-300"
                            :class="step >= 1 ? 'bg-primary-500' : 'bg-slate-200'"></div>
                        <div class="h-1 flex-1 rounded-full transition-colors duration-300"
                            :class="step >= 2 ? 'bg-primary-500' : 'bg-slate-200'"></div>
                    </div>

                    <form @submit.prevent="handleSubmit" class="scrollable-form space-y-6 pr-2">

                        <!-- STEP 1: Basic Info & Members -->
                        <div v-show="step === 1" class="space-y-6 animate-fade-in">
                            <!-- Group Name -->
                            <div class="form-group">
                                <label class="field-label">{{ t('groups.groupName') }} <span
                                        class="text-red-500">*</span></label>
                                <input v-model="form.name" type="text" :placeholder="t('common.placeholders.name')"
                                    class="glass-input w-full text-lg" autofocus />
                            </div>

                            <!-- Initial Members -->
                            <div class="form-group">
                                <label class="field-label mb-2">{{ t('groups.initialMembers') }}</label>

                                <div class="bg-slate-50 p-4 rounded-xl border border-slate-200 mb-2">
                                    <div class="flex gap-2 mb-3">
                                        <select v-model="nextMemberType" class="glass-input w-36 bg-white">
                                            <option value="name">{{ t('groups.tempMember') }}</option>
                                            <option value="email">{{ t('groups.inviteEmail') }}</option>
                                            <option value="username">{{ t('groups.inviteUsername') }}</option>
                                        </select>
                                        <input v-model="nextMemberValue" type="text"
                                            :placeholder="nextMemberType === 'name' ? 'e.g. Mom' : (nextMemberType === 'email' ? 'user@example.com' : 'username')"
                                            @keydown.enter.prevent="addMember" class="glass-input flex-1 bg-white" />
                                        <button type="button" @click="addMember" class="btn btn-primary px-4">
                                            +
                                        </button>
                                    </div>
                                    <p class="text-xs text-slate-500 mb-0">
                                        {{ nextMemberType === 'name'
                                            ? t('groups.tempMemberDesc',
                                                'Create a placeholder member. Can be bound to a real user later.')
                                            : (nextMemberType === 'email'
                                                ? t('groups.inviteEmailDesc',
                                                    'Send an invitation to a registered user directly.')
                                                : t('groups.inviteUsernameDesc', 'Invite a user by their username.'))
                                        }}
                                    </p>
                                </div>

                                <div class="flex flex-wrap gap-2 min-h-[40px]">
                                    <div v-for="(m, idx) in initialMembers" :key="idx"
                                        class="px-3 py-1.5 rounded-full text-sm flex items-center gap-2 border shadow-sm"
                                        :class="m.type === 'email' || m.type === 'username' ? 'bg-indigo-50 text-indigo-700 border-indigo-100' : 'bg-slate-50 text-slate-700 border-slate-200'">
                                        <span class="font-medium">{{ m.value }}</span>
                                        <span v-if="m.type === 'email' || m.type === 'username'"
                                            class="text-[10px] uppercase bg-white/50 px-1 rounded">Invite</span>
                                        <button @click="removeMember(idx)" type="button"
                                            class="text-slate-400 hover:text-red-500 transition-colors ml-1">&times;</button>
                                    </div>
                                    <div v-if="initialMembers.length === 0" class="text-slate-400 text-sm italic py-2">
                                        {{ t('groups.noMembersYet', 'No members added yet.') }}
                                    </div>
                                </div>
                            </div>
                        </div>

                        <!-- STEP 2: Service & Billing -->
                        <div v-show="step === 2" class="space-y-6 animate-fade-in">
                            <div class="bg-blue-50 text-blue-700 p-3 rounded-lg text-sm flex items-start gap-2">
                                <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5 flex-shrink-0"
                                    viewBox="0 0 20 20" fill="currentColor">
                                    <path fill-rule="evenodd"
                                        d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a1 1 0 000 2v3a1 1 0 001 1h1a1 1 0 100-2v-3a1 1 0 00-1-1H9z"
                                        clip-rule="evenodd" />
                                </svg>
                                {{ t('groups.optionalStep',
                                    'This step is optional. You can add service details later.') }}
                            </div>

                            <!-- Service Search & Icon -->
                            <div class="flex gap-4 items-start">
                                <div class="flex-shrink-0">
                                    <div
                                        class="w-16 h-16 rounded-xl bg-slate-100 border border-slate-200 flex items-center justify-center overflow-hidden shadow-sm relative group">
                                        <img v-if="iconPreviewUrl" :src="iconPreviewUrl" alt="Icon"
                                            @error="(e: any) => e.target.style.display = 'none'"
                                            class="w-full h-full object-cover" />
                                        <span v-else class="text-2xl text-slate-300">#</span>
                                    </div>
                                </div>

                                <div class="flex-1">
                                    <label class="field-label">{{ t('groups.serviceName') }}</label>
                                    <div class="relative">
                                        <input v-model="serviceQuery" @input="onServiceInput"
                                            @focus="showServiceDropdown = true" @blur="handleServiceBlur" type="text"
                                            class="glass-input w-full"
                                            :placeholder="t('groups.serviceSearchPlaceholder')" />

                                        <!-- Dropdown -->
                                        <div v-if="showServiceDropdown && (foundServices.length > 0 || isSearching)"
                                            class="absolute z-50 w-full mt-1 bg-white border border-slate-200 rounded-lg shadow-lg max-h-60 overflow-y-auto">
                                            <div v-if="isSearching" class="p-3 text-sm text-slate-400">{{
                                                t('common.status.searching') }}</div>
                                            <ul v-else>
                                                <li v-for="svc in foundServices" :key="svc.id"
                                                    @click="selectService(svc)"
                                                    class="px-4 py-2 hover:bg-primary-50 cursor-pointer flex items-center gap-3 transition-colors">
                                                    <img v-if="svc.icon_url" :src="svc.icon_url"
                                                        class="w-6 h-6 rounded object-cover" />
                                                    <span class="text-slate-700">{{ svc.name }}</span>
                                                </li>
                                            </ul>
                                        </div>
                                    </div>
                                </div>
                            </div>

                            <!-- Plan & Amount -->
                            <div class="grid grid-cols-2 gap-4">
                                <div>
                                    <label class="field-label">{{ t('groups.planName') }}</label>
                                    <input v-model="form.plan_name" type="text" class="glass-input w-full"
                                        placeholder="e.g. Premium" />
                                </div>
                                <div>
                                    <label class="field-label">{{ t('groups.amount') }}</label>
                                    <div class="relative">
                                        <input v-model.number="form.amount" type="number" step="0.01"
                                            class="glass-input w-full pl-8" />
                                        <span class="absolute left-3 top-1/2 -translate-y-1/2 text-slate-400">$</span>
                                    </div>
                                </div>
                            </div>

                            <!-- Billing Cycle -->
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

                        <p v-if="error" class="text-red-500 text-sm bg-red-50 p-2 rounded animate-pulse">{{ error }}</p>

                        <div class="flex justify-between pt-6 border-t border-slate-200 mt-8">
                            <button v-if="step === 1" type="button" @click="$emit('close')"
                                class="btn bg-slate-100 text-slate-600 hover:bg-slate-200">
                                {{ t('common.actions.cancel') }}
                            </button>
                            <button v-else type="button" @click="step = 1"
                                class="btn bg-slate-100 text-slate-600 hover:bg-slate-200">
                                {{ t('common.actions.back') }}
                            </button>

                            <button v-if="step === 1" type="button" @click="nextStep" class="btn btn-primary px-6">
                                {{ t('common.actions.next') }}
                            </button>

                            <div v-if="step === 2" class="flex gap-3">
                                <button type="button" @click="handleSubmit"
                                    class="btn text-slate-500 hover:text-slate-700 hover:bg-slate-50">
                                    {{ t('groups.skipAndCreate', 'Skip & Create') }}
                                </button>
                                <button type="submit" :disabled="isSubmitting" class="btn btn-primary min-w-[120px]">
                                    <span v-if="isSubmitting" class="flex items-center gap-2">
                                        <svg class="animate-spin h-4 w-4 text-white" viewBox="0 0 24 24">
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
                        </div>
                    </form>
                </div>
            </div>
        </Transition>
    </Teleport>
</template>

<style scoped>
.modal-overlay {
    position: fixed;
    top: 0;
    left: 0;
    width: 100%;
    height: 100%;
    background: rgba(15, 23, 42, 0.5);
    backdrop-filter: blur(8px);
    display: flex;
    justify-content: center;
    align-items: center;
    z-index: 1000;
    padding: 1rem;
}

.modal-content {
    width: 100%;
    max-width: 550px;
    background: white;
    border-radius: 1.5rem;
    box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.25);
    padding: 2rem;
    display: flex;
    flex-direction: column;
    max-height: 90vh;
}

.scrollable-form {
    overflow-y: auto;
    /* max-height: 60vh; */
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

.field-label {
    display: block;
    font-size: 0.875rem;
    font-weight: 600;
    color: var(--slate-700);
    margin-bottom: 0.5rem;
}

.form-group {
    margin-bottom: 1rem;
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

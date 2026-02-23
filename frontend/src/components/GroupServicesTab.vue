<script setup lang="ts">
import { ref, computed } from 'vue';
import { socket } from '../socket';
import { useI18n } from 'vue-i18n';
import { useUIStore } from '../stores/ui';

const { t } = useI18n();
const ui = useUIStore();

const props = defineProps<{
    groupId: string;
    services: any[];
    permissions: any;
}>();

const emit = defineEmits<{
    refresh: [];
}>();
const showCreateModal = ref(false);
const showEditModal = ref(false);
const selectedService = ref<any>(null);

const serviceForm = ref({
    service_name: '',
    website: '',
    plan_name: '',
    amount: 0,
    service_currency: 'TWD',
    payment_currency: 'TWD',
    billing_type: 'recurring' as 'once' | 'recurring',
    interval_unit: 'month' as 'minute' | 'hour' | 'day' | 'week' | 'month' | 'quarter' | 'year',
    interval_value: 1,
    billing_method: 'equal' as 'equal' | 'fixed' | 'percentage',
    status: 'active' as 'active' | 'paused' | 'cancelled'
});

const canCreateService = computed(() => props.permissions?.['group:create:services']);
const canUpdateService = computed(() => props.permissions?.['group:update:services']);
const canDeleteService = computed(() => props.permissions?.['group:delete:services']);

const activeServices = computed(() => props.services.filter(s => s.status === 'active'));
const pausedServices = computed(() => props.services.filter(s => s.status === 'paused'));
const cancelledServices = computed(() => props.services.filter(s => s.status === 'cancelled'));

const getStatusBadgeClass = (status: string) => {
    switch (status) {
        case 'active': return 'bg-green-100 text-green-700 border-green-200';
        case 'paused': return 'bg-yellow-100 text-yellow-700 border-yellow-200';
        case 'cancelled': return 'bg-red-100 text-red-700 border-red-200';
        default: return 'bg-slate-100 text-slate-700 border-slate-200';
    }
};

const getBillingTypeBadge = (type: string) => {
    return type === 'once' 
        ? 'bg-purple-100 text-purple-700 border-purple-200'
        : 'bg-blue-100 text-blue-700 border-blue-200';
};

const formatCurrency = (amount: number, currency: string) => {
    return new Intl.NumberFormat('zh-TW', {
        style: 'currency',
        currency: currency || 'TWD'
    }).format(amount);
};

const formatBillingCycle = (service: any) => {
    if (service.billing_type === 'once') return t('common.onetime');
    return `${service.interval_value} ${t(`common.${service.interval_unit}`)}`;
};

const openCreateModal = () => {
    serviceForm.value = {
        service_name: '',
        website: '',
        plan_name: '',
        amount: 0,
        service_currency: 'TWD',
        payment_currency: 'TWD',
        billing_type: 'recurring',
        interval_unit: 'month',
        interval_value: 1,
        billing_method: 'equal',
        status: 'active'
    };
    showCreateModal.value = true;
};

const openEditModal = (service: any) => {
    selectedService.value = service;
    serviceForm.value = { ...service };
    showEditModal.value = true;
};

const createService = () => {
    if (!serviceForm.value.service_name) {
        ui.alert(t('common.required'));
        return;
    }

    socket.emit('group:service:create', {
        groupId: props.groupId,
        ...serviceForm.value
    }, (res: any) => {
        if (res.status === 'ok') {
            ui.alert(t('common.success'));
            showCreateModal.value = false;
            emit('refresh');
        } else {
            ui.alert(res.message);
        }
    });
};

const updateService = () => {
    if (!selectedService.value) return;

    socket.emit('group:service:update', {
        groupId: props.groupId,
        serviceId: selectedService.value.id,
        ...serviceForm.value
    }, (res: any) => {
        if (res.status === 'ok') {
            ui.alert(t('common.success'));
            showEditModal.value = false;
            emit('refresh');
        } else {
            ui.alert(res.message);
        }
    });
};

const deleteService = async (service: any) => {
    if (!await ui.confirm(t('common.confirmDelete'))) return;

    socket.emit('group:service:delete', {
        groupId: props.groupId,
        serviceId: service.id
    }, (res: any) => {
        if (res.status === 'ok') {
            ui.alert(t('common.success'));
            emit('refresh');
        } else {
            ui.alert(res.message);
        }
    });
};

const updateServiceStatus = (service: any, status: string) => {
    socket.emit('group:service:update', {
        groupId: props.groupId,
        serviceId: service.id,
        status
    }, (res: any) => {
        if (res.status === 'ok') {
            emit('refresh');
        } else {
            ui.alert(res.message);
        }
    });
};
</script>

<template>
    <div class="services-tab">
        <!-- Header -->
        <div class="flex justify-between items-center mb-6">
            <div>
                <h3 class="text-lg font-bold text-slate-800">{{ t('groups.tabs.services') }}</h3>
                <p class="text-sm text-slate-600 mt-1">{{ t('groups.services.subtitle') }}</p>
            </div>
            <button 
                v-if="canCreateService"
                @click="openCreateModal"
                class="btn btn-primary"
            >
                {{ t('groups.services.addService') }}
            </button>
        </div>

        <!-- Summary Stats -->
        <div class="grid grid-cols-3 gap-4 mb-6">
            <div class="stat-card">
                <div class="text-sm text-slate-600">{{ t('groups.services.active') }}</div>
                <div class="text-2xl font-bold text-green-600">{{ activeServices.length }}</div>
            </div>
            <div class="stat-card">
                <div class="text-sm text-slate-600">{{ t('groups.services.paused') }}</div>
                <div class="text-2xl font-bold text-yellow-600">{{ pausedServices.length }}</div>
            </div>
            <div class="stat-card">
                <div class="text-sm text-slate-600">{{ t('groups.services.cancelled') }}</div>
                <div class="text-2xl font-bold text-red-600">{{ cancelledServices.length }}</div>
            </div>
        </div>

        <!-- Services List -->
        <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div 
                v-for="service in services" 
                :key="service.id"
                class="service-card"
            >
                <div class="flex items-start justify-between mb-3">
                    <div class="flex-1">
                        <h4 class="font-bold text-slate-800">{{ service.service_name }}</h4>
                        <p v-if="service.plan_name" class="text-sm text-slate-600">{{ service.plan_name }}</p>
                        <a v-if="service.website" :href="service.website" target="_blank" class="text-xs text-primary-600 hover:underline">
                            {{ service.website }}
                        </a>
                    </div>
                    <span :class="['badge ml-2', getStatusBadgeClass(service.status)]">
                        {{ t(`groups.services.status.${service.status}`) }}
                    </span>
                </div>

                <div class="space-y-2 mb-4">
                    <div class="flex justify-between text-sm">
                        <span class="text-slate-600">{{ t('groups.services.amount') }}</span>
                        <span class="font-semibold">{{ formatCurrency(service.amount, service.service_currency) }}</span>
                    </div>
                    <div class="flex justify-between text-sm">
                        <span class="text-slate-600">{{ t('groups.services.billingCycle') }}</span>
                        <span :class="['badge text-xs', getBillingTypeBadge(service.billing_type)]">
                            {{ formatBillingCycle(service) }}
                        </span>
                    </div>
                    <div class="flex justify-between text-sm">
                        <span class="text-slate-600">{{ t('groups.services.billingMethod') }}</span>
                        <span class="text-slate-800">{{ t(`groups.services.method.${service.billing_method}`) }}</span>
                    </div>
                </div>

                <div v-if="canUpdateService || canDeleteService" class="flex gap-2 pt-4 border-t border-slate-100">
                    <button v-if="canUpdateService" @click="openEditModal(service)" class="btn btn-sm btn-outline flex-1">
                        {{ t('common.edit') }}
                    </button>
                    <button v-if="service.status === 'active' && canUpdateService" @click="updateServiceStatus(service, 'paused')" class="btn btn-sm btn-outline">
                        {{ t('groups.services.pause') }}
                    </button>
                    <button v-if="service.status === 'paused' && canUpdateService" @click="updateServiceStatus(service, 'active')" class="btn btn-sm btn-outline">
                        {{ t('groups.services.resume') }}
                    </button>
                    <button v-if="canDeleteService" @click="deleteService(service)" class="btn btn-sm btn-danger-outline">
                        {{ t('common.delete') }}
                    </button>
                </div>
            </div>
        </div>

        <!-- Empty State -->
        <div v-if="services.length === 0" class="text-center py-12 text-slate-500">
            <p class="mb-4">{{ t('groups.services.empty') }}</p>
            <button v-if="canCreateService" @click="openCreateModal" class="btn btn-primary">
                {{ t('groups.services.addService') }}
            </button>
        </div>

        <!-- Create/Edit Modal -->
        <div v-if="showCreateModal || showEditModal" class="modal-overlay" @click.self="showCreateModal = false; showEditModal = false;">
            <div class="modal-content animate-scale-in" style="max-width: 600px;">
                <h3 class="text-xl font-bold text-slate-800 mb-4">
                    {{ showCreateModal ? t('groups.services.addService') : t('groups.services.editService') }}
                </h3>
                
                <div class="space-y-4 max-h-96 overflow-y-auto">
                    <div>
                        <label class="label">{{ t('groups.services.serviceName') }} *</label>
                        <input v-model="serviceForm.service_name" class="input-field" />
                    </div>

                    <div>
                        <label class="label">{{ t('groups.services.planName') }}</label>
                        <input v-model="serviceForm.plan_name" class="input-field" />
                    </div>

                    <div>
                        <label class="label">{{ t('groups.services.website') }}</label>
                        <input v-model="serviceForm.website" class="input-field" type="url" />
                    </div>

                    <div class="grid grid-cols-2 gap-4">
                        <div>
                            <label class="label">{{ t('groups.services.amount') }} *</label>
                            <input v-model.number="serviceForm.amount" class="input-field" type="number" min="0" step="0.01" />
                        </div>
                        <div>
                            <label class="label">{{ t('groups.services.currency') }}</label>
                            <select v-model="serviceForm.service_currency" class="input-field">
                                <option value="TWD">TWD</option>
                                <option value="USD">USD</option>
                                <option value="EUR">EUR</option>
                                <option value="JPY">JPY</option>
                            </select>
                        </div>
                    </div>

                    <div>
                        <label class="label">{{ t('groups.services.billingType') }}</label>
                        <select v-model="serviceForm.billing_type" class="input-field">
                            <option value="once">{{ t('common.onetime') }}</option>
                            <option value="recurring">{{ t('common.recurring') }}</option>
                        </select>
                    </div>

                    <div v-if="serviceForm.billing_type === 'recurring'" class="grid grid-cols-2 gap-4">
                        <div>
                            <label class="label">{{ t('groups.services.intervalValue') }}</label>
                            <input v-model.number="serviceForm.interval_value" class="input-field" type="number" min="1" />
                        </div>
                        <div>
                            <label class="label">{{ t('groups.services.intervalUnit') }}</label>
                            <select v-model="serviceForm.interval_unit" class="input-field">
                                <option value="day">{{ t('common.day') }}</option>
                                <option value="week">{{ t('common.week') }}</option>
                                <option value="month">{{ t('common.month') }}</option>
                                <option value="quarter">{{ t('common.quarter') }}</option>
                                <option value="year">{{ t('common.year') }}</option>
                            </select>
                        </div>
                    </div>

                    <div>
                        <label class="label">{{ t('groups.services.billingMethod') }}</label>
                        <select v-model="serviceForm.billing_method" class="input-field">
                            <option value="equal">{{ t('groups.services.method.equal') }}</option>
                            <option value="fixed">{{ t('groups.services.method.fixed') }}</option>
                            <option value="percentage">{{ t('groups.services.method.percentage') }}</option>
                        </select>
                    </div>

                    <div>
                        <label class="label">{{ t('groups.services.status') }}</label>
                        <select v-model="serviceForm.status" class="input-field">
                            <option value="active">{{ t('groups.services.status.active') }}</option>
                            <option value="paused">{{ t('groups.services.status.paused') }}</option>
                            <option value="cancelled">{{ t('groups.services.status.cancelled') }}</option>
                        </select>
                    </div>
                </div>

                <div class="flex gap-3 mt-6">
                    <button @click="showCreateModal = false; showEditModal = false;" class="btn btn-outline flex-1">
                        {{ t('common.cancel') }}
                    </button>
                    <button @click="showCreateModal ? createService() : updateService()" class="btn btn-primary flex-1">
                        {{ showCreateModal ? t('common.create') : t('common.save') }}
                    </button>
                </div>
            </div>
        </div>
    </div>
</template>

<style scoped>
.services-tab {
    animation: fadeIn 0.3s ease;
}

.service-card {
    padding: 1.25rem;
    border-radius: 0.75rem;
    background-color: white;
    border: 1px solid #e2e8f0;
    transition: all 0.3s ease;
}

.service-card:hover {
    border-color: hsl(250, 95%, 88%);
    box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.1);
}

.stat-card {
    padding: 1rem;
    border-radius: 0.75rem;
    background-color: white;
    border: 1px solid #e2e8f0;
    text-align: center;
}

.badge {
    padding: 0.25rem 0.75rem;
    border-radius: 9999px;
    font-size: 0.75rem;
    font-weight: 600;
    border: 1px solid currentColor;
    white-space: nowrap;
}

.modal-overlay {
    position: fixed;
    inset: 0;
    background-color: rgba(0, 0, 0, 0.5);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 50;
    padding: 1rem;
}

.modal-content {
    background-color: white;
    border-radius: 1rem;
    padding: 1.5rem;
    max-width: 28rem;
    width: 100%;
    box-shadow: 0 20px 25px -5px rgba(0, 0, 0, 0.1);
}

@keyframes scale-in {
    from {
        opacity: 0;
        transform: scale(0.95);
    }
    to {
        opacity: 1;
        transform: scale(1);
    }
}

.animate-scale-in {
    animation: scale-in 0.2s ease-out;
}
</style>

<script setup lang="ts">
import { ref, computed } from 'vue';
import { socket } from '../socket';
import { useI18n } from 'vue-i18n';
import { useUIStore } from '../stores/ui';

const { t } = useI18n();
const ui = useUIStore();

const props = defineProps<{
    groupId: string;
    subscriptions: any[];
    permissions: any;
}>();

const emit = defineEmits<{
    refresh: [];
}>();

const showCreateModal = ref(false);
const showEditModal = ref(false);
const selectedSubscription = ref<any>(null);

const subscriptionForm = ref({
    service_name: '',
    amount: 0,
    cycle: 'monthly' as 'monthly' | 'yearly',
    status: 'active' as 'active' | 'paused' | 'cancelled'
});

const canCreateSubscription = computed(() => props.permissions?.['group:create:subscriptions']);
const canUpdateSubscription = computed(() => props.permissions?.['group:update:subscriptions']);
const canDeleteSubscription = computed(() => props.permissions?.['group:delete:subscriptions']);

const activeSubscriptions = computed(() => props.subscriptions.filter(s => s.status === 'active'));
const pausedSubscriptions = computed(() => props.subscriptions.filter(s => s.status === 'paused'));

const totalMonthly = computed(() => {
    return props.subscriptions
        .filter(s => s.status === 'active' && s.cycle === 'monthly')
        .reduce((sum, s) => sum + (s.amount || 0), 0);
});

const totalYearly = computed(() => {
    return props.subscriptions
        .filter(s => s.status === 'active' && s.cycle === 'yearly')
        .reduce((sum, s) => sum + (s.amount || 0), 0);
});

const getStatusBadgeClass = (status: string) => {
    switch (status) {
        case 'active': return 'bg-green-100 text-green-700 border-green-200';
        case 'paused': return 'bg-yellow-100 text-yellow-700 border-yellow-200';
        case 'cancelled': return 'bg-red-100 text-red-700 border-red-200';
        default: return 'bg-slate-100 text-slate-700 border-slate-200';
    }
};

const getCycleBadgeClass = (cycle: string) => {
    return cycle === 'monthly'
        ? 'bg-blue-100 text-blue-700 border-blue-200'
        : 'bg-purple-100 text-purple-700 border-purple-200';
};

const formatCurrency = (amount: number) => {
    return new Intl.NumberFormat('zh-TW', {
        style: 'currency',
        currency: 'TWD'
    }).format(amount);
};

const formatDate = (date?: string) => {
    if (!date) return '-';
    return new Date(date).toLocaleDateString('zh-TW');
};

const formatCycleText = (cycle: string) => {
    return cycle === 'monthly' ? t('common.monthly') : t('common.yearly');
};

const openCreateModal = () => {
    subscriptionForm.value = {
        service_name: '',
        amount: 0,
        cycle: 'monthly',
        status: 'active'
    };
    showCreateModal.value = true;
};

const openEditModal = (subscription: any) => {
    selectedSubscription.value = subscription;
    subscriptionForm.value = { ...subscription };
    showEditModal.value = true;
};

const createSubscription = () => {
    if (!subscriptionForm.value.service_name || subscriptionForm.value.amount <= 0) {
        ui.alert(t('common.required'));
        return;
    }

    socket.emit('subscription:create', {
        groupId: props.groupId,
        ...subscriptionForm.value
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

const updateSubscription = () => {
    if (!selectedSubscription.value) return;

    socket.emit('subscription:update', {
        groupId: props.groupId,
        subscriptionId: selectedSubscription.value.id,
        ...subscriptionForm.value
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

const deleteSubscription = async (subscription: any) => {
    if (!await ui.confirm(t('common.confirmDelete'))) return;

    socket.emit('subscription:delete', {
        groupId: props.groupId,
        subscriptionId: subscription.id
    }, (res: any) => {
        if (res.status === 'ok') {
            ui.alert(t('common.success'));
            emit('refresh');
        } else {
            ui.alert(res.message);
        }
    });
};

const updateSubscriptionStatus = (subscription: any, status: string) => {
    socket.emit('subscription:update', {
        groupId: props.groupId,
        subscriptionId: subscription.id,
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
    <div class="subscriptions-tab">
        <!-- Header -->
        <div class="flex justify-between items-center mb-6">
            <div>
                <h3 class="text-lg font-bold text-slate-800">{{ t('groups.tabs.subscriptions') }}</h3>
                <p class="text-sm text-slate-600 mt-1">{{ t('groups.subscriptions.subtitle') }}</p>
            </div>
            <button 
                v-if="canCreateSubscription"
                @click="openCreateModal"
                class="btn btn-primary"
            >
                {{ t('groups.subscriptions.addSubscription') }}
            </button>
        </div>

        <!-- Summary Stats -->
        <div class="grid grid-cols-4 gap-4 mb-6">
            <div class="stat-card">
                <div class="text-sm text-slate-600">{{ t('groups.subscriptions.active') }}</div>
                <div class="text-2xl font-bold text-green-600">{{ activeSubscriptions.length }}</div>
            </div>
            <div class="stat-card">
                <div class="text-sm text-slate-600">{{ t('common.monthly') }}</div>
                <div class="text-2xl font-bold text-blue-600">{{ formatCurrency(totalMonthly) }}</div>
            </div>
            <div class="stat-card">
                <div class="text-sm text-slate-600">{{ t('common.yearly') }}</div>
                <div class="text-2xl font-bold text-purple-600">{{ formatCurrency(totalYearly) }}</div>
            </div>
            <div class="stat-card">
                <div class="text-sm text-slate-600">{{ t('groups.subscriptions.paused') }}</div>
                <div class="text-2xl font-bold text-yellow-600">{{ pausedSubscriptions.length }}</div>
            </div>
        </div>

        <!-- Subscriptions List -->
        <div class="space-y-3">
            <div 
                v-for="subscription in subscriptions" 
                :key="subscription.id"
                class="subscription-card"
            >
                <div class="flex items-start justify-between mb-3">
                    <div class="flex-1">
                        <h4 class="font-bold text-slate-800">{{ subscription.service_name }}</h4>
                        <p class="text-sm text-slate-600 mt-1">
                            {{ t('groups.subscriptions.nextPayment') }}: {{ formatDate(subscription.next_payment_date) }}
                        </p>
                    </div>
                    <div class="flex gap-2">
                        <span :class="['badge', getStatusBadgeClass(subscription.status)]">
                            {{ t(`groups.subscriptions.status.${subscription.status}`) }}
                        </span>
                        <span :class="['badge', getCycleBadgeClass(subscription.cycle)]">
                            {{ formatCycleText(subscription.cycle) }}
                        </span>
                    </div>
                </div>

                <div class="flex justify-between items-center pt-4 border-t border-slate-100">
                    <div class="text-2xl font-bold text-slate-800">
                        {{ formatCurrency(subscription.amount) }}
                    </div>
                    <div v-if="canUpdateSubscription || canDeleteSubscription" class="flex gap-2">
                        <button v-if="canUpdateSubscription" @click="openEditModal(subscription)" class="btn btn-sm btn-outline">
                            {{ t('common.edit') }}
                        </button>
                        <button 
                            v-if="subscription.status === 'active' && canUpdateSubscription" 
                            @click="updateSubscriptionStatus(subscription, 'paused')" 
                            class="btn btn-sm btn-outline"
                        >
                            {{ t('groups.subscriptions.pause') }}
                        </button>
                        <button 
                            v-if="subscription.status === 'paused' && canUpdateSubscription" 
                            @click="updateSubscriptionStatus(subscription, 'active')" 
                            class="btn btn-sm btn-outline"
                        >
                            {{ t('groups.subscriptions.resume') }}
                        </button>
                        <button v-if="canDeleteSubscription" @click="deleteSubscription(subscription)" class="btn btn-sm btn-danger-outline">
                            {{ t('common.delete') }}
                        </button>
                    </div>
                </div>
            </div>
        </div>

        <!-- Empty State -->
        <div v-if="subscriptions.length === 0" class="text-center py-12 text-slate-500">
            <p class="mb-4">{{ t('groups.subscriptions.empty') }}</p>
            <button v-if="canCreateSubscription" @click="openCreateModal" class="btn btn-primary">
                {{ t('groups.subscriptions.addSubscription') }}
            </button>
        </div>

        <!-- Create/Edit Modal -->
        <div v-if="showCreateModal || showEditModal" class="modal-overlay" @click.self="showCreateModal = false; showEditModal = false;">
            <div class="modal-content animate-scale-in">
                <h3 class="text-xl font-bold text-slate-800 mb-4">
                    {{ showCreateModal ? t('groups.subscriptions.addSubscription') : t('groups.subscriptions.editSubscription') }}
                </h3>
                
                <div class="space-y-4">
                    <div>
                        <label class="label">{{ t('groups.subscriptions.serviceName') }} *</label>
                        <input v-model="subscriptionForm.service_name" class="input-field" />
                    </div>

                    <div>
                        <label class="label">{{ t('groups.subscriptions.amount') }} *</label>
                        <input v-model.number="subscriptionForm.amount" class="input-field" type="number" min="0" step="0.01" />
                    </div>

                    <div>
                        <label class="label">{{ t('groups.subscriptions.billingCycle') }}</label>
                        <select v-model="subscriptionForm.cycle" class="input-field">
                            <option value="monthly">{{ t('common.monthly') }}</option>
                            <option value="yearly">{{ t('common.yearly') }}</option>
                        </select>
                    </div>

                    <div>
                        <label class="label">{{ t('groups.subscriptions.status') }}</label>
                        <select v-model="subscriptionForm.status" class="input-field">
                            <option value="active">{{ t('groups.subscriptions.status.active') }}</option>
                            <option value="paused">{{ t('groups.subscriptions.status.paused') }}</option>
                            <option value="cancelled">{{ t('groups.subscriptions.status.cancelled') }}</option>
                        </select>
                    </div>
                </div>

                <div class="flex gap-3 mt-6">
                    <button @click="showCreateModal = false; showEditModal = false;" class="btn btn-outline flex-1">
                        {{ t('common.cancel') }}
                    </button>
                    <button @click="showCreateModal ? createSubscription() : updateSubscription()" class="btn btn-primary flex-1">
                        {{ showCreateModal ? t('common.create') : t('common.save') }}
                    </button>
                </div>
            </div>
        </div>
    </div>
</template>

<style scoped>
.subscriptions-tab {
    animation: fadeIn 0.3s ease;
}

.subscription-card {
    padding: 1.25rem;
    border-radius: 0.75rem;
    background-color: white;
    border: 1px solid #e2e8f0;
    transition: all 0.3s ease;
}

.subscription-card:hover {
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

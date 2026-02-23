<script setup lang="ts">
import { ref, computed } from 'vue';
import { socket } from '../socket';
import { useI18n } from 'vue-i18n';
import { useUIStore } from '../stores/ui';

const { t } = useI18n();
const ui = useUIStore();

const props = defineProps<{
    groupId: string;
    expenses: any[];
    members: any[];
    permissions: any;
}>();

const emit = defineEmits<{
    refresh: [];
}>();

const showCreateModal = ref(false);
const showSplitsModal = ref(false);
const selectedExpense = ref<any>(null);

const expenseForm = ref({
    description: '',
    amount: 0,
    paid_by: '',
    split_type: 'equal' as 'equal' | 'custom',
    splits: [] as any[]
});

const canCreateExpense = computed(() => props.permissions?.['group:create:expenses']);
const canDeleteExpense = computed(() => props.permissions?.['group:delete:expenses']);

const totalExpenses = computed(() => {
    return props.expenses.reduce((sum, e) => sum + (e.amount || 0), 0);
});

const formatCurrency = (amount: number) => {
    return new Intl.NumberFormat('zh-TW', {
        style: 'currency',
        currency: 'TWD'
    }).format(amount);
};

const formatDate = (date: string) => {
    return new Date(date).toLocaleDateString('zh-TW');
};

const getPaidByName = (expense: any) => {
    const member = props.members.find(m => m.id === expense.paid_by);
    return member?.display_name || member?.temp_name || t('common.unknown');
};

const openCreateModal = () => {
    expenseForm.value = {
        description: '',
        amount: 0,
        paid_by: '',
        split_type: 'equal',
        splits: []
    };
    showCreateModal.value = true;
};

const initializeSplits = () => {
    if (expenseForm.value.split_type === 'equal') {
        const amount = expenseForm.value.amount;
        const count = props.members.filter(m => m.user_id).length;
        const perPerson = count > 0 ? amount / count : 0;
        
        expenseForm.value.splits = props.members
            .filter(m => m.user_id)
            .map(m => ({
                member_id: m.id,
                amount_owed: perPerson
            }));
    } else {
        expenseForm.value.splits = props.members
            .filter(m => m.user_id)
            .map(m => ({
                member_id: m.id,
                amount_owed: 0
            }));
    }
};

const createExpense = () => {
    if (!expenseForm.value.description || !expenseForm.value.paid_by || expenseForm.value.amount <= 0) {
        ui.alert(t('common.required'));
        return;
    }

    initializeSplits();

    const totalSplit = expenseForm.value.splits.reduce((sum, s) => sum + s.amount_owed, 0);
    if (Math.abs(totalSplit - expenseForm.value.amount) > 0.01) {
        ui.alert(t('groups.expenses.splitMismatch'));
        return;
    }

    socket.emit('expense:create', {
        groupId: props.groupId,
        ...expenseForm.value
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

const deleteExpense = async (expense: any) => {
    if (!await ui.confirm(t('common.confirmDelete'))) return;

    socket.emit('expense:delete', {
        groupId: props.groupId,
        expenseId: expense.id
    }, (res: any) => {
        if (res.status === 'ok') {
            ui.alert(t('common.success'));
            emit('refresh');
        } else {
            ui.alert(res.message);
        }
    });
};

const viewSplits = (expense: any) => {
    selectedExpense.value = expense;
    showSplitsModal.value = true;
};

const getMemberName = (memberId: string) => {
    const member = props.members.find(m => m.id === memberId);
    return member?.display_name || member?.temp_name || t('common.unknown');
};
</script>

<template>
    <div class="expenses-tab">
        <!-- Header -->
        <div class="flex justify-between items-center mb-6">
            <div>
                <h3 class="text-lg font-bold text-slate-800">{{ t('groups.tabs.expenses') }}</h3>
                <p class="text-sm text-slate-600 mt-1">{{ t('groups.expenses.subtitle') }}</p>
            </div>
            <button 
                v-if="canCreateExpense"
                @click="openCreateModal"
                class="btn btn-primary"
            >
                {{ t('groups.expenses.addExpense') }}
            </button>
        </div>

        <!-- Summary -->
        <div class="stat-card mb-6">
            <div class="text-sm text-slate-600">{{ t('groups.expenses.totalExpenses') }}</div>
            <div class="text-3xl font-bold text-primary-600">{{ formatCurrency(totalExpenses) }}</div>
            <div class="text-xs text-slate-500 mt-1">{{ expenses.length }} {{ t('groups.expenses.transactions') }}</div>
        </div>

        <!-- Expenses List -->
        <div class="space-y-3">
            <div 
                v-for="expense in expenses" 
                :key="expense.id"
                class="expense-card"
            >
                <div class="flex items-start justify-between">
                    <div class="flex-1">
                        <h4 class="font-bold text-slate-800">{{ expense.description }}</h4>
                        <p class="text-sm text-slate-600 mt-1">
                            {{ t('groups.expenses.paidBy') }}: {{ getPaidByName(expense) }}
                        </p>
                        <p class="text-xs text-slate-500 mt-1">{{ formatDate(expense.date || expense.created_at) }}</p>
                    </div>
                    <div class="text-right">
                        <div class="text-xl font-bold text-slate-800">{{ formatCurrency(expense.amount) }}</div>
                        <button @click="viewSplits(expense)" class="text-xs text-primary-600 hover:underline mt-1">
                            {{ t('groups.expenses.viewSplits') }}
                        </button>
                    </div>
                </div>

                <div v-if="canDeleteExpense" class="flex gap-2 mt-4 pt-4 border-t border-slate-100">
                    <button @click="deleteExpense(expense)" class="btn btn-sm btn-danger-outline">
                        {{ t('common.delete') }}
                    </button>
                </div>
            </div>
        </div>

        <!-- Empty State -->
        <div v-if="expenses.length === 0" class="text-center py-12 text-slate-500">
            <p class="mb-4">{{ t('groups.expenses.empty') }}</p>
            <button v-if="canCreateExpense" @click="openCreateModal" class="btn btn-primary">
                {{ t('groups.expenses.addExpense') }}
            </button>
        </div>

        <!-- Create Modal -->
        <div v-if="showCreateModal" class="modal-overlay" @click.self="showCreateModal = false">
            <div class="modal-content animate-scale-in" style="max-width: 500px;">
                <h3 class="text-xl font-bold text-slate-800 mb-4">{{ t('groups.expenses.addExpense') }}</h3>
                
                <div class="space-y-4">
                    <div>
                        <label class="label">{{ t('groups.expenses.description') }} *</label>
                        <input v-model="expenseForm.description" class="input-field" />
                    </div>

                    <div>
                        <label class="label">{{ t('groups.expenses.amount') }} *</label>
                        <input v-model.number="expenseForm.amount" class="input-field" type="number" min="0" step="0.01" />
                    </div>

                    <div>
                        <label class="label">{{ t('groups.expenses.paidBy') }} *</label>
                        <select v-model="expenseForm.paid_by" class="input-field">
                            <option value="">{{ t('common.select') }}</option>
                            <option v-for="member in members.filter(m => m.user_id)" :key="member.id" :value="member.id">
                                {{ member.display_name || member.temp_name }}
                            </option>
                        </select>
                    </div>

                    <div>
                        <label class="label">{{ t('groups.expenses.splitType') }}</label>
                        <select v-model="expenseForm.split_type" class="input-field">
                            <option value="equal">{{ t('groups.expenses.splitEqual') }}</option>
                            <option value="custom">{{ t('groups.expenses.splitCustom') }}</option>
                        </select>
                    </div>
                </div>

                <div class="flex gap-3 mt-6">
                    <button @click="showCreateModal = false" class="btn btn-outline flex-1">
                        {{ t('common.cancel') }}
                    </button>
                    <button @click="createExpense" class="btn btn-primary flex-1">
                        {{ t('common.create') }}
                    </button>
                </div>
            </div>
        </div>

        <!-- Splits Modal -->
        <div v-if="showSplitsModal && selectedExpense" class="modal-overlay" @click.self="showSplitsModal = false">
            <div class="modal-content animate-scale-in" style="max-width: 500px;">
                <h3 class="text-xl font-bold text-slate-800 mb-4">{{ t('groups.expenses.splitDetails') }}</h3>
                
                <div class="mb-4">
                    <p class="text-sm text-slate-600">{{ selectedExpense.description }}</p>
                    <p class="text-2xl font-bold text-slate-800 mt-2">{{ formatCurrency(selectedExpense.amount) }}</p>
                </div>

                <div class="space-y-2">
                    <div 
                        v-for="split in selectedExpense.splits" 
                        :key="split.member_id"
                        class="flex justify-between items-center p-3 bg-slate-50 rounded-lg"
                    >
                        <span class="text-sm font-medium text-slate-800">{{ getMemberName(split.member_id) }}</span>
                        <div class="text-right">
                            <div class="font-semibold text-slate-800">{{ formatCurrency(split.amount_owed) }}</div>
                            <span :class="['text-xs', split.status === 'paid' ? 'text-green-600' : 'text-yellow-600']">
                                {{ t(`groups.expenses.${split.status}`) }}
                            </span>
                        </div>
                    </div>
                </div>

                <div class="flex gap-3 mt-6">
                    <button @click="showSplitsModal = false" class="btn btn-primary flex-1">
                        {{ t('common.close') }}
                    </button>
                </div>
            </div>
        </div>
    </div>
</template>

<style scoped>
.expenses-tab {
    animation: fadeIn 0.3s ease;
}

.expense-card {
    padding: 1.25rem;
    border-radius: 0.75rem;
    background-color: white;
    border: 1px solid #e2e8f0;
    transition: all 0.3s ease;
}

.expense-card:hover {
    border-color: hsl(250, 95%, 88%);
    box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.1);
}

.stat-card {
    padding: 1.5rem;
    border-radius: 0.75rem;
    background: linear-gradient(to bottom right, hsl(250, 100%, 97%), white);
    border: 1px solid hsl(250, 100%, 94%);
    text-align: center;
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

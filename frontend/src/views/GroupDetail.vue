<script setup lang="ts">
import { ref, onMounted } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { socket } from '../socket';
import { useAuthStore } from '../stores/auth';
import AddExpenseModal from '../components/AddExpenseModal.vue';
import BillTicket from '../components/BillTicket.vue';
import BillDetailModal from '../components/BillDetailModal.vue';
import FileManager from '../components/FileManager.vue';
import MainLayout from './MainLayout.vue';
import { useI18n } from 'vue-i18n';

const { t } = useI18n();
const authStore = useAuthStore();

const route = useRoute();
const router = useRouter();
const groupId = route.params.id as string;

const group = ref<any>(null);
const members = ref<any[]>([]);
const expenses = ref<any[]>([]);
const splits = ref<any[]>([]);
const loading = ref(true);
const error = ref('');

// Member management
const newMemberValue = ref('');
const newMemberType = ref<'email' | 'name'>('name');
const showAddMember = ref(false);
const showAddExpense = ref(false);
const selectedSplit = ref<any>(null);
const showBill = ref(false);

const viewBill = (split: any) => {
    selectedSplit.value = split;
    showBill.value = true;
};

const bills = ref<any[]>([]);
const currentTab = ref('expenses'); // 'expenses' | 'bills' | 'settings'
const showBillDetail = ref(false);
const selectedBill = ref<any>(null);
const selectedBillSplits = ref<any[]>([]);

const viewBillDetail = (bill: any) => {
    selectedBill.value = bill;
    // Fetch splits for this bill
    socket.emit('bill:get', { billId: bill.id }, (res: any) => {
        if (res.status === 'ok') {
            selectedBill.value = res.bill;
            selectedBillSplits.value = res.splits;
            showBillDetail.value = true;
        } else {
            alert(res.message);
        }
    });
};

const fetchData = () => {
    loading.value = true;
    socket.emit('group:get', { groupId }, (res: any) => {
        if (res.status === 'ok') {
            group.value = res.group;
            members.value = res.members;
            loading.value = false;
        } else {
            error.value = res.message;
            loading.value = false;
        }
    });

    socket.emit('expense:list', { groupId }, (res: any) => {
        if (res.status === 'ok') {
            expenses.value = res.expenses;
        }
    });

    socket.emit('expense:get_splits', { groupId }, (res: any) => {
        if (res.status === 'ok') {
            splits.value = res.splits;
        }
    });

    socket.emit('bill:list', { groupId }, (res: any) => {
        if (res.status === 'ok') {
            bills.value = res.bills;
        }
    });
};

onMounted(() => {
    if (socket.connected) {
        fetchData();
    } else {
        socket.once('connect', fetchData);
    }
});

const handleExpenseAdded = () => {
    showAddExpense.value = false;
    fetchData();
};

const settleExpense = (expenseId: string, userId: string) => {
    if (!confirm(t('groups.confirmSettle'))) return;
    socket.emit('expense:settle', { expenseId, userId }, (res: any) => {
        if (res.status === 'ok') {
            fetchData();
        } else {
            alert(res.message);
        }
    });
};

const addMember = () => {
    if (!newMemberValue.value) return;
    const payload = {
        groupId,
        [newMemberType.value]: newMemberValue.value
    };
    socket.emit('group:add_member', payload, (res: any) => {
        if (res.status === 'ok') {
            newMemberValue.value = '';
            showAddMember.value = false;
            fetchData();
        } else {
            alert(res.message);
        }
    });
};

const bindAccount = (memberId: string) => {
    if (!confirm(t('groups.bindAccount') + '?')) return;
    socket.emit('group:bind_member', { memberId }, (res: any) => {
        if (res.status === 'ok') {
            fetchData();
        } else {
            alert(res.message);
        }
    });
};
</script>

<template>
    <MainLayout>
        <div class="group-detail" v-if="!loading && group">
            <header class="header">
                <div class="title-section">
                    <button @click="router.push('/groups')" class="back-btn">
                        <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4 inline-block mr-1" fill="none"
                            viewBox="0 0 24 24" stroke="currentColor">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                                d="M10 19l-7-7m0 0l7-7m-7 7h18" />
                        </svg>
                        {{ t('groups.backToGroups') }}
                    </button>
                    <h1>{{ group.name }}</h1>
                </div>
                <div class="actions">
                    <button @click="showAddMember = !showAddMember" class="btn btn-secondary add-member-btn">
                        {{ t('groups.addMember') }}
                    </button>
                    <button @click="showAddExpense = true" class="btn btn-primary">
                        {{ t('groups.addExpense') }}
                    </button>
                </div>
            </header>

            <!-- Group Info Header -->
            <div class="group-info-card card mb-8">
                <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
                    <div class="info-item">
                        <span class="label">{{ t('groups.serviceName') }}</span>
                        <div class="value-row">
                            <span class="value">{{ group.service_name || '---' }}</span>
                            <a v-if="group.website" :href="group.website" target="_blank" class="link-icon">
                                <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24"
                                    stroke="currentColor">
                                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                                        d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" />
                                </svg>
                            </a>
                        </div>
                    </div>
                    <div class="info-item">
                        <span class="label">{{ t('groups.planName') }}</span>
                        <span class="value">{{ group.plan_name || '---' }}</span>
                    </div>
                    <div class="info-item">
                        <span class="label">{{ t('groups.amount') }}</span>
                        <span class="value highlight">{{ group.currency }} {{ group.amount }} / {{
                            t(`common.cycles.${group.billing_cycle}`) }}</span>
                    </div>
                    <div class="info-item">
                        <span class="label">{{ t('groups.billingMethod') }}</span>
                        <span class="value">{{ t(`groups.methods.${group.billing_method}`) }}</span>
                    </div>
                </div>
            </div>

            <AddExpenseModal v-if="showAddExpense" :group-id="groupId" :members="members"
                @close="showAddExpense = false" @added="handleExpenseAdded" />

            <BillTicket v-if="showBill && selectedSplit" :group="group"
                :member="members.find(m => m.user_id === selectedSplit.user_id)" :split="selectedSplit" :show="showBill"
                @close="showBill = false" />

            <div v-if="showAddMember" class="add-member-form card">
                <h3>{{ t('groups.inviteNewMember') }}</h3>
                <div class="form-row">
                    <select v-model="newMemberType" class="type-select">
                        <option value="name">{{ t('common.fields.name') }}</option>
                        <option value="email">{{ t('common.fields.email') }}</option>
                    </select>
                    <input v-model="newMemberValue" type="text" :placeholder="t('common.placeholders.member')"
                        class="input" />
                    <button @click="addMember" class="btn btn-primary">{{ t('common.actions.add') }}</button>
                    <button @click="showAddMember = false" class="btn btn-text">{{ t('common.actions.cancel')
                        }}</button>
                </div>
            </div>

            <div class="grid grid-cols-1 lg:grid-cols-3 gap-8">
                <!-- Main Content: Expenses & Debts -->
                <div class="lg:col-span-2 space-y-8">
                    <!-- Tabs -->
                    <div class="tabs">
                        <button @click="currentTab = 'expenses'"
                            :class="['tab-btn', { active: currentTab === 'expenses' }]">
                            {{ t('groups.expenses') }}
                        </button>
                        <button @click="currentTab = 'bills'" :class="['tab-btn', { active: currentTab === 'bills' }]">
                            {{ t('groups.bills') }}
                        </button>
                    </div>

                    <div v-if="currentTab === 'expenses'">
                        <section class="expenses-section">
                            <!-- Existing Expenses Section -->
                            <div class="section-header">
                                <h2>{{ t('groups.expenses') }}</h2>
                            </div>

                            <div v-if="expenses.length === 0" class="empty-state">
                                {{ t('groups.noExpenses') }}
                            </div>
                            <div v-else class="expense-list">
                                <div v-for="expense in expenses" :key="expense.id" class="expense-card card">
                                    <div class="expense-main">
                                        <span class="desc">{{ expense.description }}</span>
                                        <span class="date">{{ new Date(expense.date).toLocaleDateString() }}</span>
                                    </div>
                                    <div class="expense-amount">
                                        ${{ expense.amount }}
                                    </div>
                                </div>
                            </div>
                        </section>

                        <section class="debts-section mt-8">
                            <div class="section-header">
                                <h2>{{ t('groups.unpaidDebts') }}</h2>
                            </div>

                            <div v-if="splits.length === 0" class="empty-state">
                                {{ t('groups.settled') }}
                            </div>
                            <div v-else class="split-list">
                                <div v-for="split in splits" :key="split.expense_id + split.user_id"
                                    class="split-card card">
                                    <div class="split-info">
                                        <div class="split-text">
                                            <span class="user-highlight">
                                                {{members.find(m => m.user_id === split.user_id)?.username ||
                                                    split.user_id
                                                }}
                                            </span>
                                            {{ t('groups.owe') }}
                                            <span class="user-highlight">{{ split.payer_name }}</span>
                                        </div>
                                        <div class="amount-row">
                                            <span class="amount text-danger">${{ split.amount_owed }}</span>
                                            <span class="reason">{{ t('groups.for') }}: {{ split.description }}</span>
                                        </div>
                                    </div>
                                    <div class="split-actions">
                                        <button @click="viewBill(split)" class="btn btn-sm btn-ghost"
                                            :title="t('groups.viewBill')">
                                            <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none"
                                                viewBox="0 0 24 24" stroke="currentColor">
                                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                                                    d="M7 21h10a2 2 0 002-2V9.414a1 1 0 00-.293-.707l-5.414-5.414A1 1 0 0012.586 3H7a2 2 0 00-2 2v14a2 2 0 002 2z" />
                                            </svg>
                                        </button>
                                        <button
                                            v-if="split.user_id === authStore.user?.id || group.created_by === authStore.user?.id"
                                            @click="settleExpense(split.expense_id, split.user_id)"
                                            class="btn btn-sm btn-success">
                                            {{ t('groups.settle') }}
                                        </button>
                                    </div>
                                </div>
                            </div>
                        </section>
                    </div>

                    <div v-else-if="currentTab === 'bills'">
                        <section class="bills-section">
                            <div class="section-header">
                                <h2>{{ t('groups.generatedBills') }}</h2>
                            </div>
                            <div v-if="bills.length === 0" class="empty-state">
                                {{ t('groups.noBills') }}
                            </div>
                            <div v-else class="bill-list">
                                <div v-for="bill in bills" :key="bill.id" class="bill-card card"
                                    @click="viewBillDetail(bill)">
                                    <div class="bill-info">
                                        <span class="bill-title">{{ bill.title }}</span>
                                        <span class="bill-date">{{ t('groups.dueDate') }}: {{ new
                                            Date(bill.due_date).toLocaleDateString() }}</span>
                                    </div>
                                    <div class="bill-amount">
                                        {{ bill.currency }} {{ bill.total_amount }}
                                        <span class="status-badge" :class="bill.status">{{
                                            t(`groups.status_${bill.status}`) }}</span>
                                    </div>
                                </div>
                            </div>
                        </section>
                    </div>
                </div>

                <BillDetailModal v-if="showBillDetail && selectedBill" :bill="selectedBill" :splits="selectedBillSplits"
                    :is-host="group.created_by === authStore.user?.id" @close="showBillDetail = false"
                    @updated="viewBillDetail(selectedBill)" />

                <!-- Sidebar: Members & Files -->
                <aside class="members-sidebar space-y-8">
                    <FileManager :group-id="groupId" />
                    <div class="card">
                        <h2>{{ t('groups.members') }}</h2>
                        <ul class="member-list">
                            <li v-for="member in members" :key="member.member_id" class="member-item">
                                <div class="member-avatar">{{ (member.username || '?')[0].toUpperCase() }}</div>
                                <div class="member-details">
                                    <span class="name">{{ member.username }}</span>
                                    <span class="role">
                                        {{ member.role }}
                                        <span v-if="member.temp_name" class="badge">({{ t('groups.nonMember') }})</span>
                                    </span>
                                </div>
                                <div v-if="member.temp_name" class="member-actions">
                                    <button @click="bindAccount(member.member_id)" class="btn btn-xs btn-outline"
                                        :title="t('groups.bindAccount')">
                                        <svg xmlns="http://www.w3.org/2000/svg" class="h-3 w-3" fill="none"
                                            viewBox="0 0 24 24" stroke="currentColor">
                                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                                                d="M13.828 10.172a4 4 0 00-5.656 0l-4 4a4 4 0 105.656 5.656l1.102-1.101" />
                                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                                                d="M10.172 13.828a4 4 0 005.656 0l4-4a4 4 0 00-5.656-5.656l-1.102 1.101" />
                                        </svg>
                                    </button>
                                </div>
                            </li>
                        </ul>
                    </div>
                </aside>
            </div>
        </div>
        <div v-else-if="loading" class="state-container">{{ t('common.status.loading') }}</div>
        <div v-else class="state-container error">{{ error }}</div>
    </MainLayout>
</template>

<style scoped>
.header {
    display: flex;
    flex-direction: column;
    gap: 1rem;
    margin-bottom: 2rem;
}

@media (min-width: 640px) {
    .header {
        flex-direction: row;
        justify-content: space-between;
        align-items: center;
    }
}

.title-section {
    display: flex;
    align-items: center;
    gap: 1rem;
}

.title-section h1 {
    font-size: 1.5rem;
    font-weight: 700;
    margin: 0;
}

.back-btn {
    background: none;
    border: 1px solid var(--border-color);
    padding: 0.25rem 0.75rem;
    border-radius: var(--radius-sm);
    color: var(--text-muted);
    font-size: 0.875rem;
    cursor: pointer;
}

.actions {
    display: flex;
    gap: 0.5rem;
}

/* Group Info Card */
.group-info-card {
    background-color: var(--bg-surface);
    border-left: 4px solid var(--primary-color);
}

.info-item {
    display: flex;
    flex-direction: column;
    gap: 0.25rem;
}

.info-item .label {
    font-size: 0.75rem;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: var(--text-muted);
}

.info-item .value {
    font-weight: 600;
    color: var(--text-main);
}

.info-item .value.highlight {
    color: var(--primary-600);
}

.value-row {
    display: flex;
    align-items: center;
    gap: 0.5rem;
}

.link-icon {
    color: var(--primary-500);
    display: flex;
}

.link-icon:hover {
    color: var(--primary-700);
}

.add-member-btn {
    background-color: transparent;
    border: 1px solid var(--primary-color);
    color: var(--primary-color);
}

.add-member-form {
    margin-bottom: 2rem;
}

.form-row {
    display: flex;
    gap: 0.5rem;
}

.type-select {
    width: 120px;
    padding: 0.5rem;
    border: 1px solid var(--border-color);
    border-radius: var(--radius-md);
}

.input {
    flex: 1;
    padding: 0.5rem;
    border: 1px solid var(--border-color);
    border-radius: var(--radius-md);
}

.btn-text {
    color: var(--text-muted);
}

.expense-card {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 1rem;
    border-left: 4px solid var(--primary-color);
}

.expense-main {
    display: flex;
    flex-direction: column;
}

.desc {
    font-weight: 600;
    color: var(--text-main);
}

.split-card {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 1rem;
    background-color: #fff9e6;
    border: 1px solid #fde68a;
}

.split-actions {
    display: flex;
    gap: 0.5rem;
}

.btn-ghost {
    background: none;
    border: none;
    color: var(--text-muted);
}

.member-list {
    list-style: none;
    padding: 0;
    margin: 0;
}

.member-item {
    display: flex;
    align-items: center;
    gap: 1rem;
    padding: 0.75rem 0;
    border-bottom: 1px solid var(--border-color);
}

.member-avatar {
    width: 40px;
    height: 40px;
    background: var(--primary-color);
    color: white;
    border-radius: 50%;
    display: flex;
    justify-content: center;
    align-items: center;
    font-weight: bold;
}

.member-details {
    flex: 1;
}

.member-details .name {
    display: block;
    font-weight: 600;
}

.role {
    font-size: 0.75rem;
    color: var(--text-muted);
    display: flex;
    align-items: center;
    gap: 0.5rem;
}

.badge {
    background-color: var(--slate-100);
    color: var(--slate-600);
    padding: 0.125rem 0.375rem;
    border-radius: 4px;
    font-size: 0.7rem;
}

.btn-xs {
    padding: 0.25rem;
}

.btn-outline {
    background: none;
    border: 1px solid var(--border-color);
    color: var(--text-muted);
}

.btn-outline:hover {
    border-color: var(--primary-color);
    color: var(--primary-color);
}

.mb-8 {
    margin-bottom: 2rem;
}

/* Tabs */
.tabs {
    display: flex;
    gap: 1rem;
    border-bottom: 1px solid var(--border-color);
    margin-bottom: 1.5rem;
}

.tab-btn {
    background: none;
    border: none;
    padding: 0.75rem 1rem;
    font-weight: 600;
    color: var(--text-muted);
    cursor: pointer;
    border-bottom: 2px solid transparent;
}

.tab-btn.active {
    color: var(--primary-color);
    border-bottom-color: var(--primary-color);
}

.tab-btn:hover {
    color: var(--text-main);
}

/* Bill List */
.bill-card {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 1rem;
    margin-bottom: 1rem;
    border-left: 4px solid var(--secondary-color);
    cursor: pointer;
    transition: transform 0.1s;
}

.bill-card:hover {
    transform: translateY(-2px);
    box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.1);
}

.bill-info {
    display: flex;
    flex-direction: column;
}

.bill-title {
    font-weight: 600;
    color: var(--text-main);
}

.bill-date {
    font-size: 0.8rem;
    color: var(--text-muted);
}

.bill-amount {
    display: flex;
    flex-direction: column;
    align-items: flex-end;
    font-weight: 600;
    color: var(--text-main);
    gap: 0.25rem;
}
</style>

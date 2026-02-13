<script setup lang="ts">
import { ref, onMounted } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { socket } from '../socket';
import { useAuthStore } from '../stores/auth';
import AddExpenseModal from '../components/AddExpenseModal.vue';
import MainLayout from './MainLayout.vue';

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
const newMemberEmail = ref('');
const showAddMember = ref(false);
const showAddExpense = ref(false);

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
    if (!confirm('Mark this debt as settled?')) return;
    socket.emit('expense:settle', { expenseId, userId }, (res: any) => {
        if (res.status === 'ok') {
            fetchData();
        } else {
            alert(res.message);
        }
    });
};

const addMember = () => {
    if (!newMemberEmail.value) return;
    socket.emit('group:add_member', { groupId, email: newMemberEmail.value }, (res: any) => {
        if (res.status === 'ok') {
            newMemberEmail.value = '';
            showAddMember.value = false;
            fetchData(); // Refresh member list
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
                    <button @click="router.push('/groups')" class="back-btn">← Back</button>
                    <h1>{{ group.name }}</h1>
                </div>
                <div class="actions">
                    <button @click="showAddMember = !showAddMember" class="btn btn-secondary add-member-btn">Add
                        Member</button>
                    <button @click="showAddExpense = true" class="btn btn-primary">Add Expense</button>
                </div>
            </header>

            <AddExpenseModal v-if="showAddExpense" :group-id="groupId" :members="members"
                @close="showAddExpense = false" @added="handleExpenseAdded" />

            <div v-if="showAddMember" class="add-member-form card">
                <h3>Invite New Member</h3>
                <div class="form-row">
                    <input v-model="newMemberEmail" type="email" placeholder="Member Email" class="input" />
                    <button @click="addMember" class="btn btn-primary">Invite</button>
                    <button @click="showAddMember = false" class="btn btn-text">Cancel</button>
                </div>
            </div>

            <div class="grid grid-cols-1 lg:grid-cols-3 gap-8">
                <!-- Main Content: Expenses & Debts -->
                <div class="lg:col-span-2 space-y-8">
                    <section class="expenses-section">
                        <div class="section-header">
                            <h2>Expenses</h2>
                        </div>

                        <div v-if="expenses.length === 0" class="empty-state">
                            No expenses recorded yet.
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

                    <section class="debts-section">
                        <div class="section-header">
                            <h2>Unpaid Debts</h2>
                        </div>

                        <div v-if="splits.length === 0" class="empty-state">
                            All settled! No outstanding debts.
                        </div>
                        <div v-else class="split-list">
                            <div v-for="split in splits" :key="split.expense_id + split.user_id"
                                class="split-card card">
                                <div class="split-info">
                                    <div class="split-text">
                                        <span class="user-highlight">{{(members.find(m => m.id ===
                                            split.user_id))?.username }}</span>
                                        owes
                                        <span class="user-highlight">{{ split.payer_name }}</span>
                                    </div>
                                    <div class="amount-row">
                                        <span class="amount text-danger">${{ split.amount_owed }}</span>
                                        <span class="reason">For: {{ split.description }}</span>
                                    </div>
                                </div>
                                <button
                                    v-if="split.user_id === authStore.user?.id || group.created_by === authStore.user?.id"
                                    @click="settleExpense(split.expense_id, split.user_id)"
                                    class="btn btn-sm btn-success">
                                    Settle
                                </button>
                            </div>
                        </div>
                    </section>
                </div>

                <!-- Sidebar: Members -->
                <aside class="members-sidebar">
                    <div class="card">
                        <h2>Members</h2>
                        <ul class="member-list">
                            <li v-for="member in members" :key="member.id" class="member-item">
                                <div class="member-avatar">{{ member.username[0].toUpperCase() }}</div>
                                <div class="member-details">
                                    <span class="name">{{ member.username }}</span>
                                    <span class="role">{{ member.role }}</span>
                                </div>
                            </li>
                        </ul>
                    </div>
                </aside>
            </div>
        </div>
        <div v-else-if="loading" class="state-container">Loading group details...</div>
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
    border: 1px solid var(--text-muted);
    padding: 0.25rem 0.75rem;
    border-radius: var(--radius-sm);
    color: var(--text-muted);
    font-size: 0.875rem;
}

.actions {
    display: flex;
    gap: 0.5rem;
}

.add-member-btn {
    margin-right: 0.5rem;
    background-color: transparent;
    border: 1px solid var(--primary-color);
    color: var(--primary-color);
}

.add-member-form {
    margin-bottom: 2rem;
}

.add-member-form h3 {
    margin-top: 0;
    margin-bottom: 1rem;
    font-size: 1rem;
}

.form-row {
    display: flex;
    gap: 0.5rem;
}

.input {
    flex: 1;
    padding: 0.5rem;
    border: 1px solid #d1d5db;
    border-radius: var(--radius-md);
}

.btn-text {
    color: var(--text-muted);
}

.section-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 1rem;
}

.section-header h2 {
    font-size: 1.25rem;
    font-weight: 600;
    margin: 0;
    color: var(--text-main);
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

.date {
    font-size: 0.85rem;
    color: var(--text-muted);
}

.expense-amount {
    font-weight: 700;
    font-size: 1.125rem;
    color: var(--text-main);
}

.debts-section {
    margin-top: 3rem;
    padding-top: 2rem;
    border-top: 1px dashed #e5e7eb;
}

.split-card {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 1rem;
    background-color: #fff9e6;
    /* Keeping the warning bg for debts */
    border: 1px solid #fde68a;
}

.split-text {
    margin-bottom: 0.25rem;
}

.user-highlight {
    font-weight: 600;
    color: var(--text-main);
}

.amount-row {
    display: flex;
    align-items: center;
    gap: 0.75rem;
}

.text-danger {
    color: var(--danger-color);
    font-weight: 700;
}

.reason {
    font-size: 0.85rem;
    color: var(--text-muted);
}

.btn-sm {
    padding: 0.25rem 0.75rem;
    font-size: 0.875rem;
}

.btn-success {
    background-color: var(--secondary-color);
    color: white;
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
    border-bottom: 1px solid #f3f4f6;
}

.member-item:last-child {
    border-bottom: none;
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
    font-size: 1rem;
}

.member-details .name {
    display: block;
    font-weight: 600;
}

.member-details .role {
    font-size: 0.75rem;
    color: var(--text-muted);
    text-transform: capitalize;
}

.empty-state {
    text-align: center;
    padding: 3rem;
    color: var(--text-muted);
    background: var(--bg-surface);
    border-radius: var(--radius-lg);
    border: 1px dashed #d1d5db;
}

.state-container {
    padding: 4rem;
    text-align: center;
    color: var(--text-muted);
}

.error {
    color: var(--danger-color);
}
</style>

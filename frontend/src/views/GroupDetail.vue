<script setup lang="ts">
import { ref, computed, onMounted } from 'vue';
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

// Search & Filter State
const searchQuery = ref('');
const memberFilter = ref('');
const startDate = ref('');
const endDate = ref('');

const filteredExpenses = computed(() => {
    return expenses.value.filter(e => {
        // Text Match
        const matchText = searchQuery.value
            ? e.description.toLowerCase().includes(searchQuery.value.toLowerCase()) ||
            e.amount.toString().includes(searchQuery.value)
            : true;

        // Member Match (Paid By)
        const matchMember = memberFilter.value
            ? e.paid_by === memberFilter.value
            : true;

        // Date Match
        let matchDate = true;
        if (startDate.value) {
            matchDate = matchDate && new Date(e.date) >= new Date(startDate.value);
        }
        if (endDate.value) {
            matchDate = matchDate && new Date(e.date) <= new Date(endDate.value + 'T23:59:59');
        }

        return matchText && matchMember && matchDate;
    });
});

const filteredBills = computed(() => {
    return bills.value.filter(b => {
        // Text Match
        const matchText = searchQuery.value
            ? b.title.toLowerCase().includes(searchQuery.value.toLowerCase()) ||
            b.total_amount.toString().includes(searchQuery.value)
            : true;

        // Date Match (Issue Date)
        let matchDate = true;
        if (startDate.value) {
            matchDate = matchDate && new Date(b.issue_date) >= new Date(startDate.value);
        }
        if (endDate.value) {
            matchDate = matchDate && new Date(b.issue_date) <= new Date(endDate.value + 'T23:59:59');
        }

        return matchText && matchDate;
    });
});

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

const exportData = async (type: 'expenses' | 'bills') => {
    try {
        const url = `/api/export/group/${groupId}/${type}`;
        // Use window.open if relying purely on cookies, 
        // OR use fetch/blob if needing to handle errors gracefully in UI.
        // Since we rely on http proxy or same domain, and cookies are httpOnly, 
        // window.open triggers GET request which includes cookies.
        window.open(url, '_blank');
    } catch (e) {
        alert('Export failed');
    }
};

const deleteGroup = () => {
    if (!confirm(t('groups.confirmDelete'))) return;
    socket.emit('group:delete', { groupId }, (res: any) => {
        if (res.status === 'ok') {
            router.push('/groups');
        } else {
            alert(res.message);
        }
    });
};

const leaveGroup = () => {
    if (!confirm(t('groups.confirmLeave'))) return;
    socket.emit('group:leave', { groupId }, (res: any) => {
        if (res.status === 'ok') {
            router.push('/groups');
        } else {
            alert(res.message);
        }
    });
};
</script>

<template>
    <MainLayout>
        <div class="group-detail animate-fade-in" v-if="!loading && group">
            <!-- Header Area -->
            <header class="flex flex-col md:flex-row justify-between items-start md:items-center gap-4 mb-8">
                <div class="flex items-center gap-4">
                    <button @click="router.push('/groups')"
                        class="p-2 rounded-xl bg-white/50 hover:bg-white text-slate-500 hover:text-primary-600 transition-all shadow-sm">
                        <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" fill="none" viewBox="0 0 24 24"
                            stroke="currentColor">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                                d="M10 19l-7-7 7-7m-7 7h18" />
                        </svg>
                    </button>

                    <!-- Group Icon -->
                    <div
                        class="h-16 w-16 rounded-2xl bg-white shadow-sm border border-slate-100 flex items-center justify-center overflow-hidden shrink-0">
                        <img v-if="group.icon_url" :src="group.icon_url" alt="Icon"
                            class="w-full h-full object-cover" />
                        <span v-else class="text-2xl font-bold text-primary-500">{{ group.name.charAt(0).toUpperCase()
                        }}</span>
                    </div>

                    <div>
                        <h1 class="text-3xl font-extrabold text-slate-800">{{ group.name }}</h1>
                        <div class="flex items-center gap-2 text-sm text-slate-500">
                            <span
                                class="px-2 py-0.5 rounded-full bg-primary-50 text-primary-700 font-bold uppercase text-xs">{{
                                    group.service_name || t('groups.genericService') }}</span>
                            <span>•</span>
                            <span>{{ t('groups.createdOnLabel') }} {{ new Date(group.created_at).toLocaleDateString()
                                }}</span>
                        </div>
                    </div>
                </div>
                <div class="flex gap-3">
                    <div class="relative group">
                        <button class="btn bg-white text-slate-600 hover:bg-slate-50 border border-slate-200 shadow-sm">
                            <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" fill="none" viewBox="0 0 24 24"
                                stroke="currentColor">
                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                                    d="M12 10v6m0 0l-3-3m3 3l3-3m2 8H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
                            </svg>
                            {{ t('common.actions.export', 'Export') }}
                        </button>
                        <div
                            class="absolute right-0 mt-2 w-48 bg-white rounded-xl shadow-xl border border-slate-100 overflow-hidden hidden group-hover:block z-20 animate-fade-in-up">
                            <a @click="exportData('expenses')"
                                class="block px-4 py-3 hover:bg-slate-50 text-slate-700 cursor-pointer flex items-center gap-2">
                                <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4 text-primary-500" fill="none"
                                    viewBox="0 0 24 24" stroke="currentColor">
                                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                                        d="M9 17v-2m3 2v-4m3 4v-6m2 10H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
                                </svg>
                                {{ t('groups.expenses') }} (CSV)
                            </a>
                            <a @click="exportData('bills')"
                                class="block px-4 py-3 hover:bg-slate-50 text-slate-700 cursor-pointer flex items-center gap-2 border-t border-slate-100">
                                <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4 text-indigo-500" fill="none"
                                    viewBox="0 0 24 24" stroke="currentColor">
                                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                                        d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
                                </svg>
                                {{ t('groups.bills') }} (CSV)
                            </a>
                        </div>
                    </div>
                    <button @click="showAddMember = !showAddMember"
                        class="btn bg-white text-primary-600 hover:bg-primary-50 border border-primary-200 shadow-sm">
                        <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" fill="none" viewBox="0 0 24 24"
                            stroke="currentColor">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                                d="M18 9v3m0 0v3m0-3h3m-3 0h-3m-2-5a4 4 0 11-8 0 4 4 0 018 0zM3 20a6 6 0 0112 0v1H3v-1z" />
                        </svg>
                        {{ t('groups.addMember') }}
                    </button>
                    <button @click="showAddExpense = true"
                        class="btn btn-primary shadow-lg hover:shadow-xl hover:-translate-y-0.5 transition-all">
                        <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" fill="none" viewBox="0 0 24 24"
                            stroke="currentColor">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                                d="M12 6v6m0 0v6m0-6h6m-6 0H6" />
                        </svg>
                        {{ t('groups.addExpense') }}
                    </button>
                </div>
            </header>

            <!-- Group Info Cards -->
            <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4 mb-8">
                <div class="glass-panel p-4 flex flex-col justify-center relative overflow-hidden">
                    <div class="absolute right-0 bottom-0 opacity-5 transform translate-x-1/4 translate-y-1/4">
                        <svg xmlns="http://www.w3.org/2000/svg" class="h-24 w-24" fill="currentColor"
                            viewBox="0 0 24 24">
                            <path d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14"
                                stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" />
                        </svg>
                    </div>
                    <span class="text-xs uppercase font-bold text-slate-400 mb-1">{{ t('groups.serviceName') }}</span>
                    <div class="flex items-center gap-2">
                        <span class="text-lg font-bold text-slate-800">{{ group.service_name || '---' }}</span>
                        <a v-if="group.website" :href="group.website" target="_blank"
                            class="text-primary-500 hover:text-primary-600">
                            <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24"
                                stroke="currentColor">
                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                                    d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" />
                            </svg>
                        </a>
                    </div>
                </div>
                <div class="glass-panel p-4 flex flex-col justify-center">
                    <span class="text-xs uppercase font-bold text-slate-400 mb-1">{{ t('groups.planName') }}</span>
                    <span class="text-lg font-bold text-slate-800">{{ group.plan_name || '---' }}</span>
                </div>
                <div
                    class="glass-panel p-4 flex flex-col justify-center bg-primary-gradient text-white relative overflow-hidden">
                    <div class="absolute -right-4 -top-4 bg-white/20 w-16 h-16 rounded-full blur-xl"></div>
                    <span class="text-xs uppercase font-bold text-primary-100 mb-1">{{ t('groups.amount') }}</span>
                    <span class="text-2xl font-extrabold">{{ group.service_currency }} {{ group.amount }} <span
                            class="text-sm font-medium opacity-80">/ {{ group.billing_type === 'recurring' ?
                                t(`common.cycles.${group.interval_unit}`) : t('common.cycles.once')
                            }}</span></span>
                </div>
                <div class="glass-panel p-4 flex flex-col justify-center">
                    <span class="text-xs uppercase font-bold text-slate-400 mb-1">{{ t('groups.billingMethod') }}</span>
                    <span class="text-lg font-bold text-slate-800 flex items-center gap-2">
                        <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5 text-slate-400" fill="none"
                            viewBox="0 0 24 24" stroke="currentColor">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                                d="M9 7h6m0 10v-3m-3 3h.01M9 17h.01M9 14h.01M12 14h.01M15 11h.01M12 11h.01M9 11h.01M7 21h10a2 2 0 002-2V5a2 2 0 00-2-2H7a2 2 0 00-2 2v14a2 2 0 002 2z" />
                        </svg>
                        {{ t(`groups.methods.${group.billing_method}`) }}
                    </span>
                </div>
            </div>

            <!-- Add Member Form (Toggleable) -->
            <div v-if="showAddMember" class="glass-panel p-6 mb-8 animate-fade-in border-primary-200 border-2">
                <h3 class="text-lg font-bold mb-4 text-primary-700">{{ t('groups.inviteNewMember') }}</h3>
                <div class="flex gap-2">
                    <select v-model="newMemberType"
                        class="bg-white border rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-primary-300 outline-none">
                        <option value="name">{{ t('common.fields.name') }}</option>
                        <option value="email">{{ t('common.fields.email') }}</option>
                    </select>
                    <input v-model="newMemberValue" type="text" :placeholder="t('common.placeholders.member')"
                        class="glass-input flex-1 bg-white" />
                    <button @click="addMember" class="btn btn-primary">{{ t('common.actions.add') }}</button>
                    <button @click="showAddMember = false" class="btn bg-slate-100 text-slate-600 hover:bg-slate-200">{{
                        t('common.actions.cancel') }}</button>
                </div>
            </div>

            <!-- Main Grid Layout -->
            <div class="grid grid-cols-1 lg:grid-cols-3 gap-8">
                <!-- Left Column: Expenses & Bills -->
                <div class="lg:col-span-2 space-y-6">
                    <!-- Search & Filter Bar -->
                    <div class="glass-panel p-4 mb-4 flex flex-wrap gap-4 items-end animate-fade-in"
                        style="animation-delay: 0.25s">
                        <div class="flex-1 min-w-[200px]">
                            <label class="form-label">{{ t('common.fields.search') }}</label>
                            <input v-model="searchQuery" type="text" :placeholder="t('common.placeholders.search')"
                                class="glass-input" />
                        </div>
                        <div class="w-40">
                            <label class="form-label">{{ t('groups.payer', 'Payer') }}</label>
                            <select v-model="memberFilter" class="glass-input">
                                <option value="">{{ t('common.all') }}</option>
                                <option v-for="m in members" :key="m.id" :value="m.user_id">{{ m.username }}</option>
                            </select>
                        </div>
                        <div class="w-40">
                            <label class="form-label">{{ t('common.fields.startDate') }}</label>
                            <input v-model="startDate" type="date" class="glass-input" />
                        </div>
                        <div class="w-40">
                            <label class="form-label">{{ t('common.fields.endDate') }}</label>
                            <input v-model="endDate" type="date" class="glass-input" />
                        </div>
                        <div class="flex gap-2">
                            <button @click="exportData('expenses')"
                                class="btn bg-slate-100 text-slate-600 hover:bg-slate-200"
                                :title="t('common.actions.export')">
                                <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" fill="none" viewBox="0 0 24 24"
                                    stroke="currentColor">
                                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                                        d="M12 10v6m0 0l-3-3m3 3l3-3m2 8H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
                                </svg>
                            </button>
                            <button @click="showAddExpense = true" class="btn btn-primary">
                                <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" fill="none" viewBox="0 0 24 24"
                                    stroke="currentColor">
                                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                                        d="M12 4v16m8-8H4" />
                                </svg>
                                {{ t('groups.addExpense') }}
                            </button>
                        </div>
                    </div>

                    <!-- Tabs -->
                    <div class="flex gap-1 bg-slate-100 p-1 rounded-xl w-fit mb-6">
                        <button @click="currentTab = 'expenses'"
                            :class="['px-4 py-2 rounded-lg text-sm font-bold transition-all', currentTab === 'expenses' ? 'bg-white text-primary-600 shadow-sm' : 'text-slate-500 hover:text-slate-700']">
                            {{ t('groups.expenses') }}
                        </button>
                        <button @click="currentTab = 'bills'"
                            :class="['px-4 py-2 rounded-lg text-sm font-bold transition-all', currentTab === 'bills' ? 'bg-white text-primary-600 shadow-sm' : 'text-slate-500 hover:text-slate-700']">
                            {{ t('groups.bills') }}
                        </button>
                        <button @click="currentTab = 'settings'"
                            :class="['px-4 py-2 rounded-lg text-sm font-bold transition-all', currentTab === 'settings' ? 'bg-white text-primary-600 shadow-sm' : 'text-slate-500 hover:text-slate-700']">
                            {{ t('groups.settings') }}
                        </button>
                    </div>

                    <div v-if="currentTab === 'settings'" class="animate-fade-in space-y-8">
                        <!-- Group Information -->
                        <section class="glass-panel p-6">
                            <h3 class="text-lg font-bold text-slate-800 mb-4">{{ t('groups.basicInfo') }}</h3>
                            <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
                                <div>
                                    <label class="text-xs font-bold text-slate-400 uppercase">{{ t('groups.startDate')
                                    }}</label>
                                    <p class="font-medium text-slate-800">{{ group.start_date ? new
                                        Date(group.start_date).toLocaleDateString() : '---' }}</p>
                                </div>
                                <div>
                                    <label class="text-xs font-bold text-slate-400 uppercase">{{
                                        t('groups.endCondition') }}</label>
                                    <p class="font-medium text-slate-800">
                                        {{ t(`groups.endConditions.${group.end_condition || 'indefinite'}`) }}
                                        <span v-if="group.end_condition === 'date' && group.end_value">
                                            ({{ new Date(group.end_value).toLocaleDateString() }})
                                        </span>
                                        <span v-if="group.end_condition === 'total_amount' && group.end_value">
                                            ({{ group.end_value }})
                                        </span>
                                    </p>
                                </div>
                                <div>
                                    <label class="text-xs font-bold text-slate-400 uppercase">{{ t('groups.maxMembers')
                                    }}</label>
                                    <p class="font-medium text-slate-800">{{ group.max_members }}</p>
                                </div>
                                <div>
                                    <label class="text-xs font-bold text-slate-400 uppercase">{{
                                        t('groups.createdOnLabel') }}</label>
                                    <p class="font-medium text-slate-800">{{ new
                                        Date(group.created_at).toLocaleDateString() }}</p>
                                </div>
                            </div>
                        </section>

                        <!-- Danger Zone -->
                        <section class="glass-panel p-6 border-l-4 border-red-500">
                            <h3 class="text-lg font-bold text-red-600 mb-4">{{ t('groups.dangerZone') }}</h3>
                            <div class="flex flex-col gap-4">
                                <div v-if="group.created_by === authStore.user?.id">
                                    <p class="text-sm text-slate-600 mb-2">{{ t('groups.confirmDelete') }}</p>
                                    <button @click="deleteGroup"
                                        class="btn bg-red-50 text-red-600 hover:bg-red-100 border border-red-200">
                                        {{ t('groups.deleteGroup') }}
                                    </button>
                                </div>
                                <div v-else>
                                    <p class="text-sm text-slate-600 mb-2">{{ t('groups.confirmLeave') }}</p>
                                    <button @click="leaveGroup"
                                        class="btn bg-red-50 text-red-600 hover:bg-red-100 border border-red-200">
                                        {{ t('groups.leaveGroup') }}
                                    </button>
                                </div>
                            </div>
                        </section>
                    </div>

                    <div v-if="currentTab === 'expenses'" class="animate-fade-in space-y-8">
                        <!-- Expense List -->
                        <section>
                            <h2 class="text-xl font-bold text-slate-800 mb-4">{{ t('groups.expenses') }}</h2>
                            <div v-if="filteredExpenses.length === 0"
                                class="flex flex-col items-center justify-center p-12 glass-panel border-dashed border-2 border-slate-200">
                                <p class="text-slate-400">
                                    {{ expenses.length > 0 ? t('groups.noSearchResults') :
                                        t('groups.noExpenses') }}
                                </p>
                            </div>
                            <div v-else class="space-y-3">
                                <div v-for="expense in filteredExpenses" :key="expense.id"
                                    class="glass-card p-4 flex justify-between items-center group hover:bg-white transition-colors">
                                    <div class="flex items-center gap-4">
                                        <div
                                            class="w-10 h-10 rounded-full bg-slate-100 flex items-center justify-center text-slate-400 group-hover:bg-primary-50 group-hover:text-primary-500 transition-colors">
                                            <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" fill="none"
                                                viewBox="0 0 24 24" stroke="currentColor">
                                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                                                    d="M12 8c-1.657 0-3 .895-3 2s1.343 2 3 2 3 .895 3 2-1.343 2-3 2m0-8c1.11 0 2.08.402 2.599 1M12 8V7m0 1v8m0 0v1m0-1c-1.11 0-2.08-.402-2.599-1M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                                            </svg>
                                        </div>
                                        <div>
                                            <p class="font-bold text-slate-800">{{ expense.description }}</p>
                                            <p class="text-xs text-slate-500">{{ new
                                                Date(expense.date).toLocaleDateString() }}</p>
                                        </div>
                                    </div>
                                    <div class="font-bold text-lg text-slate-800">
                                        ${{ expense.amount }}
                                    </div>
                                </div>
                            </div>
                        </section>

                        <!-- Debts Section -->
                        <section>
                            <h2 class="text-xl font-bold text-slate-800 mb-4">{{ t('groups.unpaidDebts') }}</h2>
                            <div v-if="splits.length === 0"
                                class="flex flex-col items-center justify-center p-12 glass-panel bg-green-50/50 border-dashed border-2 border-green-100">
                                <svg xmlns="http://www.w3.org/2000/svg" class="h-12 w-12 text-green-300 mb-2"
                                    fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                                        d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
                                </svg>
                                <p class="text-green-600 font-medium">{{ t('groups.settled') }}</p>
                            </div>
                            <div v-else class="grid grid-cols-1 md:grid-cols-2 gap-4">
                                <div v-for="split in splits" :key="split.expense_id + split.user_id"
                                    class="glass-card p-4 border-l-4 border-yellow-400 bg-yellow-50/30">
                                    <div class="flex justify-between items-start mb-3">
                                        <div class="text-sm">
                                            <span class="font-bold text-slate-800">{{members.find(m => m.user_id ===
                                                split.user_id)?.username || split.user_id}}</span>
                                            <span class="text-slate-400 mx-1">{{ t('groups.owe') }}</span>
                                            <span class="font-bold text-slate-800">{{ split.payer_name }}</span>
                                        </div>
                                        <button @click="viewBill(split)"
                                            class="text-slate-400 hover:text-primary-600 transition-colors"
                                            :title="t('groups.viewBill')">
                                            <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" fill="none"
                                                viewBox="0 0 24 24" stroke="currentColor">
                                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                                                    d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
                                            </svg>
                                        </button>
                                    </div>
                                    <div class="flex justify-between items-end">
                                        <div>
                                            <div class="text-red-500 font-extrabold text-xl">${{ split.amount_owed }}
                                            </div>
                                            <div class="text-xs text-slate-400 mt-1">{{ split.description }}</div>
                                        </div>
                                        <button
                                            v-if="split.user_id === authStore.user?.id || group.created_by === authStore.user?.id"
                                            @click="settleExpense(split.expense_id, split.user_id)"
                                            class="px-3 py-1 rounded-md bg-green-100 text-green-700 text-xs font-bold hover:bg-green-200 transition-colors">
                                            {{ t('groups.settle') }}
                                        </button>
                                    </div>
                                </div>
                            </div>
                        </section>
                    </div>

                    <div v-else-if="currentTab === 'bills'" class="animate-fade-in">
                        <h2 class="text-xl font-bold text-slate-800 mb-4">{{ t('groups.generatedBills') }}</h2>
                        <div v-if="filteredBills.length === 0"
                            class="flex flex-col items-center justify-center p-12 glass-panel border-dashed border-2 border-slate-200">
                            <p class="text-slate-400">
                                {{ bills.length > 0 ? t('groups.noSearchResults') :
                                    t('groups.noBills') }}
                            </p>
                        </div>
                        <div v-else class="space-y-3">
                            <div v-for="bill in filteredBills" :key="bill.id"
                                class="glass-card p-4 flex justify-between items-center cursor-pointer hover:bg-primary-50/50 transition-colors border-l-4 border-indigo-500"
                                @click="viewBillDetail(bill)">
                                <div>
                                    <p class="font-bold text-slate-800">{{ bill.title }}</p>
                                    <p class="text-xs text-slate-500">{{ t('groups.dueDate') }}: {{ new
                                        Date(bill.due_date).toLocaleDateString() }}</p>
                                </div>
                                <div class="text-right">
                                    <div class="font-bold text-slate-800">{{ bill.currency }} {{ bill.total_amount }}
                                    </div>
                                    <span class="text-xs px-2 py-0.5 rounded-full font-bold uppercase" :class="{
                                        'bg-green-100 text-green-700': bill.status === 'paid',
                                        'bg-yellow-100 text-yellow-700': bill.status === 'pending',
                                        'bg-red-100 text-red-700': bill.status === 'overdue'
                                    }">
                                        {{ t(`groups.status_${bill.status}`) }}
                                    </span>
                                </div>
                            </div>
                        </div>
                    </div>
                </div>

                <!-- Right Column: Sidebar (Members & Files) -->
                <aside class="space-y-6">
                    <FileManager :group-id="groupId" />

                    <div class="glass-panel p-6">
                        <h2 class="text-lg font-bold text-slate-800 mb-4 flex items-center gap-2">
                            <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5 text-primary-500" fill="none"
                                viewBox="0 0 24 24" stroke="currentColor">
                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                                    d="M12 4.354a4 4 0 110 5.292M15 21H3v-1a6 6 0 0112 0v1zm0 0h6v-1a6 6 0 00-9-5.197M13 7a4 4 0 11-8 0 4 4 0 018 0z" />
                            </svg>
                            {{ t('groups.members') }}
                        </h2>
                        <ul class="space-y-4">
                            <li v-for="member in members" :key="member.member_id" class="flex items-center gap-3">
                                <div
                                    class="w-10 h-10 rounded-full bg-slate-200 text-slate-500 flex items-center justify-center font-bold">
                                    {{ (member.username || '?')[0].toUpperCase() }}
                                </div>
                                <div class="flex-1 min-w-0">
                                    <p class="text-sm font-bold text-slate-800 truncate">{{ member.username }}</p>
                                    <p class="text-xs text-slate-500 flex items-center gap-1">
                                        {{ member.role }}
                                        <span v-if="member.temp_name"
                                            class="text-yellow-600 bg-yellow-50 px-1 rounded">({{ t('groups.nonMember')
                                            }})</span>
                                    </p>
                                </div>
                                <button v-if="member.temp_name" @click="bindAccount(member.member_id)"
                                    class="text-primary-500 hover:text-primary-700" :title="t('groups.bindAccount')">
                                    <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" fill="none"
                                        viewBox="0 0 24 24" stroke="currentColor">
                                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                                            d="M13.828 10.172a4 4 0 00-5.656 0l-4 4a4 4 0 105.656 5.656l1.102-1.101" />
                                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                                            d="M10.172 13.828a4 4 0 005.656 0l4-4a4 4 0 00-5.656-5.656l-1.102 1.101" />
                                    </svg>
                                </button>
                            </li>
                        </ul>
                    </div>
                </aside>
            </div>

            <!-- Modals -->
            <AddExpenseModal v-if="showAddExpense" :group-id="groupId" :members="members"
                @close="showAddExpense = false" @added="handleExpenseAdded" />

            <BillTicket v-if="showBill && selectedSplit" :group="group"
                :member="members.find(m => m.user_id === selectedSplit.user_id)" :split="selectedSplit" :show="showBill"
                @close="showBill = false" />

            <BillDetailModal v-if="showBillDetail && selectedBill" :bill="selectedBill" :splits="selectedBillSplits"
                :is-host="group.created_by === authStore.user?.id" @close="showBillDetail = false"
                @updated="viewBillDetail(selectedBill)" />
        </div>

        <div v-else-if="loading" class="flex flex-col items-center justify-center p-20 min-h-screen">
            <div class="animate-spin rounded-full h-12 w-12 border-b-2 border-primary-600 mb-4"></div>
            <p class="text-slate-500 animate-pulse">{{ t('common.status.loading') }}</p>
        </div>

        <div v-else class="flex flex-col items-center justify-center p-20 min-h-screen text-center">
            <p class="text-red-500 font-bold text-lg mb-2">{{ t('common.status.error') }}</p>
            <p class="text-slate-600">{{ error }}</p>
            <button @click="router.push('/groups')" class="mt-4 btn btn-primary">{{ t('groups.backToGroups') }}</button>
        </div>
    </MainLayout>
</template>

<style scoped>
/* Scoped styles minimal, relying on global utility classes from style.css */
</style>

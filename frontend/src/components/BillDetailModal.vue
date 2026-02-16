<script setup lang="ts">
import { ref, computed } from 'vue';
import { useI18n } from 'vue-i18n';
import { socket } from '../socket';

const props = defineProps<{
    bill: any;
    splits: any[];
    isHost: boolean;
}>();

const emit = defineEmits(['close', 'updated']);

const { t } = useI18n();

const editingSplitId = ref<string | null>(null);
const editAmount = ref<number>(0);

const startEdit = (split: any) => {
    editingSplitId.value = split.id;
    editAmount.value = split.amount_owed;
};

const cancelEdit = () => {
    editingSplitId.value = null;
    editAmount.value = 0;
};

const saveSplit = () => {
    if (!editingSplitId.value) return;

    socket.emit('bill:update_split', {
        splitId: editingSplitId.value,
        amount: editAmount.value
    }, (res: any) => {
        if (res.status === 'ok') {
            emit('updated');
            cancelEdit();
        } else {
            alert(res.message);
        }
    });
};

const formatDate = (dateStr: string) => {
    return new Date(dateStr).toLocaleDateString();
};
</script>

<template>
    <div class="modal-overlay" @click.self="$emit('close')">
        <div class="modal-content">
            <header class="modal-header">
                <h2>{{ bill.title }}</h2>
                <button @click="$emit('close')" class="close-btn">&times;</button>
            </header>

            <div class="bill-summary">
                <div class="row">
                    <span class="label">{{ t('groups.amount') }}:</span>
                    <span class="value">{{ bill.currency }} {{ bill.total_amount }}</span>
                </div>
                <div class="row">
                    <span class="label">{{ t('groups.dueDate') }}:</span>
                    <span class="value">{{ formatDate(bill.due_date) }}</span>
                </div>
                <div class="row">
                    <span class="label">{{ t('groups.status') }}:</span>
                    <span class="status-badge" :class="bill.status">{{ t(`groups.status_${bill.status}`) || bill.status
                        }}</span>
                </div>
            </div>

            <div class="splits-list">
                <h3>{{ t('groups.splits') }}</h3>
                <table>
                    <thead>
                        <tr>
                            <th>{{ t('groups.member') }}</th>
                            <th>{{ t('groups.amount') }}</th>
                            <th>{{ t('groups.status') }}</th>
                            <th v-if="isHost">{{ t('common.actions.edit') }}</th>
                        </tr>
                    </thead>
                    <tbody>
                        <tr v-for="split in splits" :key="split.id">
                            <td>
                                {{ split.username || split.temp_name || 'Unknown' }}
                            </td>
                            <td>
                                <div v-if="editingSplitId === split.id" class="edit-row">
                                    <input v-model.number="editAmount" type="number" step="0.01" class="input-sm">
                                </div>
                                <span v-else>{{ bill.currency }} {{ split.amount_owed }}</span>
                            </td>
                            <td>
                                <span class="status-dot" :class="split.status"></span>
                                {{ t(`groups.status_${split.status}`) || split.status }}
                            </td>
                            <td v-if="isHost">
                                <div v-if="editingSplitId === split.id" class="actions">
                                    <button @click="saveSplit" class="btn btn-primary btn-xs">OK</button>
                                    <button @click="cancelEdit" class="btn btn-xs">X</button>
                                </div>
                                <button v-else @click="startEdit(split)" class="btn btn-outline btn-xs">
                                    <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none"
                                        viewBox="0 0 24 24" stroke="currentColor">
                                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                                            d="M15.232 5.232l3.536 3.536m-2.036-5.036a2.5 2.5 0 113.536 3.536L6.5 21.036H3v-3.572L16.732 3.732z" />
                                    </svg>
                                </button>
                            </td>
                        </tr>
                    </tbody>
                </table>
            </div>
        </div>
    </div>
</template>

<style scoped>
.modal-overlay {
    position: fixed;
    top: 0;
    left: 0;
    width: 100vw;
    height: 100vh;
    background: rgba(0, 0, 0, 0.5);
    display: flex;
    justify-content: center;
    align-items: center;
    z-index: 1000;
}

.modal-content {
    background: var(--bg-surface);
    padding: 2rem;
    border-radius: var(--radius-lg);
    width: 90%;
    max-width: 600px;
    max-height: 80vh;
    overflow-y: auto;
}

.modal-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 1.5rem;
}

.bill-summary {
    background: var(--bg-main);
    padding: 1rem;
    border-radius: var(--radius-md);
    margin-bottom: 2rem;
}

.row {
    display: flex;
    justify-content: space-between;
    margin-bottom: 0.5rem;
}

.label {
    color: var(--text-muted);
}

.value {
    font-weight: 600;
}

table {
    width: 100%;
    border-collapse: collapse;
}

th {
    text-align: left;
    color: var(--text-muted);
    padding-bottom: 0.5rem;
    border-bottom: 1px solid var(--border-color);
}

td {
    padding: 0.75rem 0;
    border-bottom: 1px solid var(--border-color);
}

.input-sm {
    width: 80px;
    padding: 0.25rem;
    border: 1px solid var(--border-color);
    border-radius: 4px;
}

.actions {
    display: flex;
    gap: 0.25rem;
}

.status-badge {
    padding: 0.125rem 0.375rem;
    border-radius: 4px;
    font-size: 0.75rem;
    background: var(--slate-100);
}

.status-badge.paid {
    background: #dcfce7;
    color: #166534;
}

.status-badge.pending {
    background: #fef9c3;
    color: #854d0e;
}

.status-badge.overdue {
    background: #fee2e2;
    color: #991b1b;
}

.btn-xs {
    padding: 0.25rem 0.5rem;
    font-size: 0.75rem;
}
</style>

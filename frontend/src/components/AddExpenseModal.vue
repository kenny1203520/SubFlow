<script setup lang="ts">
import { ref } from 'vue';
import { socket } from '../socket';

const props = defineProps<{
  groupId: string;
  members: any[];
}>();

const emit = defineEmits(['close', 'added']);

const description = ref('');
const totalAmount = ref<number | null>(null);
const isSubmitting = ref(false);
const error = ref('');

// Simplified split: Equal split among all members
const handleSubmit = () => {
  if (!description.value || !totalAmount.value || totalAmount.value <= 0) return;

  isSubmitting.value = true;
  error.value = '';

  const splitAmount = totalAmount.value / props.members.length;
  const splits = props.members.map(m => ({
    userId: m.id,
    amount: splitAmount
  }));

  socket.emit('expense:add', {
    groupId: props.groupId,
    amount: totalAmount.value,
    description: description.value,
    splits
  }, (res: any) => {
    isSubmitting.value = false;
    if (res.status === 'ok') {
      emit('added', res.expense);
    } else {
      error.value = res.message || 'Failed to add expense';
    }
  });
};
</script>

<template>
  <div class="modal-overlay" @click.self="$emit('close')">
    <div class="modal-content">
      <h2>Add Expense</h2>
      <form @submit.prevent="handleSubmit">
        <div class="form-group">
          <label>Description</label>
          <input v-model="description" type="text" placeholder="e.g. Dinner, Groceries" required :disabled="isSubmitting" />
        </div>
        <div class="form-group">
          <label>Total Amount</label>
          <input v-model="totalAmount" type="number" step="0.01" placeholder="0.00" required :disabled="isSubmitting" />
        </div>
        
        <div class="split-info">
          <p>This will be split equally among all {{ members.length }} members.</p>
          <p v-if="totalAmount">Each owes: <strong>${{ (totalAmount / members.length).toFixed(2) }}</strong></p>
        </div>

        <p v-if="error" class="error-msg">{{ error }}</p>
        
        <div class="actions">
          <button type="button" @click="$emit('close')" :disabled="isSubmitting" class="cancel-btn">Cancel</button>
          <button type="submit" :disabled="isSubmitting || !totalAmount" class="submit-btn">
            {{ isSubmitting ? 'Adding...' : 'Add Expense' }}
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
  background: rgba(0,0,0,0.5);
  display: flex;
  justify-content: center;
  align-items: center;
  z-index: 1000;
}
.modal-content {
  background: white;
  padding: 2rem;
  border-radius: 8px;
  width: 100%;
  max-width: 450px;
}
.form-group {
  margin-bottom: 1.25rem;
}
label {
  display: block;
  margin-bottom: 0.5rem;
  font-weight: bold;
}
input {
  width: 100%;
  padding: 0.75rem;
  border: 1px solid #ddd;
  border-radius: 4px;
}
.split-info {
  background: #f8f9fa;
  padding: 1rem;
  border-radius: 4px;
  margin-bottom: 1.25rem;
  font-size: 0.9rem;
}
.error-msg {
  color: #e74c3c;
  margin-bottom: 1rem;
}
.actions {
  display: flex;
  justify-content: flex-end;
  gap: 1rem;
}
.submit-btn {
  background-color: #4CAF50;
  color: white;
  border: none;
  padding: 0.75rem 1.5rem;
  border-radius: 4px;
  cursor: pointer;
  font-weight: bold;
}
.cancel-btn {
  background: none;
  border: 1px solid #ddd;
  padding: 0.75rem 1.5rem;
  border-radius: 4px;
  cursor: pointer;
}
</style>

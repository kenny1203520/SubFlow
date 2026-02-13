<script setup lang="ts">
import { ref } from 'vue';
import { socket } from '../socket';

const emit = defineEmits(['close', 'created']);
const groupName = ref('');
const isSubmitting = ref(false);
const error = ref('');

const handleSubmit = () => {
  if (!groupName.value.trim()) return;
  
  isSubmitting.value = true;
  error.value = '';
  
  socket.emit('group:create', { name: groupName.value }, (res: any) => {
    isSubmitting.value = false;
    if (res.status === 'ok') {
      emit('created', res.group);
    } else {
      error.value = res.message || 'Failed to create group';
    }
  });
};
</script>

<template>
  <div class="modal-overlay" @click.self="$emit('close')">
    <div class="modal-content">
      <h2>Create New Group</h2>
      <form @submit.prevent="handleSubmit">
        <div class="form-group">
          <label for="groupName">Group Name</label>
          <input 
            id="groupName" 
            v-model="groupName" 
            type="text" 
            placeholder="e.g. Household, Trip to Japan" 
            required 
            :disabled="isSubmitting"
          />
        </div>
        <p v-if="error" class="error-msg">{{ error }}</p>
        <div class="actions">
          <button type="button" @click="$emit('close')" :disabled="isSubmitting" class="cancel-btn">Cancel</button>
          <button type="submit" :disabled="isSubmitting || !groupName.trim()" class="submit-btn">
            {{ isSubmitting ? 'Creating...' : 'Create Group' }}
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
  background: rgba(0, 0, 0, 0.5);
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
  max-width: 400px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.2);
}

h2 {
  margin-top: 0;
  margin-bottom: 1.5rem;
}

.form-group {
  margin-bottom: 1.5rem;
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
  font-size: 1rem;
}

.error-msg {
  color: #e74c3c;
  margin-bottom: 1rem;
  font-size: 0.9rem;
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

.submit-btn:disabled {
  background-color: #ccc;
  cursor: not-allowed;
}

.cancel-btn {
  background: none;
  border: 1px solid #ddd;
  padding: 0.75rem 1.5rem;
  border-radius: 4px;
  cursor: pointer;
}
</style>

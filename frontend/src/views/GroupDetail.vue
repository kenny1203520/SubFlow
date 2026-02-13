<script setup lang="ts">
import { ref, onMounted } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { socket } from '../socket';
import AddExpenseModal from '../components/AddExpenseModal.vue';

const route = useRoute();
const router = useRouter();
const groupId = route.params.id as string;

const group = ref<any>(null);
const members = ref<any[]>([]);
const expenses = ref<any[]>([]);
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
  <div class="group-detail" v-if="!loading && group">
    <header class="header">
      <div class="title-section">
        <button @click="router.push('/groups')" class="back-btn">← Back</button>
        <h1>{{ group.name }}</h1>
      </div>
      <div class="actions">
        <button @click="showAddMember = !showAddMember" class="add-member-btn">Add Member</button>
        <button @click="showAddExpense = true" class="add-expense-btn">Add Expense</button>
      </div>
    </header>

    <AddExpenseModal 
      v-if="showAddExpense" 
      :group-id="groupId" 
      :members="members" 
      @close="showAddExpense = false" 
      @added="handleExpenseAdded" 
    />

    <div v-if="showAddMember" class="add-member-form">
      <input v-model="newMemberEmail" type="email" placeholder="Member Email" />
      <button @click="addMember">Invite</button>
      <button @click="showAddMember = false">Cancel</button>
    </div>

    <div class="layout">
      <section class="expenses-section">
        <h2>Expenses</h2>
        <div v-if="expenses.length === 0" class="empty-state">
          No expenses recorded yet.
        </div>
        <div v-else class="expense-list">
          <div v-for="expense in expenses" :key="expense.id" class="expense-card">
            <div class="expense-info">
              <span class="desc">{{ expense.description }}</span>
              <span class="date">{{ new Date(expense.date).toLocaleDateString() }}</span>
            </div>
            <div class="expense-amount">
                ${{ expense.amount }}
            </div>
          </div>
        </div>
      </section>

      <aside class="members-sidebar">
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
      </aside>
    </div>
  </div>
  <div v-else-if="loading" class="state-container">Loading group details...</div>
  <div v-else class="state-container error">{{ error }}</div>
</template>

<style scoped>
.group-detail {
  padding: 2rem;
  max-width: 1200px;
  margin: 0 auto;
}

.header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 2rem;
  border-bottom: 1px solid #eee;
  padding-bottom: 1rem;
}

.title-section {
  display: flex;
  align-items: center;
  gap: 1rem;
}

.back-btn {
  background: none;
  border: 1px solid #ddd;
  padding: 0.5rem 1rem;
  border-radius: 4px;
  cursor: pointer;
}

.add-expense-btn {
  background-color: #4CAF50;
  color: white;
  border: none;
  padding: 0.75rem 1.5rem;
  border-radius: 4px;
  cursor: pointer;
  font-weight: bold;
}

.add-member-btn {
  background-color: #3498db;
  color: white;
  border: none;
  padding: 0.75rem 1.5rem;
  border-radius: 4px;
  cursor: pointer;
  margin-right: 0.5rem;
}

.layout {
  display: grid;
  grid-template-columns: 1fr 300px;
  gap: 2rem;
}

.add-member-form {
  margin-bottom: 1.5rem;
  padding: 1rem;
  background: #f8f9fa;
  border-radius: 8px;
  display: flex;
  gap: 0.5rem;
}

.add-member-form input {
  flex: 1;
  padding: 0.5rem;
  border: 1px solid #ddd;
  border-radius: 4px;
}

.add-member-form button {
  padding: 0.5rem 1rem;
  cursor: pointer;
}

.expenses-section h2, .members-sidebar h2 {
  margin-top: 0;
  font-size: 1.25rem;
  margin-bottom: 1rem;
}

.expense-card {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 1rem;
  background: white;
  border-radius: 8px;
  box-shadow: 0 1px 3px rgba(0,0,0,0.1);
  margin-bottom: 0.75rem;
}

.expense-info .desc {
  display: block;
  font-weight: bold;
}

.expense-info .date {
  font-size: 0.85rem;
  color: #666;
}

.expense-amount {
  font-weight: bold;
  font-size: 1.1rem;
}

.member-list {
  list-style: none;
  padding: 0;
}

.member-item {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.75rem;
  background: white;
  border-radius: 8px;
  margin-bottom: 0.5rem;
}

.member-avatar {
  width: 32px;
  height: 32px;
  background: #3498db;
  color: white;
  border-radius: 50%;
  display: flex;
  justify-content: center;
  align-items: center;
  font-weight: bold;
}

.member-details {
  display: flex;
  flex-direction: column;
}

.member-details .role {
  font-size: 0.75rem;
  color: #666;
  text-transform: capitalize;
}

.state-container {
  padding: 4rem;
  text-align: center;
  font-size: 1.2rem;
  color: #666;
}

.state-container.error {
  color: #e74c3c;
}
</style>

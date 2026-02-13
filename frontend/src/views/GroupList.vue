<script setup lang="ts">
import { ref, onMounted } from 'vue';
import { socket } from '../socket';
import CreateGroupModal from '../components/CreateGroupModal.vue';

const groups = ref<any[]>([]);
const showCreateModal = ref(false);

const fetchGroups = () => {
  socket.emit('group:list', (res: any) => {
    if (res.status === 'ok') {
      groups.value = res.groups;
    }
  });
};

onMounted(() => {
  if (socket.connected) {
    fetchGroups();
  } else {
    socket.once('connect', fetchGroups);
  }
});

const handleGroupCreated = () => {
  showCreateModal.value = false;
  fetchGroups();
};
</script>

<template>
  <div class="groups-view">
    <div class="header">
      <h1>My Groups</h1>
      <button @click="showCreateModal = true" class="create-btn">+ Create Group</button>
    </div>

    <div v-if="groups.length === 0" class="empty-state">
      <p>You are not in any groups yet.</p>
      <button @click="showCreateModal = true">Create your first group</button>
    </div>

    <div v-else class="group-grid">
      <div v-for="group in groups" :key="group.id" class="group-card" @click="$router.push(`/groups/${group.id}`)">
        <h3>{{ group.name }}</h3>
        <p>Created on: {{ new Date(group.created_at).toLocaleDateString() }}</p>
      </div>
    </div>

    <CreateGroupModal v-if="showCreateModal" @close="showCreateModal = false" @created="handleGroupCreated" />
  </div>
</template>

<style scoped>
.groups-view {
  padding: 2rem;
}

.header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 2rem;
}

.create-btn {
  background-color: #4CAF50;
  color: white;
  border: none;
  padding: 0.75rem 1.5rem;
  border-radius: 4px;
  cursor: pointer;
  font-weight: bold;
}

.group-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 1.5rem;
}

.group-card {
  background: white;
  padding: 1.5rem;
  border-radius: 8px;
  box-shadow: 0 2px 4px rgba(0,0,0,0.1);
  cursor: pointer;
  transition: transform 0.2s, box-shadow 0.2s;
}

.group-card:hover {
  transform: translateY(-4px);
  box-shadow: 0 4px 8px rgba(0,0,0,0.15);
}

.empty-state {
  text-align: center;
  margin-top: 4rem;
  color: #666;
}

.empty-state button {
  margin-top: 1rem;
  padding: 0.75rem 1.5rem;
  background-color: #3498db;
  color: white;
  border: none;
  border-radius: 4px;
  cursor: pointer;
}
</style>

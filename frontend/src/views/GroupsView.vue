<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useWorkspaceStore } from '../stores/workspace'
import type { Currency, Group } from '../api/types'

const workspace = useWorkspaceStore()
const editing = ref(false)
const form = reactive({ name: '', description: '', currency: 'TWD' as Currency, color: '#7357ff' })

function reset() {
  editing.value = false
  Object.assign(form, { name: '', description: '', currency: 'TWD' as Currency, color: '#7357ff' })
}

function edit(group: Group) {
  void workspace.selectGroup(group.id)
  editing.value = true
  Object.assign(form, { name: group.name, description: group.description, currency: group.currency, color: group.color })
}

async function submit() {
  if (editing.value) await workspace.updateGroup({ ...form })
  else await workspace.createGroup({ ...form })
  reset()
}

async function remove() {
  if (!workspace.currentGroup || !confirm(`確定刪除「${workspace.currentGroup.name}」？此操作無法復原。`)) return
  await workspace.deleteGroup()
  reset()
}
</script>

<template>
  <section class="page">
    <div class="page-heading"><div><p class="eyebrow">GROUPS</p><h1>群組空間</h1><p>不同家庭、旅程或團隊可以各自擁有獨立帳本。</p></div></div>
    <div class="two-column">
      <div class="card">
        <h2>你的群組</h2>
        <div v-if="workspace.groups.length" class="group-grid">
          <div v-for="group in workspace.groups" :key="group.id" class="group-tile-wrap">
            <button class="group-tile" :class="{ selected: group.id === workspace.currentGroupId }" @click="workspace.selectGroup(group.id)">
              <span class="group-color" :style="{ background: group.color }"></span>
              <strong>{{ group.name }}</strong>
              <small>{{ group.currency }} · {{ group.description || '尚無說明' }}</small>
            </button>
            <button v-if="group.id === workspace.currentGroupId && workspace.isOwner" class="ghost edit-link" @click="edit(group)">編輯</button>
          </div>
        </div>
        <p v-else class="empty-inline">尚未建立群組。</p>
      </div>
      <form class="card form-card" @submit.prevent="submit">
        <h2>{{ editing ? '編輯群組' : '建立新群組' }}</h2>
        <label>群組名稱<input v-model="form.name" required maxlength="120" placeholder="例如：我們家"></label>
        <label>簡短說明<textarea v-model="form.description" rows="3" placeholder="這個群組用來管理…"></textarea></label>
        <div class="form-row">
          <label>幣別<select v-model="form.currency"><option>TWD</option><option>USD</option><option>JPY</option><option>EUR</option></select></label>
          <label>代表色<input v-model="form.color" type="color"></label>
        </div>
        <div class="form-actions">
          <button class="primary" :disabled="workspace.loading">{{ editing ? '儲存變更' : '建立群組' }}</button>
          <button v-if="editing" type="button" class="ghost" @click="reset">取消</button>
          <button v-if="editing" type="button" class="ghost danger-text push-right" @click="remove">刪除群組</button>
        </div>
      </form>
    </div>
  </section>
</template>

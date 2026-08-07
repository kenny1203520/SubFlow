<script setup lang="ts">
import { ref } from 'vue'
import { useWorkspaceStore } from '../stores/workspace'
import EmptyState from '../components/EmptyState.vue'

const workspace = useWorkspaceStore()
const email = ref('')

async function invite() {
    await workspace.invite(email.value)
    email.value = ''
}

async function removeMember(userId: string, label: string) {
    if (confirm(`確定要移除「${label}」？`)) await workspace.removeMember(userId)
}
</script>

<template>
    <section class="page">
        <div class="page-heading">
            <div>
                <p class="eyebrow">PEOPLE</p>
                <h1>成員與邀請</h1>
                <p>邀請連結有效七天，只能由相同 Email 的帳號接受一次。</p>
            </div>
        </div>
        <div class="two-column">
            <div class="card">
                <h2>群組成員</h2>
                <div v-if="workspace.members.length" class="rows">
                    <div v-for="member in workspace.members" :key="member.id" class="row">
                        <div class="avatar">{{ (member.user?.name || member.user?.email || '?').slice(0,
                            1).toUpperCase() }}</div>
                        <div class="grow"><strong>{{ member.user?.name || '未命名成員' }}</strong><small>{{
                                member.user?.email }}</small></div>
                        <span class="pill">{{ member.role === 'owner' ? '擁有者' : '成員' }}</span>
                        <button v-if="workspace.isOwner && member.role !== 'owner'" class="ghost danger-text"
                            @click="removeMember(member.userId, member.user?.name || member.user?.email || '此成員')">移除</button>
                    </div>
                </div>
                <EmptyState v-else title="沒有成員資料" description="選擇群組後即可查看。" />
            </div>
            <div v-if="workspace.isOwner">
                <form class="card form-card" @submit.prevent="invite">
                    <h2>邀請新成員</h2>
                    <label>Email<input v-model="email" type="email" required placeholder="friend@example.com"></label>
                    <button class="primary" :disabled="workspace.loading">寄送邀請</button>
                </form>
                <div class="card invitations">
                    <h2>邀請紀錄</h2>
                    <div v-if="workspace.invitations.length">
                        <div v-for="item in workspace.invitations" :key="item.id" class="invite">
                            <div><strong>{{ item.email }}</strong><small>{{ item.status }} · {{ new
                                Date(item.expiresAt).toLocaleDateString('zh-TW') }}</small><a v-if="item.debugUrl"
                                    :href="item.debugUrl">開發測試連結</a></div>
                            <div v-if="item.status === 'pending' || item.status === 'delivery_failed'" class="actions">
                                <button class="ghost" @click="workspace.resendInvitation(item.id)">重送</button>
                                <button class="ghost danger-text"
                                    @click="workspace.revokeInvitation(item.id)">撤銷</button>
                            </div>
                        </div>
                    </div>
                    <p v-else class="empty-inline">尚無邀請紀錄。</p>
                </div>
            </div>
            <div v-else class="card owner-note">
                <h2>成員管理</h2>
                <p>只有群組擁有者可以邀請或移除成員。</p>
            </div>
        </div>
    </section>
</template>

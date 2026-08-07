<script setup lang="ts">
import { onMounted, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from './stores/auth'
import { useWorkspaceStore } from './stores/workspace'
import { useTheme } from './theme'
import LanguageSwitcher from './components/LanguageSwitcher.vue'
import ThemeSwitcher from './components/ThemeSwitcher.vue'

const auth = useAuthStore(), workspace = useWorkspaceStore(), router = useRouter(); useTheme()
async function hydrateWorkspace() { await workspace.loadGroups() }
onMounted(async () => { await auth.initialize(); if (auth.authenticated) await hydrateWorkspace() })
watch(() => auth.authenticated, authenticated => { if (authenticated) { void hydrateWorkspace(); return } workspace.clear(); if (auth.ready && router.currentRoute.value.path !== '/auth') void router.replace('/auth') })
</script>

<template>
    <div v-if="!auth.ready" class="splash">
        <div class="splash-mark">SF</div><strong>SubFlow</strong>
    </div>
    <RouterView v-else-if="!auth.authenticated" v-slot="{ Component, route }">
        <component :is="Component" :key="route.fullPath" />
    </RouterView>
    <div v-else class="shell">
        <aside class="sidebar">
            <RouterLink class="brand" to="/"><span>SF</span><strong>SubFlow</strong></RouterLink>
            <p class="sidebar-label">工作空間</p>
            <nav>
                <RouterLink to="/"><svg viewBox="0 0 24 24">
                        <path d="M4 13h6V4H4v9Zm0 7h6v-4H4v4Zm10 0h6v-9h-6v9Zm0-16v4h6V4h-6Z" />
                    </svg><span>總覽</span></RouterLink>
                <RouterLink to="/personal/expenses"><svg viewBox="0 0 24 24">
                        <path d="M4 5h16v14H4zM8 9h8M8 13h5" />
                    </svg><span>個人帳本</span></RouterLink>
                <RouterLink to="/groups"><svg viewBox="0 0 24 24">
                        <path d="M4 6.5 12 3l8 3.5-8 3-8-3Zm0 5 8 3 8-3M4 16.5l8 3 8-3" />
                    </svg><span>群組</span></RouterLink>
            </nav>
            <div class="sidebar-bottom">
                <RouterLink class="profile-link" to="/profile"><span
                        class="user-avatar">{{ auth.name.slice(0, 1).toUpperCase() }}</span><span><small>帳號設定</small>{{ auth.name }}</span>
                </RouterLink><button class="logout-button" @click="auth.logout">登出</button>
            </div>
        </aside>
        <main class="main">
            <header class="topbar">
                <div><small class="topbar-label">SubFlow</small><strong>個人與群組記帳</strong></div>
                <div class="topbar-actions"><span v-if="workspace.loading" class="sync"><i></i>同步中</span>
                    <ThemeSwitcher />
                    <LanguageSwitcher />
                </div>
            </header>
            <div v-if="workspace.permissionDenied" class="notice danger">你沒有權限查看目前群組。</div>
            <div v-else-if="workspace.error" class="notice">{{ workspace.error }} <button
                    @click="workspace.refreshGroup">重新整理</button></div>
            <RouterView v-slot="{ Component, route }">
                <component :is="Component" :key="route.fullPath" />
            </RouterView>
        </main>
    </div>
</template>

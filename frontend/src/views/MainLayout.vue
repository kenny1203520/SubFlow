<script setup lang="ts">
import { useRouter } from 'vue-router';
import { useAuthStore } from '../stores/auth';
import { useLayoutStore } from '../stores/layout';
import { useI18n } from 'vue-i18n';
import NotificationCenter from '../components/NotificationCenter.vue';

const router = useRouter();
const authStore = useAuthStore();
const layoutStore = useLayoutStore();
const { t, locale } = useI18n();

const setLanguage = (lang: string) => {
    locale.value = lang;
};

const logout = async () => {
    authStore.clearUser();
    router.push('/auth');
};
</script>

<template>
    <div class="app-layout">
        <!-- 背景裝飾 (Premium Feel) -->
        <div class="bg-decoration blur-1"></div>
        <div class="bg-decoration blur-2"></div>

        <!-- Sidebar Overlay (Mobile) -->
        <div v-if="layoutStore.isSidebarOpen" @click="layoutStore.closeSidebar" class="overlay lg:hidden"></div>

        <!-- Sidebar -->
        <aside
            :class="['sidebar', 'glass-panel', { 'open': layoutStore.isSidebarOpen, 'collapsed': layoutStore.isCollapsed }]">
            <div class="sidebar-header">
                <div class="logo-area">
                    <div class="logo-icon animate-pulse">S</div>
                    <span class="logo-text hide-on-collapsed">SubFlow</span>
                </div>

                <!-- Desktop Collapse Button -->
                <button @click="layoutStore.toggleCollapse" class="collapse-btn"
                    :title="layoutStore.isCollapsed ? t('common.actions.expand') : t('common.actions.collapse')">
                    <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24"
                        stroke="currentColor">
                        <path v-if="!layoutStore.isCollapsed" stroke-linecap="round" stroke-linejoin="round"
                            stroke-width="2" d="M11 19l-7-7 7-7m8 14l-7-7 7-7" />
                        <path v-else stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                            d="M13 5l7 7-7 7M5 5l7 7-7 7" />
                    </svg>
                </button>
            </div>

            <nav class="nav-links">
                <router-link to="/dashboard" class="nav-item" @click="layoutStore.closeSidebar">
                    <div class="icon-wrapper">
                        <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" fill="none" viewBox="0 0 24 24"
                            stroke="currentColor">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                                d="M4 6a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2H6a2 2 0 01-2-2V6zM14 6a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2V6zM4 16a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2H6a2 2 0 01-2-2v-2zM14 16a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2v-2z" />
                        </svg>
                    </div>
                    <span class="nav-text hide-on-collapsed">{{ t('dashboard.dashboard') }}</span>
                </router-link>
                <router-link to="/groups" class="nav-item" @click="layoutStore.closeSidebar">
                    <div class="icon-wrapper">
                        <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" fill="none" viewBox="0 0 24 24"
                            stroke="currentColor">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                                d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z" />
                        </svg>
                    </div>
                    <span class="nav-text hide-on-collapsed">{{ t('dashboard.groups') }}</span>
                </router-link>
                <router-link to="/subscriptions" class="nav-item" @click="layoutStore.closeSidebar">
                    <div class="icon-wrapper">
                        <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" fill="none" viewBox="0 0 24 24"
                            stroke="currentColor">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                                d="M3 10h18M7 15h1m4 0h1m-7 4h12a3 3 0 003-3V8a3 3 0 00-3-3H6a3 3 0 00-3 3v8a3 3 0 003 3z" />
                        </svg>
                    </div>
                    <span class="nav-text hide-on-collapsed">{{ t('dashboard.subscriptions') }}</span>
                </router-link>
            </nav>

            <div class="sidebar-footer">
                <div class="lang-switcher-container hide-on-collapsed">
                    <button class="lang-btn" @click="setLanguage('zh')">中文</button>
                    <button class="lang-btn" @click="setLanguage('en')">EN</button>
                </div>
                <div class="user-panel glass-card" v-if="authStore.user">
                    <img v-if="authStore.user.avatar_url" :src="authStore.user.avatar_url"
                        class="avatar-sm object-cover" />
                    <div v-else class="avatar-sm">{{ authStore.user.username[0].toUpperCase() }}</div>
                    <div class="user-info hide-on-collapsed">
                        <span class="username">{{ authStore.user.username }}</span>
                    </div>
                    <button @click="logout" class="logout-btn hide-on-collapsed">
                        <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24"
                            stroke="currentColor">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                                d="M17 16l4-4m0 0l-4-4m4 4H7m6 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h4a3 3 0 013 3v1" />
                        </svg>
                    </button>
                </div>
            </div>
        </aside>

        <!-- Main Content Area -->
        <div :class="['main-wrapper', { 'sidebar-collapsed': layoutStore.isCollapsed }]">
            <!-- Top Header -->
            <header class="top-header glass-panel">
                <div class="header-left">
                    <button @click="layoutStore.toggleSidebar" class="menu-btn lg:hidden">
                        <svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6" fill="none" viewBox="0 0 24 24"
                            stroke="currentColor">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                                d="M4 6h16M4 12h16M4 18h16" />
                        </svg>
                    </button>
                    <h2 class="page-title">{{ router.currentRoute.value.name || 'Dashboard' }}</h2>
                </div>
                <div class="header-right">
                    <NotificationCenter />
                    <div class="divider"></div>
                    <router-link to="/profile" class="profile-link">
                        <img v-if="authStore.user?.avatar_url" :src="authStore.user.avatar_url"
                            class="avatar-sm object-cover" />
                        <div v-else class="avatar-sm">{{ authStore.user?.username[0].toUpperCase() }}</div>
                    </router-link>
                </div>
            </header>

            <main class="content-view">
                <div class="animate-fade-in-up">
                    <slot></slot>
                </div>
            </main>
        </div>
    </div>
</template>

<style scoped>
.app-layout {
    display: flex;
    min-height: 100vh;
    position: relative;
    overflow: hidden;
}

/* Background Gradients */
.bg-decoration {
    position: fixed;
    border-radius: 50%;
    z-index: -1;
    filter: blur(80px);
    opacity: 0.15;
}

.blur-1 {
    width: 400px;
    height: 400px;
    background: var(--primary-color);
    top: -100px;
    left: -100px;
}

.blur-2 {
    width: 300px;
    height: 300px;
    background: #a855f7;
    bottom: -50px;
    right: -50px;
}

/* Sidebar Overhaul */
.sidebar {
    width: 280px;
    height: 100vh;
    display: flex;
    flex-direction: column;
    z-index: 50;
    transition: all var(--transition-normal);
    position: fixed;
    left: 0;
    top: 0;
    transform: translateX(-100%);
    background: rgba(255, 255, 255, 0.45) !important;
}

@media (min-width: 1024px) {
    .sidebar {
        transform: translateX(0);
    }

    .sidebar.collapsed {
        width: 88px;
    }
}

.sidebar.open {
    transform: translateX(0);
}

.sidebar-header {
    padding: 2rem 1.5rem;
    display: flex;
    align-items: center;
    justify-content: space-between;
}

.logo-area {
    display: flex;
    align-items: center;
    gap: 1rem;
}

.logo-icon {
    width: 40px;
    height: 40px;
    background: var(--primary-gradient);
    border-radius: 12px;
    display: flex;
    align-items: center;
    justify-content: center;
    color: white;
    font-weight: 800;
    font-size: 1.25rem;
    box-shadow: var(--shadow-md);
}

.logo-text {
    font-size: 1.5rem;
    font-weight: 700;
    background: var(--primary-gradient);
    -webkit-background-clip: text;
    -webkit-text-fill-color: transparent;
    letter-spacing: -1px;
}

/* Nav Items */
.nav-links {
    flex: 1;
    padding: 1rem;
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
}

.nav-item {
    display: flex;
    align-items: center;
    gap: 1rem;
    padding: 0.875rem 1rem;
    border-radius: var(--radius-md);
    color: var(--slate-600);
    text-decoration: none;
    transition: all var(--transition-fast);
}

.icon-wrapper {
    width: 36px;
    height: 36px;
    display: flex;
    align-items: center;
    justify-content: center;
    border-radius: 10px;
    transition: all var(--transition-fast);
}

.nav-item:hover {
    background: rgba(99, 102, 241, 0.05);
    color: var(--primary-600);
}

.nav-item.router-link-active {
    background: white;
    box-shadow: var(--shadow-sm);
    color: var(--primary-600);
}

.nav-item.router-link-active .icon-wrapper {
    background: var(--primary-gradient);
    color: white;
    box-shadow: 0 4px 10px rgba(99, 102, 241, 0.2);
}

/* Sidebar Footer */
.sidebar-footer {
    padding: 2rem 1rem;
}

.lang-switcher-container {
    display: flex;
    gap: 0.5rem;
    margin-bottom: 2rem;
}

.lang-btn {
    flex: 1;
    padding: 0.5rem;
    border-radius: 8px;
    border: 1px solid var(--border-color);
    background: white;
    font-size: 0.75rem;
    cursor: pointer;
}

.user-panel {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    padding: 0.75rem;
}

/* Main Wrapper & Top Header */
.main-wrapper {
    flex: 1;
    display: flex;
    flex-direction: column;
    transition: all var(--transition-normal);
}

@media (min-width: 1024px) {
    .main-wrapper {
        margin-left: 280px;
    }

    .main-wrapper.sidebar-collapsed {
        margin-left: 88px;
    }
}

.top-header {
    height: 80px;
    padding: 0 2rem;
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin: 1.5rem;
}

.header-right {
    display: flex;
    align-items: center;
    gap: 1.5rem;
}

.icon-btn {
    position: relative;
    background: none;
    border: none;
    color: var(--slate-500);
    cursor: pointer;
    padding: 0.5rem;
}

.badge {
    position: absolute;
    top: 6px;
    right: 6px;
    width: 8px;
    height: 8px;
    background: var(--danger-color);
    border-radius: 50%;
    border: 2px solid white;
}

.divider {
    width: 1px;
    height: 24px;
    background: var(--border-color);
}

.avatar-sm {
    width: 40px;
    height: 40px;
    border-radius: 12px;
    background: var(--slate-100);
    display: flex;
    align-items: center;
    justify-content: center;
    font-weight: 700;
}

.content-view {
    padding: 0 1.5rem 1.5rem;
    flex: 1;
}

.overlay {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.3);
    backdrop-filter: blur(4px);
    z-index: 40;
}

.hide-on-collapsed {
    transition: opacity 0.2s;
}

.collapsed .hide-on-collapsed {
    opacity: 0;
    pointer-events: none;
    display: none;
}

.collapse-btn {
    position: absolute;
    right: -12px;
    top: 30px;
    width: 24px;
    height: 24px;
    background: white;
    border: 1px solid var(--border-color);
    border-radius: 50%;
    display: flex;
    align-items: center;
    justify-content: center;
    cursor: pointer;
    box-shadow: var(--shadow-sm);
}

.logout-btn {
    background: none;
    border: none;
    color: var(--slate-400);
    cursor: pointer;
    margin-left: auto;
}
</style>

<script setup lang="ts">
import { ref } from 'vue';
import { useRouter } from 'vue-router';
import { useAuthStore } from '../stores/auth';
import { useI18n } from 'vue-i18n';

const router = useRouter();
const authStore = useAuthStore();
const isSidebarOpen = ref(false); // For mobile
const isCollapsed = ref(false); // For desktop
const { t, locale } = useI18n();

const toggleSidebar = () => {
    isSidebarOpen.value = !isSidebarOpen.value;
};

const toggleCollapse = () => {
    isCollapsed.value = !isCollapsed.value;
};

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
        <!-- Mobile Header -->
        <header class="mobile-header lg:hidden">
            <button @click="toggleSidebar" class="menu-btn">
                <svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6" fill="none" viewBox="0 0 24 24"
                    stroke="currentColor">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M4 12h16M4 18h16" />
                </svg>
            </button>
            <span class="logo-text-sm">SubFlow</span>
        </header>

        <!-- Sidebar Overlay (Mobile) -->
        <div v-if="isSidebarOpen" @click="isSidebarOpen = false" class="overlay lg:hidden"></div>

        <!-- Sidebar -->
        <aside :class="['sidebar', { 'open': isSidebarOpen, 'collapsed': isCollapsed }]">
            <div class="sidebar-header">
                <div class="logo-area" v-if="!isCollapsed">
                    <div class="logo-icon">S</div>
                    <span class="logo-text">SubFlow</span>
                </div>
                <div class="logo-area justify-center w-full" v-else>
                    <div class="logo-icon">S</div>
                </div>

                <!-- Desktop Collapse Button -->
                <button @click="toggleCollapse" class="collapse-btn hidden lg:flex"
                    :title="isCollapsed ? 'Expand' : 'Collapse'">
                    <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24"
                        stroke="currentColor">
                        <path v-if="!isCollapsed" stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                            d="M11 19l-7-7 7-7m8 14l-7-7 7-7" />
                        <path v-else stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                            d="M13 5l7 7-7 7M5 5l7 7-7 7" />
                    </svg>
                </button>

                <!-- Mobile Close Button -->
                <button @click="toggleSidebar" class="close-btn lg:hidden">
                    <svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6" fill="none" viewBox="0 0 24 24"
                        stroke="currentColor">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                            d="M6 18L18 6M6 6l12 12" />
                    </svg>
                </button>
            </div>

            <nav class="nav-links">
                <router-link to="/dashboard" class="nav-item" @click="isSidebarOpen = false"
                    :title="t('dashboard.dashboard')">
                    <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5 flex-shrink-0" fill="none"
                        viewBox="0 0 24 24" stroke="currentColor">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                            d="M4 6a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2H6a2 2 0 01-2-2V6zM14 6a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2V6zM4 16a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2H6a2 2 0 01-2-2v-2zM14 16a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2v-2z" />
                    </svg>
                    <span v-if="!isCollapsed" class="nav-text">{{ t('dashboard.dashboard') }}</span>
                </router-link>
                <router-link to="/groups" class="nav-item" @click="isSidebarOpen = false"
                    :title="t('dashboard.groups')">
                    <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5 flex-shrink-0" fill="none"
                        viewBox="0 0 24 24" stroke="currentColor">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                            d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z" />
                    </svg>
                    <span v-if="!isCollapsed" class="nav-text">{{ t('dashboard.groups') }}</span>
                </router-link>
                <router-link to="/subscriptions" class="nav-item" @click="isSidebarOpen = false"
                    :title="t('dashboard.subscriptions')">
                    <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5 flex-shrink-0" fill="none"
                        viewBox="0 0 24 24" stroke="currentColor">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                            d="M3 10h18M7 15h1m4 0h1m-7 4h12a3 3 0 003-3V8a3 3 0 00-3-3H6a3 3 0 00-3 3v8a3 3 0 003 3z" />
                    </svg>
                    <span v-if="!isCollapsed" class="nav-text">{{ t('dashboard.subscriptions') }}</span>
                </router-link>
            </nav>

            <div class="sidebar-footer">
                <!-- Language Switcher -->
                <div class="lang-switcher" :class="{ 'collapsed': isCollapsed }">
                    <button :class="['lang-opt', { active: locale === 'zh' }]" @click="setLanguage('zh')">
                        {{ isCollapsed ? '中' : '中文' }}
                    </button>
                    <button :class="['lang-opt', { active: locale === 'en' }]" @click="setLanguage('en')">
                        {{ isCollapsed ? 'EN' : 'EN' }}
                    </button>
                </div>

                <!-- User Profile -->
                <div class="user-profile" v-if="authStore.user" :class="{ 'collapsed': isCollapsed }">
                    <div class="user-info" v-if="!isCollapsed">
                        <div class="avatar">{{ authStore.user.username[0].toUpperCase() }}</div>
                        <div class="details">
                            <span class="username">{{ authStore.user.username }}</span>
                        </div>
                    </div>
                    <div class="avatar" v-else :title="authStore.user.username">{{
                        authStore.user.username[0].toUpperCase() }}</div>

                    <button @click="logout" class="logout-icon-btn" :title="t('common.actions.logout')">
                        <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" fill="none" viewBox="0 0 24 24"
                            stroke="currentColor">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                                d="M17 16l4-4m0 0l-4-4m4 4H7m6 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h4a3 3 0 013 3v1" />
                        </svg>
                    </button>
                </div>
            </div>
        </aside>

        <!-- Main Content -->
        <main :class="['main-content', { 'sidebar-collapsed': isCollapsed }]">
            <div class="container">
                <slot></slot>
            </div>
        </main>
    </div>
</template>

<style scoped>
.app-layout {
    display: flex;
    min-height: 100vh;
    background-color: var(--bg-body);
    flex-direction: column;
}

@media (min-width: 1024px) {
    .app-layout {
        flex-direction: row;
    }
}

/* Sidebar Styling */
.sidebar {
    position: fixed;
    top: 0;
    left: 0;
    bottom: 0;
    width: 260px;
    background-color: var(--bg-sidebar);
    color: var(--text-light);
    transform: translateX(-100%);
    transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
    z-index: 50;
    display: flex;
    flex-direction: column;
    box-shadow: 4px 0 24px rgba(0, 0, 0, 0.1);
}

.sidebar.open {
    transform: translateX(0);
}

@media (min-width: 1024px) {
    .sidebar {
        position: fixed;
        transform: none;
        box-shadow: none;
    }

    .sidebar.collapsed {
        width: 80px;
    }
}

.sidebar-header {
    height: 70px;
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 0 1.5rem;
    border-bottom: 1px solid rgba(255, 255, 255, 0.05);
    position: relative;
}

.collapse-btn {
    position: absolute;
    right: -12px;
    top: 24px;
    width: 24px;
    height: 24px;
    background-color: var(--primary-color);
    border-radius: 50%;
    color: white;
    border: 2px solid var(--bg-sidebar);
    display: flex;
    align-items: center;
    justify-content: center;
    cursor: pointer;
    z-index: 100;
}

.logo-area {
    display: flex;
    align-items: center;
    gap: 0.75rem;
}

.logo-icon {
    width: 32px;
    height: 32px;
    background: linear-gradient(135deg, var(--primary-500), var(--primary-700));
    border-radius: 8px;
    display: flex;
    align-items: center;
    justify-content: center;
    font-weight: 800;
    color: white;
    flex-shrink: 0;
}

.logo-text {
    font-size: 1.25rem;
    font-weight: 700;
    letter-spacing: -0.025em;
    color: white;
}

.nav-links {
    padding: 1.5rem 1rem;
    flex: 1;
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
}

.nav-item {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    padding: 0.75rem 1rem;
    color: var(--slate-400);
    border-radius: var(--radius-md);
    transition: all 0.2s;
    font-weight: 500;
    text-decoration: none;
    white-space: nowrap;
}

.sidebar.collapsed .nav-item {
    justify-content: center;
    padding: 0.75rem;
}

.nav-item:hover {
    background-color: rgba(255, 255, 255, 0.05);
    color: white;
}

.nav-item.router-link-active {
    background-color: var(--primary-600);
    color: white;
    box-shadow: 0 4px 12px rgba(99, 102, 241, 0.3);
}

/* Footer Section */
.sidebar-footer {
    padding: 1.5rem;
    background-color: rgba(0, 0, 0, 0.2);
    border-top: 1px solid rgba(255, 255, 255, 0.05);
}

/* Language Switcher */
.lang-switcher {
    display: flex;
    background-color: rgba(0, 0, 0, 0.3);
    padding: 0.25rem;
    border-radius: var(--radius-md);
    margin-bottom: 1.5rem;
    transition: all 0.3s;
}

.lang-switcher.collapsed {
    flex-direction: column;
    gap: 0.25rem;
}

.lang-opt {
    flex: 1;
    background: none;
    border: none;
    color: var(--slate-400);
    padding: 0.375rem;
    font-size: 0.8125rem;
    font-weight: 600;
    border-radius: var(--radius-sm);
    cursor: pointer;
    transition: all 0.2s;
}

.lang-opt.active {
    background-color: var(--primary-600);
    color: white;
    box-shadow: 0 1px 2px rgba(0, 0, 0, 0.2);
}

.lang-opt:hover:not(.active) {
    color: white;
}

/* User Profile */
.user-profile {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    padding-top: 0.5rem;
}

.user-profile.collapsed {
    flex-direction: column;
    gap: 1rem;
}

.user-info {
    flex: 1;
    display: flex;
    align-items: center;
    gap: 0.75rem;
    overflow: hidden;
}

.avatar {
    width: 36px;
    height: 36px;
    background-color: var(--slate-700);
    color: white;
    border-radius: 50%;
    display: flex;
    align-items: center;
    justify-content: center;
    font-weight: 600;
    font-size: 0.875rem;
    border: 2px solid var(--bg-sidebar);
    flex-shrink: 0;
}

.details {
    display: flex;
    flex-direction: column;
    overflow: hidden;
}

.username {
    font-size: 0.875rem;
    font-weight: 600;
    color: white;
    white-space: nowrap;
}

.logout-icon-btn {
    background: none;
    border: none;
    color: var(--slate-400);
    cursor: pointer;
    padding: 0.5rem;
    border-radius: var(--radius-md);
    transition: all 0.2s;
}

.logout-icon-btn:hover {
    background-color: rgba(239, 68, 68, 0.1);
    color: var(--danger-color);
}

/* Mobile Header */
.mobile-header {
    height: 60px;
    background-color: var(--bg-surface);
    border-bottom: 1px solid var(--border-color);
    padding: 0 1rem;
    display: flex;
    align-items: center;
    justify-content: space-between;
}

.logo-text-sm {
    font-weight: 700;
    font-size: 1.1rem;
    color: var(--text-main);
}

.menu-btn,
.close-btn {
    background: none;
    border: none;
    color: var(--text-main);
    padding: 0.5rem;
    cursor: pointer;
}

.close-btn {
    color: white;
}

/* Main Content */
.main-content {
    flex: 1;
    overflow-y: auto;
    padding: 2rem;
    transition: margin-left 0.3s cubic-bezier(0.4, 0, 0.2, 1);
}

@media (min-width: 1024px) {
    .main-content {
        margin-left: 260px;
        height: 100vh;
    }

    .main-content.sidebar-collapsed {
        margin-left: 80px;
    }
}

.overlay {
    position: fixed;
    inset: 0;
    background-color: rgba(0, 0, 0, 0.5);
    z-index: 40;
    backdrop-filter: blur(2px);
}
</style>

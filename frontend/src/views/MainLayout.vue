<script setup lang="ts">
import { computed } from 'vue';
import { useRouter, useRoute } from 'vue-router';
import http from '../http';
import { useAuthStore } from '../stores/auth';
import { useLayoutStore } from '../stores/layout';
import { useI18n } from 'vue-i18n';
import NotificationCenter from '../components/NotificationCenter.vue';

const router = useRouter();
const route = useRoute();
const authStore = useAuthStore();
const layoutStore = useLayoutStore();
const { t, locale } = useI18n();

const pageTitle = computed(() => {
    const titleKey = route.meta?.titleKey as string;
    return titleKey ? t(titleKey) : t('dashboard.dashboard');
});

const setLanguage = (lang: string) => {
    locale.value = lang;
};

const logout = async () => {
    try {
        await http.post('/auth/signout');
    } catch (err) {
        console.error("Signout failed", err);
    } finally {
        authStore.clearUser();
        router.push('/auth');
    }
};
</script>

<template>
    <div class="app-layout">
        <!-- Background Elements (Global, defined in style.css, but adding local extras) -->
        <div class="fixed top-0 left-0 w-full h-full overflow-hidden -z-10 pointer-events-none">
            <div class="blob blob-1"></div>
            <div class="blob blob-2 opacity-30"></div>
        </div>

        <!-- Sidebar Overlay (Mobile) -->
        <div v-if="layoutStore.isSidebarOpen" @click="layoutStore.closeSidebar" class="overlay lg:hidden"></div>

        <!-- Floating Sidebar -->
        <aside
            :class="['sidebar glass-panel', { 'open': layoutStore.isSidebarOpen, 'collapsed': layoutStore.isCollapsed }]">
            <div class="sidebar-header">
                <div class="logo-area">
                    <div class="w-10 h-10 flex items-center justify-center">
                        <img src="/favicon.svg" class="w-10 h-10 drop-shadow-md" alt="SubFlow" />
                    </div>
                    <span class="logo-text hide-on-collapsed">SubFlow</span>
                </div>

                <button @click="layoutStore.toggleCollapse" class="collapse-btn md:flex hidden"
                    :title="layoutStore.isCollapsed ? t('common.actions.expand') : t('common.actions.collapse')">
                    <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24"
                        stroke="currentColor">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7"
                            v-if="!layoutStore.isCollapsed" />
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" v-else />
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
                    <div class="active-indicator"></div>
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
                    <div class="active-indicator"></div>
                </router-link>

                <router-link to="/subscriptions" class="nav-item" @click="layoutStore.closeSidebar">
                    <div class="icon-wrapper">
                        <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" fill="none" viewBox="0 0 24 24"
                            stroke="currentColor">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                                d="M12 8c-1.657 0-3 .895-3 2s1.343 2 3 2 3 .895 3 2-1.343 2-3 2m0-8c1.11 0 2.08.402 2.599 1M12 8V7m0 1v8m0 0v1m0-1c-1.11 0-2.08-.402-2.599-1M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                        </svg>
                    </div>
                    <span class="nav-text hide-on-collapsed">{{ t('dashboard.subscriptions') }}</span>
                    <div class="active-indicator"></div>
                </router-link>

                <router-link to="/services" class="nav-item" @click="layoutStore.closeSidebar">
                    <div class="icon-wrapper">
                        <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" fill="none" viewBox="0 0 24 24"
                            stroke="currentColor">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                                d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" />
                        </svg>
                    </div>
                    <span class="nav-text hide-on-collapsed">{{ t('services.title', 'Services') }}</span>
                    <div class="active-indicator"></div>
                </router-link>

                <router-link to="/wallet" class="nav-item" @click="layoutStore.closeSidebar">
                    <div class="icon-wrapper">
                        <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" fill="none" viewBox="0 0 24 24"
                            stroke="currentColor">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                                d="M3 10h18M7 15h1m4 0h1m-7 4h12a3 3 0 003-3V8a3 3 0 00-3-3H6a3 3 0 00-3 3v8a3 3 0 003 3z" />
                        </svg>
                    </div>
                    <span class="nav-text hide-on-collapsed">{{ t('wallet.wallet', 'Wallet') }}</span>
                    <div class="active-indicator"></div>
                </router-link>

                <router-link to="/activity" class="nav-item" @click="layoutStore.closeSidebar">
                    <div class="icon-wrapper">
                        <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" fill="none" viewBox="0 0 24 24"
                            stroke="currentColor">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                                d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
                        </svg>
                    </div>
                    <span class="nav-text hide-on-collapsed">{{ t('activity.title', 'Activity') }}</span>
                    <div class="active-indicator"></div>
                </router-link>
                <router-link to="/security" class="nav-item" @click="layoutStore.closeSidebar">
                    <div class="icon-wrapper">
                        <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" fill="none" viewBox="0 0 24 24"
                            stroke="currentColor">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                                d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" />
                        </svg>
                    </div>
                    <span class="nav-text hide-on-collapsed">{{ t('security.title', 'Security') }}</span>
                    <div class="active-indicator"></div>
                </router-link>

                <router-link v-if="authStore.isAdmin" to="/admin" class="nav-item" @click="layoutStore.closeSidebar">
                    <div class="icon-wrapper">
                        <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" fill="none" viewBox="0 0 24 24"
                            stroke="currentColor">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                                d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                                d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
                        </svg>
                    </div>
                    <span class="nav-text hide-on-collapsed">{{ t('admin.title', 'Admin Dashboard') }}</span>
                    <div class="active-indicator"></div>
                </router-link>
            </nav>

            <div class="sidebar-footer">
                <div class="lang-switcher-container hide-on-collapsed mb-4">
                    <button class="lang-btn" :class="{ active: locale === 'zh' }" @click="setLanguage('zh')">中文</button>
                    <button class="lang-btn" :class="{ active: locale === 'en' }" @click="setLanguage('en')">EN</button>
                </div>

                <div class="user-panel glass-card p-3 flex items-center gap-3" v-if="authStore.user">
                    <img v-if="authStore.user.avatar_url" :src="authStore.user.avatar_url"
                        class="w-10 h-10 rounded-xl object-cover border border-white/50" />
                    <div v-else
                        class="w-10 h-10 rounded-xl bg-primary-100 text-primary-600 flex items-center justify-center font-bold text-lg border border-white/50">
                        {{ authStore.user?.username?.[0]?.toUpperCase() || '?' }}
                    </div>

                    <div class="flex-1 min-w-0 hide-on-collapsed" v-if="authStore.user">
                        <p class="text-sm font-bold truncate text-slate-800">{{ authStore.user.username }}</p>
                        <p class="text-xs text-slate-500 truncate">{{ authStore.systemRoles.join(', ') }}
                        </p>
                    </div>

                    <button @click="logout"
                        class="text-slate-400 hover:text-danger-color transition-colors hide-on-collapsed">
                        <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" fill="none" viewBox="0 0 24 24"
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
            <!-- Header -->
            <header class="top-header">
                <div class="flex items-center gap-4">
                    <button @click="layoutStore.toggleSidebar"
                        class="lg:hidden p-2 text-slate-600 hover:bg-white/50 rounded-lg transition-colors">
                        <svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6" fill="none" viewBox="0 0 24 24"
                            stroke="currentColor">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                                d="M4 6h16M4 12h16M4 18h16" />
                        </svg>
                    </button>
                    <h2 class="text-2xl font-bold text-slate-800 tracking-tight">{{ pageTitle }}</h2>
                </div>

                <div class="flex items-center gap-4">
                    <div class="hidden md:flex relative">
                        <input type="text" placeholder="Search..."
                            class="glass-input pl-10 py-2 h-10 w-64 text-sm rounded-full focus:w-72 transition-all" />
                        <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4 absolute left-3 top-3 text-slate-400"
                            fill="none" viewBox="0 0 24 24" stroke="currentColor">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                                d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
                        </svg>
                    </div>

                    <NotificationCenter />

                    <router-link to="/profile" class="profile-link hover:scale-105 transition-transform">
                        <img v-if="authStore.user?.avatar_url" :src="authStore.user.avatar_url"
                            class="w-10 h-10 rounded-full object-cover border-2 border-white shadow-sm" />
                        <div v-else
                            class="w-10 h-10 rounded-full bg-slate-200 flex items-center justify-center text-slate-600 font-bold border-2 border-white shadow-sm">
                            {{ authStore.user?.username?.[0]?.toUpperCase() || '?' }}
                        </div>
                    </router-link>
                </div>
            </header>

            <main class="content-view">
                <div class="animate-fade-in-up h-full">
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
    background: var(--bg-body);
}

/* Sidebar Floating Style
    */
.sidebar {
    position: fixed;
    top: 1rem;
    left: 1rem;
    bottom: 1rem;
    width: 260px;
    z-index: 50;
    display: flex;
    flex-direction: column;
    padding: 1.5rem;
    transition: all var(--transition-bounce);
    /* Glass Panel styles inherited
    from global .glass-panel */
    border: 1px solid rgba(255, 255, 255, 0.7);
    box-shadow: 0 8px 32px rgba(31, 38, 135,
            0.05);
}

.sidebar.collapsed {
    width: 80px;
    padding: 1.5rem 0.75rem;
}

.sidebar.collapsed .logo-area span,
.sidebar.collapsed .hide-on-collapsed {
    opacity: 0;
    pointer-events: none;
    display: none;
    /* Crucial for layout */
}

.sidebar.collapsed .collapse-btn {
    display: flex;
    /* Force display */
}

/* On Mobile, sidebar slides out completely */
@media (max-width: 1024px) {
    .sidebar {
        transform:
            translateX(calc(-100% - 2rem));
        bottom: 0;
        top: 0;
        left: 0;
        margin: 0;
        border-radius: 0;
        height: 100vh;
    }

    .sidebar.open {
        transform: translateX(0);
    }
}

.sidebar-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 2rem;
    padding: 0 0.5rem;
    min-height: 40px;
    /* Ensure height */
}

/* When collapsed, center the logo and button stack or adjust */
.sidebar.collapsed .sidebar-header {
    flex-direction: column;
    gap: 1rem;
    padding: 0;
    justify-content: center;
}

.sidebar.collapsed .collapse-btn {
    margin: 0 auto;
}

.logo-area {
    display: flex;
    align-items: center;
    gap: 1rem;
}

.logo-text {
    font-size: 1.25rem;
    font-weight: 700;
    background: var(--primary-gradient);
    -webkit-background-clip:
        text;
    background-clip: text;
    -webkit-text-fill-color: transparent;
}

/* Nav Links */
.nav-links {
    flex: 1;
    display:
        flex;
    flex-direction: column;
    gap: 0.5rem;
}

.nav-item {
    position: relative;
    display: flex;
    align-items: center;
    gap: 1rem;
    padding: 0.8rem;
    border-radius: 12px;
    color: var(--slate-500);
    transition: all 0.2s;
    font-weight: 500;
}

.nav-item:hover {
    color: var(--primary-600);
    background: rgba(255, 255, 255, 0.5);
}

.nav-item.router-link-active {
    background: white;
    color: var(--primary-600);
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.03);
}

.nav-item.router-link-active .icon-wrapper {
    color: var(--primary-600);
}

.icon-wrapper {
    width: 24px;
    height: 24px;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: all 0.2s;
}

/* Collapse Button */
.collapse-btn {
    width: 24px;
    height: 24px;
    border-radius: 50%;
    background: white;
    border: 1px solid var(--slate-200);
    color: var(--slate-400);
    display: flex;
    align-items: center;
    justify-content: center;
    cursor:
        pointer;
    box-shadow: 0 2px 5px rgba(0, 0, 0, 0.05);
}

.collapse-btn:hover {
    color: var(--primary-600);
    border-color:
        var(--primary-200);
}

/* Main Wrapper - Replaced with Tailwind-friendly padding logic */
.main-wrapper {
    margin-left: 260px;
    flex: 1;
    min-height: 100vh;
    transition: all var(--transition-bounce);
    display: flex;
    flex-direction: column;
    width: 100%;
    /* Ensure width */
}

.main-wrapper.sidebar-collapsed {
    margin-left: 80px;
}

@media (max-width: 1024px) {
    .main-wrapper {
        margin-left: 0;
    }
}

.top-header {
    height: var(--header-height);
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 0 2rem;
    margin-bottom: 0;
}

.content-view {
    padding: 0 2rem 2rem 2rem;
    flex: 1;
    width: 100%;
    /* Ensure full width */
    max-width: 1600px;
    margin: 0 auto;
}

/* Lang Switcher */
.lang-switcher-container {
    display: flex;
    background: rgba(0, 0, 0, 0.03);
    padding: 4px;
    border-radius: 10px;
}

.lang-btn {
    flex: 1;
    padding: 6px;
    border-radius: 8px;
    font-size: 0.75rem;
    font-weight: 600;
    color:
        var(--slate-500);
    background: transparent;
    border: none;
    cursor: pointer;
    transition: all 0.2s;
}

.lang-btn.active {
    background: white;
    color: var(--primary-600);
    box-shadow: 0 2px 4px rgba(0, 0, 0, 0.05);
}

.overlay {
    position:
        fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.2);
    backdrop-filter: blur(2px);
    z-index: 40;
}
</style>
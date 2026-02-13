<script setup lang="ts">
import { ref } from 'vue';
import { useRouter } from 'vue-router';
import { useAuthStore } from '../stores/auth';

const router = useRouter();
const authStore = useAuthStore();
const isSidebarOpen = ref(false);

const toggleSidebar = () => {
    isSidebarOpen.value = !isSidebarOpen.value;
};

const logout = async () => {
    // Logic to be passed from parent or store
    authStore.clearUser();
    router.push('/auth');
};
</script>

<template>
    <div class="app-layout">
        <!-- Mobile Header -->
        <header class="mobile-header lg:hidden">
            <button @click="toggleSidebar" class="menu-btn">
                ☰
            </button>
            <span class="logo">SubFlow</span>
        </header>

        <!-- Sidebar / Navigation -->
        <aside :class="['sidebar', { 'open': isSidebarOpen }]">
            <div class="sidebar-header">
                <h2>SubFlow</h2>
                <button @click="toggleSidebar" class="close-btn lg:hidden">×</button>
            </div>

            <nav class="nav-links">
                <router-link to="/dashboard" class="nav-item" @click="isSidebarOpen = false">
                    Dashboard
                </router-link>
                <router-link to="/groups" class="nav-item" @click="isSidebarOpen = false">
                    Groups
                </router-link>
                <router-link to="/subscriptions" class="nav-item" @click="isSidebarOpen = false">
                    Subscriptions
                </router-link>
            </nav>

            <div class="user-info" v-if="authStore.user">
                <div class="user-details">
                    <div class="avatar">{{ authStore.user.username[0].toUpperCase() }}</div>
                    <div class="info">
                        <span class="username">{{ authStore.user.username }}</span>
                        <span class="email">{{ authStore.user.email }}</span>
                    </div>
                </div>
                <button @click="logout" class="logout-btn">Logout</button>
            </div>
        </aside>

        <!-- Main Content -->
        <main class="main-content">
            <div class="container">
                <slot></slot>
            </div>
        </main>

        <!-- Overlay for mobile sidebar -->
        <div v-if="isSidebarOpen" @click="isSidebarOpen = false" class="overlay lg:hidden"></div>
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

/* Mobile Header */
.mobile-header {
    background-color: var(--bg-sidebar);
    color: white;
    padding: 1rem;
    display: flex;
    align-items: center;
    gap: 1rem;
}

.menu-btn {
    color: white;
    font-size: 1.5rem;
}

.logo {
    font-weight: bold;
    font-size: 1.25rem;
}

/* Sidebar */
.sidebar {
    position: fixed;
    top: 0;
    left: 0;
    bottom: 0;
    width: 280px;
    background-color: var(--bg-sidebar);
    color: var(--text-light);
    transform: translateX(-100%);
    transition: transform 0.3s ease;
    z-index: 50;
    display: flex;
    flex-direction: column;
}

.sidebar.open {
    transform: translateX(0);
}

@media (min-width: 1024px) {
    .sidebar {
        position: static;
        transform: none;
        flex-shrink: 0;
    }
}

.sidebar-header {
    padding: 2rem 1.5rem;
    display: flex;
    justify-content: space-between;
    align-items: center;
}

.sidebar-header h2 {
    margin: 0;
    font-size: 1.5rem;
    color: var(--primary-color);
}

.close-btn {
    color: white;
    font-size: 1.5rem;
}

.nav-links {
    padding: 0 1rem;
    flex: 1;
}

.nav-item {
    display: block;
    padding: 0.75rem 1rem;
    margin-bottom: 0.5rem;
    color: #9ca3af;
    border-radius: var(--radius-md);
    transition: all 0.2s;
}

.nav-item:hover,
.nav-item.router-link-active {
    background-color: rgba(255, 255, 255, 0.1);
    color: white;
}

.user-info {
    padding: 1.5rem;
    background-color: rgba(0, 0, 0, 0.2);
    border-top: 1px solid rgba(255, 255, 255, 0.1);
}

.user-details {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    margin-bottom: 1rem;
}

.avatar {
    width: 40px;
    height: 40px;
    background-color: var(--primary-color);
    border-radius: 50%;
    display: flex;
    align-items: center;
    justify-content: center;
    font-weight: bold;
    color: white;
}

.info {
    display: flex;
    flex-direction: column;
    overflow: hidden;
}

.username {
    font-weight: 600;
    color: white;
}

.email {
    font-size: 0.75rem;
    color: #9ca3af;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
}

.logout-btn {
    width: 100%;
    padding: 0.5rem;
    border: 1px solid #ef4444;
    color: #ef4444;
    border-radius: var(--radius-md);
    font-size: 0.875rem;
    transition: all 0.2s;
}

.logout-btn:hover {
    background-color: #ef4444;
    color: white;
}

/* Main Content */
.main-content {
    flex: 1;
    padding: 2rem 1rem;
    overflow-y: auto;
}

@media (min-width: 640px) {
    .main-content {
        padding: 3rem 2rem;
    }
}

/* Overlay */
.overlay {
    position: fixed;
    inset: 0;
    background-color: rgba(0, 0, 0, 0.5);
    z-index: 40;
    backdrop-filter: blur(2px);
}

/* Utilities */
.lg\:hidden {
    display: flex;
}

@media (min-width: 1024px) {
    .lg\:hidden {
        display: none !important;
    }
}
</style>

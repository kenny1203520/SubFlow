<script setup lang="ts">
import { useRoute, useRouter } from 'vue-router';
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';

const router = useRouter();
const route = useRoute();
const { t, locale } = useI18n();

const setLanguage = (lang: string) => {
    locale.value = lang;
};

const navigation = [
    { key: 'admin.adminDashboard', href: '/admin', current: computed(() => route.path === '/admin') },
    { key: 'admin.userManagement', href: '/admin/users', current: computed(() => route.path.startsWith('/admin/users')) },
    { key: 'admin.systemSettings', href: '/admin/settings', current: computed(() => route.path.startsWith('/admin/settings')) },
    { key: 'admin.ipBlocks', href: '/admin/ip-blocks', current: computed(() => route.path.startsWith('/admin/ip-blocks')) },
    { key: 'admin.rolesManagement', href: '/admin/roles', current: computed(() => route.path.startsWith('/admin/roles')) }
];
</script>

<template>
    <div class="min-h-screen bg-neutral-900 text-neutral-100 flex flex-col md:flex-row">

        <!-- Sidebar Navigation -->
        <nav class="w-full md:w-64 bg-neutral-950/80 border-r border-neutral-800 p-6 flex flex-col shrink-0">
            <div class="mb-8">
                <h1 class="text-xl font-bold bg-gradient-to-r from-red-500 to-rose-500 bg-clip-text text-transparent">
                    {{ t('admin.adminPanel') }}
                </h1>
                <p class="text-sm text-neutral-400 mt-1">{{ t('admin.systemControl') }}</p>
            </div>

            <div class="flex-1 space-y-2">
                <router-link v-for="item in navigation" :key="item.key" :to="item.href" :class="[
                    item.current.value ? 'bg-neutral-800/80 text-white' : 'text-neutral-400 hover:bg-neutral-800/50 hover:text-white',
                    'block px-4 py-2.5 rounded-xl font-medium transition-all duration-200'
                ]">
                    {{ t(item.key) }}
                </router-link>
            </div>

            <div class="mt-auto pt-6 border-t border-neutral-800">
                <button @click="router.push('/dashboard')"
                    class="w-full flex items-center px-4 py-2.5 text-neutral-400 hover:bg-neutral-800/50 hover:text-white rounded-xl transition-all duration-200 mb-2">
                    <svg class="w-5 h-5 mr-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                            d="M10 19l-7-7m0 0l7-7m-7 7h18"></path>
                    </svg>
                    {{ t('admin.exitAdmin') }}
                </button>

                <!-- Language Switcher -->
                <div class="lang-switcher">
                    <button class="lang-btn" :class="{ active: locale === 'zh' }" @click="setLanguage('zh')">中文</button>
                    <button class="lang-btn" :class="{ active: locale === 'en' }" @click="setLanguage('en')">EN</button>
                </div>
            </div>
        </nav>

        <!-- Main Content -->
        <main class="flex-1 min-w-0 overflow-y-auto relative h-screen">
            <div class="absolute inset-0 bg-neutral-900 overflow-y-auto">
                <div class="max-w-7xl mx-auto p-4 sm:p-6 lg:p-8 w-full min-h-full">
                    <router-view></router-view>
                </div>
            </div>
        </main>

    </div>
</template>

<style scoped>
.lang-switcher {
    display: flex;
    background: rgba(255, 255, 255, 0.05);
    padding: 4px;
    border-radius: 10px;
    margin-top: 1rem;
}

.lang-btn {
    flex: 1;
    padding: 6px;
    border-radius: 8px;
    font-size: 0.75rem;
    font-weight: 600;
    color: #a3a3a3;
    background: transparent;
    border: none;
    cursor: pointer;
    transition: all 0.2s;
}

.lang-btn:hover {
    color: #ffffff;
}

.lang-btn.active {
    background: rgba(255, 255, 255, 0.1);
    color: #ffffff;
    box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
}
</style>

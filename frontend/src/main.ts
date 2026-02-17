import { createApp } from 'vue'
import './style.css'
import App from './App.vue'
import { createPinia } from 'pinia'
import { createRouter, createWebHistory } from 'vue-router'
import i18n from './i18n'

import Auth from './views/Auth.vue'
import Dashboard from './views/Dashboard.vue'
import GroupList from './views/GroupList.vue'
import GroupDetail from './views/GroupDetail.vue'
import SubscriptionList from './views/SubscriptionList.vue'
import WalletView from './views/WalletView.vue'
import SecurityView from './views/SecurityView.vue'
import ProfileView from './views/ProfileView.vue'
import AdminDashboard from './views/AdminDashboard.vue'

const routes = [
    { path: '/', redirect: '/dashboard' },
    { path: '/auth', component: Auth },
    {
        path: '/dashboard',
        component: Dashboard,
        meta: { requiresAuth: true }
    },
    { path: '/groups', component: GroupList, meta: { requiresAuth: true } },
    { path: '/groups/:id', component: GroupDetail, meta: { requiresAuth: true } },
    { path: '/subscriptions', component: SubscriptionList, meta: { requiresAuth: true } },
    { path: '/wallet', component: WalletView, meta: { requiresAuth: true } },
    { path: '/security', component: SecurityView, meta: { requiresAuth: true } },
    { path: '/profile', component: ProfileView, meta: { requiresAuth: true } },
    { path: '/admin', component: AdminDashboard, meta: { requiresAuth: true, requiresAdmin: true } },
]

const router = createRouter({
    history: createWebHistory(),
    routes,
})

// Simple Route Guard
router.beforeEach((_to, _from, next) => {
    next();
});

const pinia = createPinia()

const app = createApp(App)
app.use(pinia)
app.use(router)
app.use(i18n)
app.mount('#app')

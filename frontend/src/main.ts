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
import ForgotPassword from './views/ForgotPassword.vue'
import ResetPassword from './views/ResetPassword.vue'
import VerifyEmail from './views/VerifyEmail.vue'
import ActivityLog from './views/ActivityLog.vue'

const routes = [
    { path: '/', redirect: '/dashboard' },
    { path: '/auth', component: Auth },
    { path: '/auth/forgot-password', component: ForgotPassword },
    { path: '/auth/reset-password/:token', component: ResetPassword },
    { path: '/auth/verify-email/:token', component: VerifyEmail },
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
    { path: '/activity', component: ActivityLog, meta: { requiresAuth: true } },
    { path: '/admin', component: AdminDashboard, meta: { requiresAuth: true, requiresAdmin: true } },
]

const router = createRouter({
    history: createWebHistory(),
    routes,
})

const pinia = createPinia()
const app = createApp(App)
app.use(pinia)

import { useAuthStore } from './stores/auth'
import http from './http'

const authStore = useAuthStore()

// Initialize auth
const initApp = async () => {
    try {
        const res = await http.get('/auth/user');
        if (res.data) {
            authStore.setUser(res.data);
        }
    } catch (err) {
        console.log('Not authenticated');
    } finally {
        app.use(router)
        app.use(i18n)
        app.mount('#app')
    }
}

// Better Route Guard
router.beforeEach(async (to, from, next) => {
    if (to.meta.requiresAuth && !authStore.user) {
        // Try one last time to check auth if user is null
        try {
            const res = await http.get('/auth/user');
            if (res.data) {
                authStore.setUser(res.data);
                return next();
            }
        } catch (e) { }
        return next('/auth');
    }
    next();
});

initApp();

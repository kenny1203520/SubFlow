import { createApp } from 'vue'
import './assets/tailwind.css'
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
import AdminLayout from './views/admin/AdminLayout.vue'
import AdminDashboard from './views/admin/AdminDashboard.vue'
import UserManagement from './views/admin/UserManagement.vue'
import SystemSettings from './views/admin/SystemSettings.vue'
import IpBlocks from './views/admin/IpBlocks.vue'
import RolesManagement from './views/admin/RolesManagement.vue'
import ActivityLogs from './views/admin/ActivityLogs.vue'
import ForgotPassword from './views/ForgotPassword.vue'
import ResetPassword from './views/ResetPassword.vue'
import VerifyEmail from './views/VerifyEmail.vue'
import ActivityLog from './views/ActivityLog.vue'
import ServiceManagementView from './views/ServiceManagementView.vue'

const routes = [
    { path: '/', redirect: '/dashboard' },
    { path: '/auth', component: Auth },
    { path: '/auth/forgot-password', component: ForgotPassword },
    { path: '/auth/reset-password/:token', component: ResetPassword },
    { path: '/auth/verify-email/:token', component: VerifyEmail },
    {
        path: '/dashboard',
        component: Dashboard,
        meta: { requiresAuth: true, titleKey: 'dashboard.dashboard' }
    },
    { path: '/groups', component: GroupList, meta: { requiresAuth: true, titleKey: 'dashboard.groups' } },
    { path: '/groups/:id', component: GroupDetail, meta: { requiresAuth: true, titleKey: 'groups.groupDetail' } },
    { path: '/subscriptions', component: SubscriptionList, meta: { requiresAuth: true, titleKey: 'dashboard.subscriptions' } },
    { path: '/wallet', component: WalletView, meta: { requiresAuth: true, titleKey: 'wallet.wallet' } },
    { path: '/security', component: SecurityView, meta: { requiresAuth: true, titleKey: 'security.title' } },
    { path: '/profile', component: ProfileView, meta: { requiresAuth: true, titleKey: 'profile.profile' } },
    { path: '/activity', component: ActivityLog, meta: { requiresAuth: true, titleKey: 'activity.title' } },
    { path: '/services', component: ServiceManagementView, meta: { requiresAuth: true, titleKey: 'services.title' } },
    { 
        path: '/admin', 
        component: AdminLayout, 
        meta: { requiresAuth: true, requiresAdmin: true, titleKey: 'admin.title' },
        children: [
            { path: '', component: AdminDashboard },
            { path: 'users', component: UserManagement, meta: { titleKey: 'admin.userManagement' } },
            { path: 'settings', component: SystemSettings, meta: { titleKey: 'admin.systemSettings' } },
            { path: 'ip-blocks', component: IpBlocks, meta: { titleKey: 'admin.ipBlocks' } },
            { path: 'roles', component: RolesManagement, meta: { titleKey: 'admin.rolesManagement' } },
            { path: 'logs', component: ActivityLogs, meta: { titleKey: 'admin.logs.title' } }
        ]
    },
]

const router = createRouter({
    history: createWebHistory(),
    routes,
})

router.afterEach((to) => {
    const titleKey = to.meta.titleKey as string | undefined;
    if (titleKey) {
        document.title = `Subflow | ${i18n.global.t(titleKey)}`;
    } else {
        document.title = 'Subflow';
    }
});

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
        // console.log('Not authenticated');
    } finally {
        app.use(router)
        app.use(i18n)
        app.mount('#app')
    }
}

// Better Route Guard
router.beforeEach(async (to, _from, next) => {
    // Check Auth
    if (to.meta.requiresAuth && !authStore.user) {
        // Try one last time to check auth if user is null
        try {
            const res = await http.get('/auth/user');
            if (res.data) {
                authStore.setUser(res.data);
            } else {
                return next('/auth');
            }
        } catch (e) { 
            return next('/auth');
        }
    }

    // Check Admin
    if (to.meta.requiresAdmin && !authStore.hasPermission('system', 'read', 'admin')) {
        return next('/dashboard'); // Unauthorized admin access, return to dashboard
    }

    next();
});

initApp();

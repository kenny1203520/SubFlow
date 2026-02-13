import { createApp } from 'vue'
import './style.css'
import App from './App.vue'
import { createPinia } from 'pinia'
import { createRouter, createWebHistory } from 'vue-router'

import Auth from './views/Auth.vue'
import Dashboard from './views/Dashboard.vue'

const routes = [
    { path: '/', redirect: '/dashboard' },
    { path: '/auth', component: Auth },
    {
        path: '/dashboard',
        component: Dashboard,
        meta: { requiresAuth: true }
    },
    { path: '/groups', component: { template: '<div>Groups Placeholder</div>' } },
    { path: '/subscriptions', component: { template: '<div>Subscriptions Placeholder</div>' } },
]

const router = createRouter({
    history: createWebHistory(),
    routes,
})

// Simple Route Guard
router.beforeEach((to, from, next) => {
    next();
});

const pinia = createPinia()

const app = createApp(App)
app.use(pinia)
app.use(router)
app.mount('#app')

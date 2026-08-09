import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'
import { pb } from './pocketbase'
import AuthView from './views/AuthView.vue'
import DashboardView from './views/DashboardView.vue'
import GroupsView from './views/GroupsView.vue'
import MembersView from './views/MembersView.vue'
import GroupAuditView from './views/GroupAuditView.vue'
import SubscriptionsView from './views/SubscriptionsView.vue'
import ExpensesView from './views/ExpensesView.vue'
import ProfileView from './views/ProfileView.vue'
import InviteView from './views/InviteView.vue'
import GroupWorkspaceView from './views/GroupWorkspaceView.vue'
import AdminView from './views/AdminView.vue'
import SetupView from './views/SetupView.vue'

export const routes: RouteRecordRaw[] = [
  { path: '/setup', name: 'setup', component: SetupView, meta: { public: true } },
  { path: '/auth', name: 'auth', component: AuthView, meta: { public: true } },
  { path: '/invite/:token', name: 'invite', component: InviteView },
  { path: '/', name: 'dashboard', component: DashboardView },
  { path: '/groups', name: 'groups', component: GroupsView },
  {
    path: '/groups/:groupId', component: GroupWorkspaceView,
    children: [
      { path: '', redirect: { name: 'group-overview' } },
      { path: 'overview', name: 'group-overview', component: DashboardView },
      { path: 'expenses', name: 'group-expenses', component: ExpensesView },
      { path: 'subscriptions', name: 'group-subscriptions', component: SubscriptionsView },
      { path: 'members', name: 'group-members', component: MembersView },
      { path: 'audit', name: 'group-audit', component: GroupAuditView },
      { path: 'settings', name: 'group-settings', component: GroupsView },
    ],
  },
  { path: '/personal/expenses', name: 'personal-expenses', component: ExpensesView },
  { path: '/personal/subscriptions', name: 'personal-subscriptions', component: SubscriptionsView },
  { path: '/members', redirect: { name: 'groups' } },
  { path: '/subscriptions', redirect: { name: 'personal-subscriptions' } },
  { path: '/expenses', redirect: { name: 'personal-expenses' } },
  { path: '/profile', name: 'profile', component: ProfileView },
  { path: '/admin', name: 'admin', component: AdminView },
  { path: '/admin/:section(settings|users|roles|audit)', name: 'admin-section', component: AdminView },
  { path: '/:pathMatch(.*)*', redirect: { name: 'dashboard' } },
]

export const router = createRouter({ history: createWebHistory(), routes })
router.beforeEach(async to => {
	if (to.name === 'setup') return true
  if (!to.meta.public && !pb.authStore.isValid) return { name: 'auth', query: { redirect: to.fullPath } }
  if (to.name === 'auth' && pb.authStore.isValid) return { name: 'dashboard' }
  if (String(to.path).startsWith('/admin')) {
    try { const response=await fetch('/api/subflow/v1/system/access',{headers:{Authorization:`Bearer ${pb.authStore.token}`}}); const body=await response.json(); const permissions:string[]=body?.data?.permissions||[]; if (!permissions.some(value=>value==='*'||value.startsWith('system.'))) return {name:'dashboard',query:{denied:'admin'}} } catch { return {name:'dashboard'} }
  }
  return true
})

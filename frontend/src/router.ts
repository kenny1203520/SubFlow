import { createRouter,createWebHistory } from 'vue-router'
import { pb } from './pocketbase'
import AuthView from './views/AuthView.vue'
import DashboardView from './views/DashboardView.vue'
import GroupsView from './views/GroupsView.vue'
import MembersView from './views/MembersView.vue'
import SubscriptionsView from './views/SubscriptionsView.vue'
import ExpensesView from './views/ExpensesView.vue'
import ProfileView from './views/ProfileView.vue'
import InviteView from './views/InviteView.vue'
import GroupWorkspaceView from './views/GroupWorkspaceView.vue'

export const router=createRouter({history:createWebHistory(),routes:[{path:'/auth',component:AuthView,meta:{public:true}},{path:'/invite/:token',component:InviteView},{path:'/',component:DashboardView},{path:'/groups',component:GroupsView},{path:'/groups/:groupId',component:GroupWorkspaceView,children:[{path:'',redirect:to=>`/groups/${to.params.groupId}/overview`},{path:'overview',component:DashboardView},{path:'expenses',component:ExpensesView},{path:'subscriptions',component:SubscriptionsView},{path:'members',component:MembersView},{path:'settings',component:GroupsView}]},{path:'/personal/expenses',component:ExpensesView},{path:'/personal/subscriptions',component:SubscriptionsView},{path:'/members',redirect:'/groups'},{path:'/subscriptions',redirect:'/personal/subscriptions'},{path:'/expenses',redirect:'/personal/expenses'},{path:'/profile',component:ProfileView}]})
router.beforeEach(to=>{if(!to.meta.public&&!pb.authStore.isValid)return{path:'/auth',query:{redirect:to.fullPath}};if(to.path==='/auth'&&pb.authStore.isValid)return'/';return true})

// @vitest-environment jsdom
import { reactive } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { createMemoryHistory, createRouter } from 'vue-router'

const auth=reactive({ready:true,authenticated:true,name:'Kenny',record:{id:'user-1',name:'Kenny',email:'kenny@example.com',timezone:'Asia/Taipei'},token:'token',initialize:vi.fn(async()=>{}),logout:vi.fn()})
const group={id:'group-1',name:'Trip',description:'Trip ledger',currency:'TWD',timezone:'Asia/Taipei',color:'#7357ff',ownerId:'user-1',createdAt:'',updatedAt:''}
const workspace=reactive({groups:[group],currentGroupId:'group-1',currentGroup:group,currentMembership:{userId:'user-1',role:'owner'},isOwner:true,members:[],invitations:[],subscriptions:[],expenses:[],settlements:[],groupRoles:[],ownershipTransfer:undefined,groupPermissions:['group.audit.read'],groupAuditLogs:[],groupErrors:{access:'',members:'',subscriptions:'',expenses:'',settlements:'',summary:''},personalSubscriptions:[],personalExpenses:[],personalSummary:null,summary:null,loading:false,error:'',localizedError:'',permissionDenied:false,loadGroups:vi.fn(async()=>{}),selectGroup:vi.fn(async()=>{}),refreshDashboard:vi.fn(async()=>{}),refreshPersonal:vi.fn(async()=>{}),refreshGroup:vi.fn(async()=>{}),loadGroupRoles:vi.fn(async()=>{}),loadOwnershipTransfer:vi.fn(async()=>{}),createOwnershipTransfer:vi.fn(async()=>true),respondOwnershipTransfer:vi.fn(async()=>true),cancelOwnershipTransfer:vi.fn(async()=>true),loadGroupAuditLogs:vi.fn(async()=>{}),retryLast:vi.fn()})
const setup=reactive({ready:true,initialized:true,status:{initialized:true},refresh:vi.fn(async()=>({initialized:true}))})
vi.mock('./stores/auth',()=>({useAuthStore:()=>auth}))
vi.mock('./stores/workspace',()=>({useWorkspaceStore:()=>workspace}))
vi.mock('./stores/setup',()=>({useSetupStore:()=>setup}))
vi.mock('./pocketbase',()=>({pb:{authStore:{isValid:true}}}))
vi.mock('./theme',()=>({useTheme:()=>({preference:{value:'system'},resolved:{value:'dark'},setTheme:vi.fn(),toggle:vi.fn()})}))
// Supplied by vite-plugin-pwa at build time; it does not resolve under vitest.
vi.mock('virtual:pwa-register/vue',()=>({useRegisterSW:()=>({needRefresh:{value:false},offlineReady:{value:false},updateServiceWorker:vi.fn()})}))

import App from './App.vue'
import { routes } from './router'

describe('navigation stability',()=>{
  beforeEach(()=>{vi.clearAllMocks();Object.assign(workspace.groupErrors,{access:'',members:'',subscriptions:'',expenses:'',settlements:'',summary:''})})
  it('keeps rendering after more than thirty route changes',async()=>{
    const router=createRouter({history:createMemoryHistory(),routes})
    await router.push('/');await router.isReady()
    const wrapper=mount(App,{global:{plugins:[router],stubs:{LanguageSwitcher:true,ThemeSwitcher:true,ToastContainer:true,TimezoneMismatchDialog:true}}})
    const targets=['/','/personal/expenses','/personal/subscriptions','/groups','/groups/group-1/overview','/groups/group-1/expenses','/groups/group-1/subscriptions','/groups/group-1/members','/groups/group-1/audit','/groups/group-1/settings','/about']
    for(let index=0;index<36;index++){await router.push(targets[index%targets.length]);await new Promise(resolve=>setTimeout(resolve,0));const context=`route ${router.currentRoute.value.fullPath}, iteration ${index}`;expect(wrapper.find('.main').exists(),context).toBe(true);expect(wrapper.find('.error-state').exists(),context).toBe(false);expect(wrapper.find('.main section').exists(),context).toBe(true)}
    wrapper.unmount()
  })
  it('keeps the shell available when one group resource fails',async()=>{
    workspace.groupErrors.subscriptions='Subscriptions could not be loaded'
    const router=createRouter({history:createMemoryHistory(),routes})
    await router.push('/groups/group-1/subscriptions');await router.isReady()
    const wrapper=mount(App,{global:{plugins:[router],stubs:{LanguageSwitcher:true,ThemeSwitcher:true,ToastContainer:true,TimezoneMismatchDialog:true}}})
    expect(wrapper.find('.error-state').exists()).toBe(false)
    expect(wrapper.find('.resource-error').text()).toContain('Subscriptions could not be loaded')
    wrapper.unmount()
  })
})

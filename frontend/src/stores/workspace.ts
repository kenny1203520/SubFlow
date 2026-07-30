import { computed,ref } from 'vue'
import { defineStore } from 'pinia'
import { ApiClient,ApiError } from '../api/client'
import { SSEClient } from '../api/sse'
import type { Currency,DashboardSummary,Expense,Group,Invitation,Membership,Subscription,SubFlowEvent } from '../api/types'
import { useAuthStore } from './auth'

export const useWorkspaceStore=defineStore('workspace',()=>{
  const auth=useAuthStore();const api=new ApiClient(()=>auth.token,auth.logout)
  const groups=ref<Group[]>([]),currentGroupId=ref(''),members=ref<Membership[]>([]),invitations=ref<Invitation[]>([]),subscriptions=ref<Subscription[]>([]),expenses=ref<Expense[]>([]),summary=ref<DashboardSummary|null>(null),loading=ref(false),error=ref(''),permissionDenied=ref(false)
  const currentGroup=computed(()=>groups.value.find(v=>v.id===currentGroupId.value));let sse:SSEClient|undefined
  async function run(task:()=>Promise<void>){loading.value=true;error.value='';permissionDenied.value=false;try{await task()}catch(e){const value=e as {status?:number;message?:string};permissionDenied.value=value.status===403;error.value=value.message??'操作失敗'}finally{loading.value=false}}
  async function loadGroups(){await run(async()=>{groups.value=(await api.get<Group[]>('/groups?perPage=100')).data;if(!currentGroupId.value&&groups.value[0])currentGroupId.value=groups.value[0].id})}
  async function selectGroup(id:string){currentGroupId.value=id;sse?.stop();sse=new SSEClient(()=>auth.token,onEvent,auth.logout);void sse.start(id);await refreshGroup()}
  async function refreshGroup(){if(!currentGroupId.value)return;const id=currentGroupId.value;await run(async()=>{const [m,s,e,d]=await Promise.all([api.get<Membership[]>(`/groups/${id}/members?perPage=100`),api.get<Subscription[]>(`/groups/${id}/subscriptions?perPage=100`),api.get<Expense[]>(`/groups/${id}/expenses?perPage=100`),api.get<DashboardSummary>(`/groups/${id}/summary`)]);members.value=m.data;subscriptions.value=s.data;expenses.value=e.data;summary.value=d.data;const me=m.data.find(member=>member.userId===auth.record?.id);if(me?.role==='owner'){invitations.value=(await api.get<Invitation[]>(`/groups/${id}/invitations?perPage=100`)).data}else{invitations.value=[]}})}
  async function onEvent(event:SubFlowEvent){if(event.groupId!==currentGroupId.value)return;if(event.resource==='groups'||event.resource==='group_members')await loadGroups();await refreshGroup()}
  async function createGroup(input:{name:string;description:string;currency:Currency;color:string}){await run(async()=>{const group=(await api.post<Group>('/groups',input)).data;groups.value.unshift(group);await selectGroup(group.id)})}
  async function invite(email:string){if(!currentGroupId.value)return;await run(async()=>{const v=(await api.post<Invitation>(`/groups/${currentGroupId.value}/invitations`,{email})).data;invitations.value.unshift(v)})}
  async function resendInvitation(id:string){await run(async()=>{const v=(await api.post<Invitation>(`/invitations/${id}/resend`)).data;invitations.value=invitations.value.map(x=>x.id===id?v:x)})}
  async function revokeInvitation(id:string){await run(async()=>{await api.post(`/invitations/${id}/revoke`);await refreshGroup()})}
  async function acceptInvitation(token:string){await run(async()=>{const v=(await api.post<Invitation>('/invitations/accept',{token})).data;await loadGroups();await selectGroup(v.groupId)})}
  async function addSubscription(input:Omit<Subscription,'id'|'groupId'|'createdAt'|'updatedAt'>){if(!currentGroupId.value)return;await run(async()=>{subscriptions.value.unshift((await api.post<Subscription>(`/groups/${currentGroupId.value}/subscriptions`,input)).data);await refreshGroup()})}
  async function deleteSubscription(id:string){await run(async()=>{await api.delete(`/subscriptions/${id}`);await refreshGroup()})}
  async function addExpense(input:Omit<Expense,'id'|'groupId'|'createdAt'|'updatedAt'>){if(!currentGroupId.value)return;await run(async()=>{expenses.value.unshift((await api.post<Expense>(`/groups/${currentGroupId.value}/expenses`,input)).data);await refreshGroup()})}
  async function deleteExpense(id:string){await run(async()=>{await api.delete(`/expenses/${id}`);await refreshGroup()})}
  function clear(){sse?.stop();groups.value=[];currentGroupId.value='';members.value=[];invitations.value=[];subscriptions.value=[];expenses.value=[];summary.value=null}
  function isForbidden(value:unknown){return value instanceof ApiError&&value.status===403}
  return{groups,currentGroupId,currentGroup,members,invitations,subscriptions,expenses,summary,loading,error,permissionDenied,loadGroups,selectGroup,refreshGroup,createGroup,invite,resendInvitation,revokeInvitation,acceptInvitation,addSubscription,deleteSubscription,addExpense,deleteExpense,clear,isForbidden}
})

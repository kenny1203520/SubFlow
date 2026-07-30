import { computed,ref } from 'vue'
import { defineStore } from 'pinia'
import { pb } from '../pocketbase'

export const useAuthStore=defineStore('auth',()=>{
  const record=ref(pb.authStore.record)
  const ready=ref(false)
  const authenticated=computed(()=>pb.authStore.isValid&&!!record.value)
  const token=computed(()=>pb.authStore.token)
  const name=computed(()=>String(record.value?.name||record.value?.email||'使用者'))
  pb.authStore.onChange((_token,next)=>{record.value=next})
  async function initialize(){if(pb.authStore.isValid){try{await pb.collection('users').authRefresh()}catch{pb.authStore.clear()}}record.value=pb.authStore.record;ready.value=true}
  async function login(email:string,password:string){await pb.collection('users').authWithPassword(email,password);record.value=pb.authStore.record}
  async function register(input:{email:string;password:string;name:string}){await pb.collection('users').create({email:input.email,password:input.password,passwordConfirm:input.password,name:input.name,timezone:Intl.DateTimeFormat().resolvedOptions().timeZone});await login(input.email,input.password)}
  async function updateProfile(input:{name:string;timezone:string}){if(!record.value)throw new Error('尚未登入');record.value=await pb.collection('users').update(record.value.id,input)}
  function logout(){pb.authStore.clear();record.value=null}
  return{record,ready,authenticated,token,name,initialize,login,register,updateProfile,logout}
})

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ApiClient } from '../api/client'
import type { AccessRole, AuditLog } from '../api/types'
import { useAuthStore } from '../stores/auth'
import { useI18n } from '../i18n'

const auth=useAuthStore(),{tr,formatDate}=useI18n()
const api=new ApiClient(()=>auth.token,auth.logout)
const roles=ref<AccessRole[]>([]),logs=ref<AuditLog[]>([]),name=ref(''),key=ref(''),permissions=ref(''),error=ref('')
async function load(){try{const [roleResult,logResult]=await Promise.all([api.get<AccessRole[]>('/system/roles'),api.get<AuditLog[]>('/system/audit-logs?perPage=100')]);roles.value=roleResult.data;logs.value=logResult.data}catch{error.value=tr('requestFailed')}}
async function create(){try{const value=(await api.post<AccessRole>('/system/roles',{name:name.value,key:key.value,permissions:permissions.value.split(',').map(value=>value.trim()).filter(Boolean)})).data;roles.value.push(value);name.value='';key.value='';permissions.value=''}catch{error.value=tr('requestFailed')}}
onMounted(()=>void load())
</script>
<template><section class="page narrow"><div class="page-heading"><div><p class="eyebrow">{{tr('settings')}}</p><h1>{{tr('settings')}}</h1><p>{{tr('accountSettings')}}</p></div></div><p v-if="error" class="notice danger">{{error}}</p><section class="card form-card"><h2>{{tr('members')}}</h2><div class="data-list"><article v-for="role in roles" :key="role.id" class="data-row"><div class="grow"><strong>{{role.name}}</strong><small>{{role.key}} · {{role.permissions.join(', ')||'—'}}</small></div><span v-if="role.protected" class="pill">{{tr('status')}}</span></article></div><form class="inline-create" @submit.prevent="create"><input v-model="name" required :placeholder="tr('name')"><input v-model="key" required :placeholder="tr('category')"><input v-model="permissions" :placeholder="tr('notes')"><button class="primary">{{tr('add')}}</button></form></section><section class="card"><h2>{{tr('records',{count:logs.length})}}</h2><div class="data-list"><article v-for="log in logs" :key="log.id" class="data-row"><div class="grow"><strong>{{log.action}}</strong><small>{{log.resource}} · {{formatDate(log.createdAt)}} · {{log.outcome}}</small></div></article><p v-if="!logs.length" class="empty-inline">{{tr('noSummary')}}</p></div></section></section></template>

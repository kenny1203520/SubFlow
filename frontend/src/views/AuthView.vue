<script setup lang="ts">
import { ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import LanguageSwitcher from '../components/LanguageSwitcher.vue'
import ThemeSwitcher from '../components/ThemeSwitcher.vue'
const auth=useAuthStore(),router=useRouter(),route=useRoute(),mode=ref<'login'|'register'>('login'),email=ref(''),password=ref(''),name=ref(''),error=ref(''),busy=ref(false)
async function submit(){busy.value=true;error.value='';try{if(mode.value==='login')await auth.login(email.value,password.value);else await auth.register({email:email.value,password:password.value,name:name.value});await router.replace(String(route.query.redirect||'/'))}catch(reason){error.value=reason instanceof Error?reason.message:'無法登入'}finally{busy.value=false}}
</script>
<template><main class="auth-page"><div class="auth-controls"><ThemeSwitcher/><LanguageSwitcher/></div><section class="auth-story"><div class="brand hero-brand"><span>SF</span><strong>SubFlow</strong></div><p class="eyebrow">SHARED MONEY, CLEARLY</p><h1>共享財務，清楚管理</h1><p>與團隊一起管理支出、訂閱與共同財務。</p></section><section class="auth-panel"><div class="auth-card"><div class="segmented"><button type="button" :class="{active:mode==='login'}" @click="mode='login'">登入</button><button type="button" :class="{active:mode==='register'}" @click="mode='register'">建立帳號</button></div><h2>{{mode==='login'?'歡迎回來':'開始使用 SubFlow'}}</h2><form @submit.prevent="submit"><label v-if="mode==='register'">顯示名稱<input v-model="name" required autocomplete="name"></label><label>Email<input v-model="email" type="email" required autocomplete="email"></label><label>密碼<input v-model="password" type="password" minlength="8" required :autocomplete="mode==='login'?'current-password':'new-password'"></label><p v-if="error" class="form-error">{{error}}</p><button class="primary wide" :disabled="busy">{{busy?'處理中…':mode==='login'?'登入':'建立帳號'}}</button></form></div></section></main></template>

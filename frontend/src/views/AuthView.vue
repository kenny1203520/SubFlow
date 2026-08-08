<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRoute,useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import { useI18n } from '../i18n'
import ThemeSwitcher from '../components/ThemeSwitcher.vue'
import LanguageSwitcher from '../components/LanguageSwitcher.vue'
const auth=useAuthStore(),router=useRouter(),route=useRoute(),{tr}=useI18n()
const mode=ref<'login'|'register'>('login'),email=ref(''),password=ref(''),name=ref(''),busy=ref(false),error=ref(''),providers=ref<Array<{name:string;displayName?:string}>>([])
async function submit(){busy.value=true;error.value='';try{if(mode.value==='login')await auth.login(email.value,password.value);else await auth.register({email:email.value,password:password.value,name:name.value});await router.replace(String(route.query.redirect||'/'))}catch(reason){error.value=reason instanceof Error?reason.message:tr('loginFailed')}finally{busy.value=false}}
function oauth(provider:string){busy.value=true;error.value='';auth.loginOAuth(provider).then(()=>router.replace(String(route.query.redirect||'/'))).catch(reason=>{error.value=reason instanceof Error?reason.message:tr('loginFailed')}).finally(()=>busy.value=false)}
onMounted(()=>{void auth.oauthProviders().then(values=>providers.value=values).catch(()=>{providers.value=[]})})
</script>
<template><main class="auth-page">
  <div class="auth-controls"><ThemeSwitcher/><LanguageSwitcher/></div>
  <section class="auth-story"><div class="brand hero-brand"><span>SF</span><strong>SubFlow</strong></div><p class="eyebrow">{{tr('sharedMoneyClearly')}}</p><h1>{{tr('sharedFinance')}}</h1><p>{{tr('sharedFinanceDesc')}}</p></section>
  <section class="auth-panel"><div class="auth-card"><div class="segmented"><button type="button" :class="{active:mode==='login'}" @click="mode='login'">{{tr('login')}}</button><button type="button" :class="{active:mode==='register'}" @click="mode='register'">{{tr('register')}}</button></div><h2>{{mode==='login'?tr('welcomeBack'):tr('startSubFlow')}}</h2><form @submit.prevent="submit"><label v-if="mode==='register'">{{tr('displayName')}}<input v-model="name" required autocomplete="name"></label><label>{{tr('email')}}<input v-model="email" type="email" required autocomplete="email"></label><label>{{tr('password')}}<input v-model="password" type="password" minlength="8" required :autocomplete="mode==='login'?'current-password':'new-password'"></label><p v-if="error" class="form-error">{{error}}</p><button class="primary wide" :disabled="busy">{{busy?tr('processing'):mode==='login'?tr('login'):tr('register')}}</button></form><div v-if="mode==='login'&&providers.length" class="oauth-options"><span>or</span><button v-for="provider in providers" :key="provider.name" type="button" class="ghost wide" :disabled="busy" @click="oauth(provider.name)">{{provider.displayName||provider.name}}</button></div></div></section>
</main></template>

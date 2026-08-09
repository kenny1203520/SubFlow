<script setup lang="ts">
import { onMounted, ref, watch } from 'vue'
import { useRoute,useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import { useI18n } from '../i18n'
import { useSetupStore } from '../stores/setup'
import ThemeSwitcher from '../components/ThemeSwitcher.vue'
import LanguageSwitcher from '../components/LanguageSwitcher.vue'
import BaseInput from '../components/BaseInput.vue'
import PasswordField from '../components/PasswordField.vue'
const auth=useAuthStore(),setup=useSetupStore(),router=useRouter(),route=useRoute(),{tr}=useI18n()
const mode=ref<'login'|'register'>('login'),email=ref(''),password=ref(''),name=ref(''),busy=ref(false),error=ref(''),registered=ref(false),providers=ref<Array<{name:string;displayName?:string}>>([])
async function submit(){busy.value=true;error.value='';registered.value=false;try{if(mode.value==='login'){await auth.login(email.value,password.value);await router.replace(String(route.query.redirect||'/'))}else{await auth.register({email:email.value,password:password.value,name:name.value});registered.value=true;mode.value='login';password.value=''}}catch(reason){error.value=reason instanceof Error?reason.message:tr('loginFailed')}finally{busy.value=false}}
function oauth(provider:string){busy.value=true;error.value='';auth.loginOAuth(provider).then(()=>router.replace(String(route.query.redirect||'/'))).catch(reason=>{error.value=reason instanceof Error?reason.message:tr('loginFailed')}).finally(()=>busy.value=false)}
onMounted(()=>{void setup.refresh().then(status=>{if(!status.initialized){void router.replace('/setup')}}).catch(()=>{});void auth.oauthProviders().then(values=>providers.value=values).catch(()=>{providers.value=[]})})
watch(()=>setup.allowRegistration,value=>{if(!value&&mode.value==='register')mode.value='login'})
</script>
<template><main class="auth-page"><div class="auth-controls"><ThemeSwitcher/><LanguageSwitcher/></div><section class="auth-story"><div class="brand hero-brand"><span>SF</span><strong>SubFlow</strong></div><p class="eyebrow">{{tr('sharedMoneyClearly')}}</p><h1>{{tr('sharedFinance')}}</h1><p>{{tr('sharedFinanceDesc')}}</p></section><section class="auth-panel"><div class="auth-card"><div v-if="setup.allowRegistration" class="segmented"><button type="button" :class="{active:mode==='login'}" @click="mode='login'">{{tr('login')}}</button><button type="button" :class="{active:mode==='register'}" @click="mode='register'">{{tr('register')}}</button></div><h2>{{mode==='login'?tr('welcomeBack'):tr('startSubFlow')}}</h2><form @submit.prevent="submit"><BaseInput v-if="mode==='register'" v-model="name" :label="tr('displayName')" required autocomplete="name"/><BaseInput v-model="email" :label="tr('email')" type="email" required autocomplete="email"/><PasswordField v-model="password" :label="tr('password')" :minlength="8" required :autocomplete="mode==='login'?'current-password':'new-password'"/><p v-if="registered" class="notice success">{{tr('verificationSent')}}</p><p v-if="error" class="form-error">{{error}}</p><button class="primary wide" :disabled="busy">{{busy?tr('processing'):mode==='login'?tr('login'):tr('register')}}</button></form><div v-if="mode==='login'&&providers.length" class="oauth-options"><span>or</span><button v-for="provider in providers" :key="provider.name" type="button" class="ghost wide" :disabled="busy" @click="oauth(provider.name)">{{provider.displayName||provider.name}}</button></div></div></section></main></template>

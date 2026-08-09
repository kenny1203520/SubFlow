<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute,useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import { useI18n } from '../i18n'
import { useSetupStore } from '../stores/setup'
import ThemeSwitcher from '../components/ThemeSwitcher.vue'
import LanguageSwitcher from '../components/LanguageSwitcher.vue'
import BaseInput from '../components/BaseInput.vue'
import PasswordField from '../components/PasswordField.vue'
import CaptchaChallenge from '../components/CaptchaChallenge.vue'

const auth=useAuthStore(), setup=useSetupStore(), router=useRouter(), route=useRoute(), {tr}=useI18n()
const mode=ref<'login'|'register'|'otp'|'mfa'>('login'), email=ref(''), password=ref(''), name=ref(''), code=ref(''), otpId=ref(''), mfaId=ref(''), captchaToken=ref(''), busy=ref(false), error=ref(''), registered=ref(false)
const providers=ref<Array<{name:string;displayName?:string}>>([]), methods=ref<{otp?:{enabled?:boolean};mfa?:{enabled?:boolean};password?:{enabled?:boolean}}>({})
const canPasswordRegistration=computed(()=>setup.allowRegistration)
function authError(reason: unknown) {
  const raw=reason instanceof Error ? reason.message : ''
  if (raw.includes('oidc_verified_email_required') || raw.includes('cannot be blank')) return tr('oidcVerifiedEmailRequired')
  if (raw.includes('captcha')) return tr('captchaVerificationFailed')
  return raw || tr('loginFailed')
}
async function finish(){ await router.replace(String(route.query.redirect||'/')) }
async function requestCode(next: 'otp'|'mfa') { const result=await auth.requestOTP(email.value, captchaToken.value); otpId.value=result.otpId; mode.value=next }
async function submit(){
  busy.value=true; error.value=''; registered.value=false
  try {
    if(mode.value==='login') { await auth.login(email.value,password.value); await finish() }
    else if(mode.value==='register') { await auth.register({email:email.value,password:password.value,name:name.value, captchaToken:captchaToken.value} as any); registered.value=true; mode.value='login'; password.value='' }
    else { await auth.loginOTP(otpId.value,code.value,mfaId.value); await finish() }
  } catch(reason:any) {
    const id=reason?.response?.mfaId || reason?.data?.mfaId
    if(id) { mfaId.value=id; try { await requestCode('mfa') } catch(requestReason) { error.value=authError(requestReason) } }
    else error.value=authError(reason)
  } finally { busy.value=false }
}
async function startOTP(){ busy.value=true; error.value=''; try { await requestCode('otp') } catch(reason) { error.value=authError(reason) } finally { busy.value=false } }
function cancelCode(){ mode.value='login'; code.value=''; otpId.value=''; mfaId.value='' }
function oauth(provider:string){ busy.value=true; error.value=''; auth.loginOAuth(provider).then(finish).catch(reason=>error.value=authError(reason)).finally(()=>busy.value=false) }
onMounted(async()=>{ try { const status=await setup.refresh(); if(!status.initialized){ await router.replace('/setup'); return }; providers.value=await auth.oauthProviders(); methods.value=await (await import('../pocketbase')).pb.collection('users').listAuthMethods() } catch { providers.value=[] } })
watch(()=>setup.allowRegistration,value=>{if(!value&&mode.value==='register')mode.value='login'})
</script>
<template><main class="auth-page"><div class="auth-controls"><ThemeSwitcher/><LanguageSwitcher/></div><section class="auth-story"><div class="brand hero-brand"><span>SF</span><strong>SubFlow</strong></div><p class="eyebrow">{{tr('sharedMoneyClearly')}}</p><h1>{{tr('sharedFinance')}}</h1><p>{{tr('sharedFinanceDesc')}}</p></section><section class="auth-panel"><div class="auth-card">
  <div v-if="canPasswordRegistration && (mode==='login'||mode==='register')" class="segmented"><button type="button" :class="{active:mode==='login'}" @click="mode='login'">{{tr('login')}}</button><button type="button" :class="{active:mode==='register'}" @click="mode='register'">{{tr('register')}}</button></div>
  <h2>{{mode==='register'?tr('startSubFlow'):mode==='mfa'?tr('mfaVerification'):mode==='otp'?tr('emailOtpLogin'):tr('welcomeBack')}}</h2>
  <form @submit.prevent="submit"><BaseInput v-if="mode==='register'" v-model="name" :label="tr('displayName')" required autocomplete="name"/><BaseInput v-if="mode==='login'||mode==='register'||mode==='otp'||mode==='mfa'" v-model="email" :label="tr('email')" type="email" required autocomplete="email" :disabled="mode==='mfa'"/><PasswordField v-if="mode==='login'||mode==='register'" v-model="password" :label="tr('password')" :minlength="8" required :autocomplete="mode==='login'?'current-password':'new-password'"/><CaptchaChallenge v-if="mode==='register'||mode==='otp'||mode==='mfa'" v-model="captchaToken"/><BaseInput v-if="mode==='otp'||mode==='mfa'" v-model="code" :label="tr('verificationCode')" required autocomplete="one-time-code" inputmode="numeric"/>
  <p v-if="registered" class="notice success">{{tr('verificationSent')}}</p><p v-if="error" class="form-error">{{error}}</p><button class="primary wide" :disabled="busy">{{busy?tr('processing'):mode==='login'?tr('login'):mode==='register'?tr('register'):tr('verify')}}</button></form>
  <div v-if="(mode==='otp'||mode==='mfa')" class="auth-actions"><button type="button" class="ghost" :disabled="busy" @click="startOTP">{{tr('resendCode')}}</button><button type="button" class="ghost" :disabled="busy" @click="cancelCode">{{tr('cancel')}}</button></div>
  <div v-if="mode==='login' && methods.otp?.enabled" class="oauth-options"><button type="button" class="ghost wide" :disabled="busy" @click="startOTP">{{tr('emailOtpLogin')}}</button></div>
  <div v-if="mode==='login'&&providers.length" class="oauth-options"><span>{{tr('or')}}</span><button v-for="provider in providers" :key="provider.name" type="button" class="ghost wide" :disabled="busy" @click="oauth(provider.name)">{{provider.displayName||provider.name}}</button></div>
</div></section></main></template>

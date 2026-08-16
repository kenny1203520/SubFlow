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

type AuthMode = 'login' | 'register' | 'otp-request' | 'otp-verify' | 'mfa-verify'
const auth=useAuthStore(), setup=useSetupStore(), router=useRouter(), route=useRoute(), {tr}=useI18n()
const mode=ref<AuthMode>('login'), email=ref(''), password=ref(''), name=ref(''), code=ref(''), otpId=ref(''), mfaId=ref(''), captchaToken=ref(''), busy=ref(false), error=ref(''), registered=ref(false)
const providers=ref<Array<{name:string;displayName?:string}>>([]), methods=ref<{otp?:{enabled?:boolean};mfa?:{enabled?:boolean};password?:{enabled?:boolean}}>({})
const canPasswordRegistration=computed(()=>setup.allowRegistration)
const isCodeMode=computed(()=>mode.value==='otp-verify'||mode.value==='mfa-verify')
const captchaRef=ref<{ solve: () => Promise<string> } | null>(null)
function captchaFlowKey(m:AuthMode) { return m==='register'?'register':m==='otp-request'?'otpRequest':m==='login'?'login':'' }
const captchaConfig=computed(()=>{ const key=captchaFlowKey(mode.value); return key ? setup.status.captchaFlows?.[key as 'register'|'otpRequest'|'login'] : undefined })
const captchaEnabled=computed(()=>!!captchaConfig.value?.enabled)
// When the active flow's trigger is 'submit', the widget hasn't rendered/solved
// yet -- ask it to solve now and use the resolved token directly rather than
// relying on captchaToken, which the submit-mode widget never populates ahead of time.
async function resolveCaptchaToken() {
  if(!captchaEnabled.value) return ''
  if(captchaConfig.value?.trigger==='submit') return (captchaToken.value=await (captchaRef.value?.solve()||Promise.resolve('')))
  return captchaToken.value
}

function authError(reason: any) {
  const raw=[reason?.message, reason?.response?.message, reason?.data?.message, reason?.response?.data?.message].filter(Boolean).join(' ').toLowerCase()
  if(raw.includes('oidc_verified_email_required')||raw.includes('cannot be blank')) return tr('oidcVerifiedEmailRequired')
  if(raw.includes('captcha')) return tr('captchaVerificationFailed')
  if(raw.includes('invalid or expired otp')||raw.includes('invalid otp')) return tr('otpInvalidOrExpired')
  if(raw.includes('mfa')) return tr('mfaExpired')
  if(raw.includes('invalid')||raw.includes('authenticate')||raw.includes('password')) return tr('invalidCredentials')
  if(raw.includes('registration')||raw.includes('forbidden')) return tr('registrationDisabled')
  if(raw.includes('smtp')||raw.includes('mail')||raw.includes('email')) return tr('otpRequestFailed')
  return tr('authenticationUnavailable')
}
async function finish(){ await router.replace(String(route.query.redirect||'/')) }
function backToPasswordLogin(){ mode.value='login'; code.value=''; otpId.value=''; mfaId.value=''; captchaToken.value=''; error.value='' }
function startOtpRequest(){ mode.value='otp-request'; code.value=''; otpId.value=''; captchaToken.value=''; error.value='' }
async function requestStandaloneOtp(){
  busy.value=true; error.value=''
  try { const token=await resolveCaptchaToken(); const result=await auth.requestOTP(email.value,token); otpId.value=result.otpId; captchaToken.value=''; mode.value='otp-verify' }
  catch(reason){ error.value=authError(reason) } finally { busy.value=false }
}
async function requestMfaOtp(){
  const result=await auth.requestOTP(email.value)
  otpId.value=result.otpId; mode.value='mfa-verify'
}
async function passwordLogin(){
  try { const token=await resolveCaptchaToken(); await auth.login(email.value,password.value,token); await finish() }
  catch(reason:any) {
    const id=reason?.response?.mfaId||reason?.data?.mfaId
    if(!id) throw reason
    mfaId.value=id
    await requestMfaOtp()
  }
}
async function verifyCode(){
  busy.value=true; error.value=''
  try { await auth.loginOTP(otpId.value,code.value,mode.value==='mfa-verify'?mfaId.value:''); await finish() }
  catch(reason){ error.value=authError(reason) } finally { busy.value=false }
}
async function submit(){
  if(mode.value==='otp-request') return requestStandaloneOtp()
  if(isCodeMode.value) return verifyCode()
  busy.value=true; error.value=''; registered.value=false
  try {
    if(mode.value==='login') await passwordLogin()
    else { const token=await resolveCaptchaToken(); await auth.register({email:email.value,password:password.value,name:name.value,captchaToken:token} as any); registered.value=true; mode.value='login'; password.value='' }
  } catch(reason) { error.value=authError(reason) } finally { busy.value=false }
}
async function resend(){ if(mode.value==='mfa-verify') { busy.value=true;error.value='';try{await requestMfaOtp()}catch(reason){error.value=authError(reason)}finally{busy.value=false} } else await requestStandaloneOtp() }
function oauth(provider:string){ busy.value=true; error.value=''; auth.loginOAuth(provider).then(finish).catch(reason=>error.value=authError(reason)).finally(()=>busy.value=false) }
onMounted(async()=>{ try { const status=await setup.refresh(); if(!status.initialized){ await router.replace('/setup'); return }; providers.value=await auth.oauthProviders(); methods.value=await (await import('../pocketbase')).pb.collection('users').listAuthMethods() } catch { providers.value=[] } })
watch(()=>setup.allowRegistration,value=>{if(!value&&mode.value==='register')mode.value='login'})
</script>
<template>
  <main class="auth-page"><div class="auth-controls"><ThemeSwitcher/><LanguageSwitcher/></div>
    <section class="auth-story"><div class="brand hero-brand"><span>SF</span><strong>SubFlow</strong></div><p class="eyebrow">{{tr('sharedMoneyClearly')}}</p><h1>{{tr('sharedFinance')}}</h1><p>{{tr('sharedFinanceDesc')}}</p></section>
    <section class="auth-panel"><div class="auth-card">
      <div v-if="canPasswordRegistration&&(mode==='login'||mode==='register')" class="segmented"><button type="button" :class="{active:mode==='login'}" @click="mode='login'">{{tr('login')}}</button><button type="button" :class="{active:mode==='register'}" @click="mode='register'">{{tr('register')}}</button></div>
      <button v-if="mode==='otp-request'||isCodeMode" type="button" class="back-auth" @click="backToPasswordLogin">← {{tr('backToPasswordLogin')}}</button>
      <h2>{{mode==='register'?tr('startSubFlow'):mode==='mfa-verify'?tr('mfaVerification'):mode==='otp-request'?tr('emailOtpRequestTitle'):mode==='otp-verify'?tr('emailOtpVerificationTitle'):tr('welcomeBack')}}</h2>
      <p v-if="mode==='otp-request'||mode==='otp-verify'||mode==='mfa-verify'" class="auth-step-description">{{mode==='mfa-verify'?tr('mfaVerificationDesc'):mode==='otp-request'?tr('emailOtpRequestDesc'):tr('emailOtpVerificationDesc')}}</p>
      <form @submit.prevent="submit">
        <BaseInput v-if="mode==='register'" v-model="name" :label="tr('displayName')" required autocomplete="name"/>
        <BaseInput v-if="mode==='login'||mode==='register'||mode==='otp-request'||isCodeMode" v-model="email" :label="tr('email')" type="email" required autocomplete="email" :disabled="isCodeMode"/>
        <PasswordField v-if="mode==='login'||mode==='register'" v-model="password" :label="tr('password')" :minlength="8" required :autocomplete="mode==='login'?'current-password':'new-password'"/>
        <CaptchaChallenge v-if="captchaEnabled" ref="captchaRef" v-model="captchaToken" :trigger="captchaConfig?.trigger||'load'" :mode="captchaConfig?.mode||'interactive'"/>
        <BaseInput v-if="isCodeMode" v-model="code" :label="tr('verificationCode')" required autocomplete="one-time-code" inputmode="numeric"/>
        <p v-if="registered" class="notice success">{{tr('verificationSent')}}</p><p v-if="error" class="form-error">{{error}}</p>
        <button class="primary wide" :disabled="busy">{{busy?tr('processing'):mode==='login'?tr('login'):mode==='register'?tr('register'):mode==='otp-request'?tr('sendVerificationCode'):tr('verify')}}</button>
      </form>
      <div v-if="isCodeMode" class="auth-actions"><button type="button" class="ghost" :disabled="busy" @click="resend">{{tr('resendCode')}}</button></div>
      <div v-if="mode==='login'&&methods.otp?.enabled" class="oauth-options"><button type="button" class="ghost wide" :disabled="busy" @click="startOtpRequest">{{tr('emailOtpLogin')}}</button></div>
      <div v-if="mode==='login'&&providers.length" class="oauth-options"><span>{{tr('or')}}</span><button v-for="provider in providers" :key="provider.name" type="button" class="ghost wide" :disabled="busy" @click="oauth(provider.name)">{{provider.displayName||provider.name}}</button></div>
    </div></section>
  </main>
</template>

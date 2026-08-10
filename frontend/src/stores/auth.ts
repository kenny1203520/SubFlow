import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import { pb } from '../pocketbase'
import { useI18n } from '../i18n'
import { ApiClient } from '../api/client'
import type { SystemAccess } from '../api/types'

export const useAuthStore = defineStore('auth', () => {
  const { tr } = useI18n()
  const record = ref(pb.authStore.record)
  const authToken = ref(pb.authStore.token)
  const authValid = ref(pb.authStore.isValid)
  const ready = ref(false)
  const permissions = ref<string[]>([])
  const authenticated = computed(() => authValid.value && !!authToken.value && !!record.value)
  const token = computed(() => authToken.value)
  const name = computed(() => String(record.value?.name || record.value?.email || tr('userFallback')))
  const canAdminister = computed(() => permissions.value.some(value => value === '*' || value.startsWith('system.')))

  pb.authStore.onChange((nextToken, nextRecord) => {
    authToken.value = nextToken
    authValid.value = pb.authStore.isValid
    record.value = nextRecord
  })

  async function refreshAccess() {
    if (!pb.authStore.isValid) { permissions.value=[]; return }
    try {
      const api = new ApiClient(() => pb.authStore.token, logout)
      permissions.value = (await api.get<SystemAccess>('/system/access')).data.permissions || []
    } catch { permissions.value=[] }
  }

  async function initialize() {
    if (pb.authStore.isValid) {
      try { await pb.collection('users').authRefresh() } catch { pb.authStore.clear() }
    }
    authToken.value = pb.authStore.token
    authValid.value = pb.authStore.isValid
    record.value = pb.authStore.record
    await refreshAccess()
    ready.value = true
  }

  async function login(email: string, password: string) {
    await pb.collection('users').authWithPassword(email, password)
    authToken.value = pb.authStore.token
    authValid.value = pb.authStore.isValid
    record.value = pb.authStore.record
    await refreshAccess()
    ready.value = true
  }

  async function register(input: { email: string; password: string; name: string; captchaToken?: string }) {
    const api = new ApiClient(() => '', () => {})
    await api.post('/auth/register', { email: input.email, password: input.password, adminName: input.name, captchaToken: input.captchaToken || '' })
    await pb.collection('users').requestVerification(input.email)
  }
  async function requestPasswordReset(email: string, captchaToken = '') {
    await pb.collection('users').requestPasswordReset(email, { headers: captchaToken ? { 'X-SubFlow-Captcha': captchaToken } : {} })
  }
  async function requestOTP(email: string, captchaToken = '') {
    return pb.collection('users').requestOTP(email, { headers: captchaToken ? { 'X-SubFlow-Captcha': captchaToken } : {} })
  }
  async function loginOTP(otpId: string, password: string, mfaId = '') {
    await pb.collection('users').authWithOTP(otpId, password, mfaId ? { mfaId } : {})
    authToken.value = pb.authStore.token; authValid.value = pb.authStore.isValid; record.value = pb.authStore.record
    await refreshAccess(); ready.value = true
  }

  async function updateProfile(input: { name: string; timezone: string; default_currency?: string }) {
    if (!record.value) throw new Error(tr('notSignedIn'))
    record.value = await pb.collection('users').update(record.value.id, input)
  }
  async function oauthProviders() { const methods=await pb.collection('users').listAuthMethods(); return methods.oauth2?.providers||[] }
  async function loginOAuth(provider:string) { await pb.collection('users').authWithOAuth2({provider,createData:{timezone:Intl.DateTimeFormat().resolvedOptions().timeZone,default_currency:'TWD'}}); await refreshAccess() }
  function logout() {
    pb.authStore.clear(); authToken.value = ''; authValid.value = false; record.value = null; permissions.value=[]
  }
  return { record, ready, authenticated, token, name, permissions, canAdminister, initialize, login, register, requestPasswordReset, requestOTP, loginOTP, updateProfile, oauthProviders, loginOAuth, refreshAccess, logout }
})

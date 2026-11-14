<script setup lang="ts">
import { computed, nextTick, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import { useI18n } from '../i18n'
import { useSetupStore } from '../stores/setup'
import ThemeSwitcher from '../components/ThemeSwitcher.vue'
import LanguageSwitcher from '../components/LanguageSwitcher.vue'
import Logo from '../components/Logo.vue'
import BaseInput from '../components/BaseInput.vue'
import PasswordField from '../components/PasswordField.vue'
import CaptchaChallenge from '../components/CaptchaChallenge.vue'
import OAuthProviderList from '../components/OAuthProviderList.vue'
import type { OAuthProviderInfo } from '../stores/auth'

type AuthMode = 'login' | 'register' | 'reset-request' | 'reset-confirm' | 'email-change-confirm' | 'otp-request' | 'otp-verify' | 'mfa-verify'
type CaptchaFlow = 'register' | 'passwordReset' | 'otpRequest' | 'login'

const auth = useAuthStore()
const setup = useSetupStore()
const router = useRouter()
const route = useRoute()
const { tr } = useI18n()

const mode = ref<AuthMode>('login')
const email = ref('')
const password = ref('')
const name = ref('')
const code = ref('')
const otpId = ref('')
const mfaId = ref('')
const captchaToken = ref('')
const resetToken = ref('')
const resetPassword = ref('')
const resetPasswordConfirm = ref('')
const emailChangePassword = ref('')
const busy = ref(false)
const error = ref('')
const notice = ref('')
const registered = ref(false)
const providers = ref<OAuthProviderInfo[]>([])
const methods = ref<{ otp?: { enabled?: boolean }; mfa?: { enabled?: boolean }; password?: { enabled?: boolean } }>({})
const authCardRef = ref<HTMLElement | null>(null)

const canPasswordRegistration = computed(() => setup.allowRegistration)
const isCodeMode = computed(() => mode.value === 'otp-verify' || mode.value === 'mfa-verify')
const captchaRef = ref<{ solve: () => Promise<string>; reset: () => void } | null>(null)

function captchaFlowKey(): CaptchaFlow | '' {
  if (mode.value === 'register') return 'register'
  if (mode.value === 'reset-request') return 'passwordReset'
  if (mode.value === 'otp-request' || mode.value === 'mfa-verify') return 'otpRequest'
  if (mode.value === 'login') return 'login'
  return ''
}

const captchaConfig = computed(() => {
  const key = captchaFlowKey()
  return key ? setup.status.captchaFlows?.[key] : undefined
})
const captchaEnabled = computed(() => !!captchaConfig.value?.enabled)

async function resolveCaptchaToken() {
  if (!captchaEnabled.value) return ''
  if (captchaConfig.value?.trigger === 'submit') {
    return (captchaToken.value = await (captchaRef.value?.solve() || Promise.resolve('')))
  }
  return captchaToken.value
}

function resetCaptcha() {
  captchaToken.value = ''
  captchaRef.value?.reset()
}

function authError(reason: unknown) {
  const value = reason as { message?: string; response?: { message?: string; data?: { message?: string; mfaId?: string } }; data?: { message?: string; mfaId?: string } }
  const raw = [value?.message, value?.response?.message, value?.data?.message, value?.response?.data?.message].filter(Boolean).join(' ').toLowerCase()
  if (raw.includes('oidc_verified_email_required') || raw.includes('cannot be blank')) return tr('oidcVerifiedEmailRequired')
  if (raw.includes('captcha')) return tr('captchaVerificationFailed')
  if (raw.includes('invalid or expired otp') || raw.includes('invalid otp')) return tr('otpInvalidOrExpired')
  if (raw.includes('mfa')) return tr('mfaExpired')
  if (raw.includes('token') || raw.includes('expired')) return tr('accountLinkExpired')
  if (raw.includes('invalid') || raw.includes('authenticate') || raw.includes('password')) return tr('invalidCredentials')
  if (raw.includes('registration') || raw.includes('forbidden')) return tr('registrationDisabled')
  if (raw.includes('smtp') || raw.includes('mail') || raw.includes('email')) return tr('otpRequestFailed')
  return tr('authenticationUnavailable')
}

function responseMfaId(reason: unknown) {
  const value = reason as { response?: { mfaId?: string; data?: { mfaId?: string } }; data?: { mfaId?: string } }
  return value?.response?.mfaId || value?.response?.data?.mfaId || value?.data?.mfaId || ''
}

async function finish() {
  await router.replace(String(route.query.redirect || '/'))
}

function backToPasswordLogin() {
  mode.value = 'login'
  code.value = ''
  otpId.value = ''
  mfaId.value = ''
  captchaToken.value = ''
  error.value = ''
}

async function requestStandaloneOtp() {
  busy.value = true
  error.value = ''
  try {
    const token = await resolveCaptchaToken()
    const result = await auth.requestOTP(email.value, token)
    otpId.value = result.otpId
    captchaToken.value = ''
    mode.value = 'otp-verify'
  } catch (reason) {
    error.value = authError(reason)
  } finally {
    resetCaptcha()
    busy.value = false
  }
}

async function requestMfaOtp() {
  if (!mfaId.value) return
  await nextTick()
  const token = await resolveCaptchaToken()
  const result = await auth.requestMFAOTP(mfaId.value, token)
  otpId.value = result.otpId
  captchaToken.value = ''
}

async function startMfaFlow(id: string) {
  mfaId.value = id
  otpId.value = ''
  code.value = ''
  mode.value = 'mfa-verify'
  await requestMfaOtp()
}

async function passwordLogin() {
  try {
    const token = await resolveCaptchaToken()
    await auth.login(email.value, password.value, token)
    await finish()
  } catch (reason) {
    const id = responseMfaId(reason)
    if (!id) throw reason
    await startMfaFlow(id)
  }
}

async function verifyCode() {
  busy.value = true
  error.value = ''
  try {
    await auth.loginOTP(otpId.value, code.value, mode.value === 'mfa-verify' ? mfaId.value : '')
    await finish()
  } catch (reason) {
    error.value = mode.value === 'otp-verify' && responseMfaId(reason) ? tr('otpMfaMustUseFirstFactor') : authError(reason)
  } finally {
    busy.value = false
  }
}

async function requestPasswordReset() {
  busy.value = true
  error.value = ''
  notice.value = ''
  try {
    const token = await resolveCaptchaToken()
    await auth.requestPasswordReset(email.value.trim(), token)
    notice.value = tr('resetPasswordSent')
  } catch (reason) {
    error.value = authError(reason)
  } finally {
    resetCaptcha()
    busy.value = false
  }
}

async function completePasswordReset() {
  busy.value = true
  error.value = ''
  try {
    await auth.confirmPasswordReset(resetToken.value, resetPassword.value, resetPasswordConfirm.value)
    resetToken.value = ''
    await router.replace({ name: 'auth', query: { notice: 'password-reset' } })
  } catch (reason) {
    error.value = authError(reason)
  } finally {
    busy.value = false
  }
}

async function completeEmailChange() {
  busy.value = true
  error.value = ''
  try {
    await auth.confirmEmailChange(resetToken.value, emailChangePassword.value)
    resetToken.value = ''
    await router.replace({ name: 'auth', query: { notice: 'email-changed' } })
  } catch (reason) {
    error.value = authError(reason)
  } finally {
    busy.value = false
  }
}

async function submit() {
  if (mode.value === 'otp-request') return requestStandaloneOtp()
  if (mode.value === 'otp-verify' || mode.value === 'mfa-verify') return verifyCode()

  busy.value = true
  error.value = ''
  registered.value = false
  try {
    if (mode.value === 'login') await passwordLogin()
    else if (mode.value === 'register') {
      const token = await resolveCaptchaToken()
      await auth.register({ email: email.value, password: password.value, name: name.value, captchaToken: token })
      registered.value = true
      mode.value = 'login'
      password.value = ''
    } else if (mode.value === 'reset-request') await requestPasswordReset()
    else if (mode.value === 'reset-confirm') await completePasswordReset()
    else if (mode.value === 'email-change-confirm') await completeEmailChange()
  } catch (reason) {
    error.value = authError(reason)
  } finally {
    resetCaptcha()
    busy.value = false
  }
}

async function resend() {
  busy.value = true
  error.value = ''
  try {
    if (mode.value === 'mfa-verify') await requestMfaOtp()
    else await requestStandaloneOtp()
  } catch (reason) {
    error.value = authError(reason)
  } finally {
    resetCaptcha()
    busy.value = false
  }
}

async function oauth(provider: string) {
  busy.value = true
  error.value = ''
  try {
    await auth.loginOAuth(provider)
    await finish()
  } catch (reason) {
    const id = responseMfaId(reason)
    if (id) {
      try {
        await startMfaFlow(id)
      } catch (mfaReason) {
        error.value = authError(mfaReason)
      }
    } else error.value = authError(reason)
  } finally {
    resetCaptcha()
    busy.value = false
  }
}

function removeFragmentToken() {
  const token = window.location.hash.startsWith('#') ? window.location.hash.slice(1) : ''
  if (token) window.history.replaceState(window.history.state, '', window.location.pathname + window.location.search)
  return token
}

function syncRouteMode() {
  error.value = ''
  notice.value = route.query.notice === 'password-reset'
    ? tr('resetCompleted')
    : route.query.notice === 'email-changed'
      ? tr('emailChangeCompleted')
      : ''

  if (route.path === '/auth/reset-password') {
    resetToken.value = removeFragmentToken()
    mode.value = route.query.recovery === 'request' || !resetToken.value ? 'reset-request' : 'reset-confirm'
    if (!resetToken.value && route.query.recovery !== 'request') notice.value = tr('accountLinkExpired')
    return
  }
  if (route.path === '/auth/confirm-email-change') {
    resetToken.value = removeFragmentToken()
    mode.value = 'email-change-confirm'
    return
  }

  resetToken.value = ''
  resetPassword.value = ''
  resetPasswordConfirm.value = ''
  emailChangePassword.value = ''
  mode.value = 'login'
}

watch(mode, async value => {
  if (value !== "reset-request") return
  await nextTick()
  authCardRef.value?.querySelector<HTMLInputElement>("form input[type=email]")?.focus()
})
watch(() => route.fullPath, syncRouteMode, { immediate: true })

onMounted(async () => {
  try {
    const status = await setup.refresh()
    if (!status.initialized) {
      await router.replace('/setup')
      return
    }
    providers.value = await auth.oauthProviders()
    methods.value = await (await import('../pocketbase')).pb.collection('users').listAuthMethods()
  } catch {
    providers.value = []
  }
})

watch(canPasswordRegistration, value => {
  if (!value && mode.value === 'register') mode.value = 'login'
})
</script>

<template>
  <main class="auth-page">
    <div class="auth-controls"><ThemeSwitcher /><LanguageSwitcher /></div>
    <section class="auth-story">
      <Logo class="hero-brand" variant="lockup" size="md" />
      <p class="eyebrow">{{ tr('sharedMoneyClearly') }}</p>
      <h1>{{ tr('sharedFinance') }}</h1>
      <p>{{ tr('sharedFinanceDesc') }}</p>
    </section>
    <section class="auth-panel">
      <div ref="authCardRef" class="auth-card">
        <div v-if="canPasswordRegistration && (mode === 'login' || mode === 'register')" class="segmented">
          <button type="button" :aria-pressed="mode === 'login'" :class="{ active: mode === 'login' }" @click="mode = 'login'">{{ tr('login') }}</button>
          <button type="button" :aria-pressed="mode === 'register'" :class="{ active: mode === 'register' }" @click="mode = 'register'">{{ tr('register') }}</button>
        </div>

        <button v-if="mode === 'otp-request' || isCodeMode" type="button" class="back-auth" @click="backToPasswordLogin">← {{ tr('backToPasswordLogin') }}</button>
        <button v-else-if="mode === 'reset-request'" type="button" class="back-auth" @click="mode = 'login'; error = ''; notice = ''">← {{ tr('backToPasswordLogin') }}</button>
        <h2>{{ mode === 'register' ? tr('startSubFlow') : mode === 'mfa-verify' ? tr('mfaVerification') : mode === 'otp-request' ? tr('emailOtpRequestTitle') : mode === 'otp-verify' ? tr('emailOtpVerificationTitle') : mode === 'reset-request' ? tr('forgotPassword') : mode === 'reset-confirm' ? tr('chooseNewPassword') : mode === 'email-change-confirm' ? tr('confirmEmailChange') : tr('welcomeBack') }}</h2>
        <p v-if="mode === 'otp-request' || mode === 'otp-verify' || mode === 'mfa-verify'" class="auth-step-description">{{ mode === 'mfa-verify' ? tr('mfaVerificationDesc') : mode === 'otp-request' ? tr('emailOtpRequestDesc') : tr('emailOtpVerificationDesc') }}</p>

        <form @submit.prevent="submit">
          <BaseInput v-if="mode === 'register'" v-model="name" :label="tr('displayName')" required autocomplete="name" />
          <BaseInput v-if="mode === 'login' || mode === 'register' || mode === 'otp-request' || mode === 'otp-verify' || mode === 'reset-request'" v-model="email" :label="tr('email')" type="email" required autocomplete="email" :disabled="mode === 'otp-verify'" />
          <PasswordField v-if="mode === 'login' || mode === 'register'" v-model="password" :label="tr('password')" :minlength="8" required :autocomplete="mode === 'login' ? 'current-password' : 'new-password'" />
          <template v-if="mode === 'reset-confirm' && resetToken">
            <PasswordField v-model="resetPassword" :label="tr('newPassword')" :minlength="8" required autocomplete="new-password" />
            <PasswordField v-model="resetPasswordConfirm" :label="tr('confirmNewPassword')" :minlength="8" required autocomplete="new-password" />
          </template>
          <PasswordField v-if="mode === 'email-change-confirm' && resetToken" v-model="emailChangePassword" :label="tr('currentPassword')" required autocomplete="current-password" />
          <CaptchaChallenge v-if="captchaEnabled && (mode === 'login' || mode === 'register' || mode === 'reset-request' || mode === 'otp-request' || mode === 'mfa-verify')" ref="captchaRef" v-model="captchaToken" :flow="captchaFlowKey()" :trigger="captchaConfig?.trigger || 'load'" :mode="captchaConfig?.mode || 'interactive'" />
          <BaseInput v-if="isCodeMode" v-model="code" :label="tr('verificationCode')" required autocomplete="one-time-code" inputmode="numeric" :maxlength="8" />

          <p v-if="registered" class="notice success" role="status">{{ tr('verificationSent') }}</p>
          <p v-if="notice" class="notice success" role="status">{{ notice }}</p>
          <p v-if="mode === 'reset-request' && !setup.status.passwordResetAvailable" class="notice" role="status">{{ tr('passwordResetUnavailable') }}</p>
          <p v-if="(mode === 'reset-confirm' || mode === 'email-change-confirm') && !resetToken" class="notice" role="alert">{{ tr('accountLinkExpired') }}</p>
          <p v-if="error" class="form-error" role="alert">{{ error }}</p>

          <button v-if="mode !== 'reset-request' || setup.status.passwordResetAvailable" class="primary wide" :disabled="busy || ((mode === 'reset-confirm' || mode === 'email-change-confirm') && !resetToken)" type="submit">
            {{ busy ? tr('processing') : mode === 'login' ? tr('login') : mode === 'register' ? tr('register') : mode === 'otp-request' || mode === 'reset-request' ? tr('sendVerificationCode') : mode === 'reset-confirm' ? tr('setNewPassword') : mode === 'email-change-confirm' ? tr('confirmEmailChange') : tr('verify') }}
          </button>
        </form>

        <div v-if="mode === 'login'" class="auth-actions">
          <button v-if="setup.status.passwordResetAvailable" type="button" class="text-button" :disabled="busy" @click="mode = 'reset-request'; error = ''; notice = ''">{{ tr('forgotPassword') }}</button>
          <button v-if="methods.otp?.enabled" type="button" class="ghost wide" :disabled="busy" @click="mode = 'otp-request'; code = ''; otpId = ''; error = ''">{{ tr('emailOtpLogin') }}</button>
        </div>
        <div v-if="isCodeMode" class="auth-actions">
          <button type="button" class="ghost" :disabled="busy" @click="resend">{{ tr('resendCode') }}</button>
          <p v-if="mode === 'mfa-verify' && captchaEnabled" class="field-help">{{ tr('otpResendCaptchaHelp') }}</p>
        </div>
        <div v-if="mode === 'reset-confirm' && (!resetToken || error)" class="auth-actions"><RouterLink class="text-button" :to="{ path: '/auth/reset-password', query: { recovery: 'request' } }">{{ tr('requestNewResetLink') }}</RouterLink></div>
        <div v-if="mode === 'email-change-confirm' && (!resetToken || error)" class="auth-actions"><RouterLink class="text-button" to="/profile">{{ tr('requestNewEmailChange') }}</RouterLink></div>
        <div v-if="mode === 'login' && providers.length" class="oauth-options oauth-options--providers"><span>{{ tr('or') }}</span><OAuthProviderList :providers="providers" :busy="busy" @select="oauth" /></div>
      </div>
    </section>
  </main>
</template>

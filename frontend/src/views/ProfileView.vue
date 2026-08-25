<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { ProviderLinkedElsewhereError, useAuthStore } from '../stores/auth'
import { useSetupStore } from '../stores/setup'
import { useI18n } from '../i18n'
import { useTheme } from '../theme'
import TimezoneSelect from '../components/TimezoneSelect.vue'
import CurrencySelect from '../components/CurrencySelect.vue'
import { useWorkspaceStore } from '../stores/workspace'
import { useToastStore } from '../stores/toast'
import CategoryManagement from '../components/CategoryManagement.vue'
import { pb } from '../pocketbase'
import { ApiClient } from '../api/client'
import type { Contact } from '../api/types'
import CaptchaChallenge from '../components/CaptchaChallenge.vue'
const auth = useAuthStore(), setup = useSetupStore(), workspace=useWorkspaceStore(), toast = useToastStore(), saved = ref(false), resetSent = ref(false), resetBusy = ref(false), resetCaptchaToken = ref(''), { t, tr } = useI18n(), { preference, setTheme } = useTheme()
const form = reactive({ name: String(auth.record?.name || ''), timezone: String(auth.record?.timezone || Intl.DateTimeFormat().resolvedOptions().timeZone), default_currency:String(auth.record?.defaultCurrency||'TWD') })
async function submit() { await auth.updateProfile(form); saved.value = true; setTimeout(() => saved.value = false, 1800) }
const passwordResetCaptcha = computed(()=>setup.status.captchaFlows?.passwordReset)
const passwordResetCaptchaEnabled = computed(()=>!!passwordResetCaptcha.value?.enabled)
const resetCaptchaRef = ref<{ solve: () => Promise<string>; reset: () => void } | null>(null)
async function resetPassword() {
  if (!auth.record?.email) return
  resetBusy.value = true
  try {
    const token = passwordResetCaptchaEnabled.value ? (passwordResetCaptcha.value?.trigger === 'submit' ? await (resetCaptchaRef.value?.solve() || Promise.resolve('')) : resetCaptchaToken.value) : ''
    await auth.requestPasswordReset(String(auth.record.email), token)
    resetSent.value = true
  } finally {
    resetCaptchaToken.value = ''
    resetCaptchaRef.value?.reset()
    resetBusy.value = false
  }
}

const contactClient = new ApiClient(() => pb.authStore.token, () => {})
const contacts = ref<Contact[]>([])
const contactForm = reactive({ id: '', name: '', email: '' })
const contactBusy = ref(false)
const contactSearch = ref('')
const filteredContacts = computed(() => contacts.value.filter(contact => !contactSearch.value || (contact.name + contact.email).toLowerCase().includes(contactSearch.value.toLowerCase())))
async function loadContacts() { try { contacts.value = (await contactClient.get<Contact[]>('/personal/contacts')).data } catch { contacts.value = [] } }
function editContact(contact: Contact) { Object.assign(contactForm, { id: contact.id, name: contact.name, email: contact.email }) }
function resetContact() { Object.assign(contactForm, { id: '', name: '', email: '' }) }
async function saveContact() { contactBusy.value = true; try { const result = contactForm.id ? await contactClient.patch<Contact>('/personal/contacts/' + contactForm.id, { name: contactForm.name, email: contactForm.email }) : await contactClient.post<Contact>('/personal/contacts', { name: contactForm.name, email: contactForm.email }); const index = contacts.value.findIndex(value => value.id === result.data.id); if (index >= 0) contacts.value[index] = result.data; else contacts.value.push(result.data); resetContact() } finally { contactBusy.value = false } }
async function deleteContact(contact: Contact) { if (!confirm(tr('deleteContactConfirm'))) return; await contactClient.delete('/personal/contacts/' + contact.id); contacts.value = contacts.value.filter(value => value.id !== contact.id) }
const providers = ref<{ name: string; displayName: string }[]>([])
const linkedProviders = ref<Set<string>>(new Set())
const providerBusy = reactive<Record<string, boolean>>({})
async function loadProviders() {
  try {
    providers.value = await auth.oauthProviders()
    linkedProviders.value = new Set((await auth.listLinkedProviders()).map(p => p.provider))
  } catch { /* provider list is a nice-to-have; leave the card empty on failure */ }
}
onMounted(async()=>{ try { if (!setup.ready) await setup.refresh() } finally { await loadProviders(); await loadContacts() } })
async function toggleProvider(name: string) {
  providerBusy[name] = true
  const wasLinked = linkedProviders.value.has(name)
  try {
    if (wasLinked) {
      await auth.unlinkProvider(name)
      linkedProviders.value.delete(name)
      toast.push('success', t.value.providerUnlinked)
    } else {
      await auth.linkOAuth(name)
      linkedProviders.value.add(name)
      toast.push('success', t.value.providerLinked)
    }
  } catch (reason) {
    toast.push('error', reason instanceof ProviderLinkedElsewhereError ? t.value.providerLinkedElsewhere : (wasLinked ? t.value.providerUnlinkFailed : t.value.providerLinkFailed))
  } finally {
    providerBusy[name] = false
  }
}
</script>
<template>
    <section class="page profile-page">
        <div class="page-heading">
            <div>
                <p class="eyebrow">{{t.profile}}</p>
                <h1>{{ t.profile }}</h1>
                <p>{{ t.profileDesc }}</p>
            </div>
        </div>
        <div class="profile-layout">
            <form class="card form-card profile-account" @submit.prevent="submit">
                <h2>{{t.accountInfo}}</h2><label>{{ t.email }}<input :value="auth.record?.email"
                        disabled></label><label>{{ t.displayName }}<input v-model="form.name"
                        required></label><label>{{ t.timezone }}
                    <TimezoneSelect v-model="form.timezone" />
                </label><label>{{t.currency}}<CurrencySelect v-model="form.default_currency" :currencies="workspace.currencies"/></label><button class="primary">{{ t.save }}</button><span v-if="saved" class="success">{{ t.saved }}</span>
            </form>
            <section class="card form-card profile-appearance">
                <div>
                    <p class="eyebrow">{{t.appearance}}</p>
                    <h2>{{ t.theme }}</h2>
                    <p class="setting-description">{{ t.themeDesc }}</p>
                </div>
                <div class="theme-options"><button type="button" :class="{ selected: preference === 'system' }"
                        @click="setTheme('system')"><span
                            class="theme-preview system-preview"></span><strong>{{ t.systemTheme }}</strong></button><button
                        type="button" :class="{ selected: preference === 'light' }" @click="setTheme('light')"><span
                            class="theme-preview light-preview"></span><strong>{{ t.lightTheme }}</strong></button><button
                        type="button" :class="{ selected: preference === 'dark' }" @click="setTheme('dark')"><span
                            class="theme-preview dark-preview"></span><strong>{{ t.darkTheme }}</strong></button></div>
            </section>
            <section class="card form-card profile-security">
                <div><p class="eyebrow">{{t.loginSecurity}}</p><h2>{{ t.loginSecurity }}</h2><p class="setting-description">{{ t.passwordResetUnavailable }}</p></div>
                <CaptchaChallenge v-if="passwordResetCaptchaEnabled" ref="resetCaptchaRef" v-model="resetCaptchaToken" flow="passwordReset" :trigger="passwordResetCaptcha?.trigger || 'load'" :mode="passwordResetCaptcha?.mode || 'interactive'" />
                <button type="button" class="ghost" :disabled="!auth.record?.email || resetBusy" @click="resetPassword">{{ resetBusy ? t.processing : t.resetPassword }}</button>
                <p v-if="resetSent" class="success">{{ t.resetPasswordSent }}</p>
            </section>
            <section class="card form-card profile-categories"><CategoryManagement scope="personal" /></section>            <section class="card form-card profile-contacts">
                <div><p class="eyebrow">{{ tr('contacts') }}</p><h2>{{ tr('contactManagement') }}</h2><p class="setting-description">{{ tr('contactManagementDesc') }}</p></div>
                <div class="contact-form"><input v-model="contactForm.name" :placeholder="tr('contactName')"><input v-model="contactForm.email" type="email" :placeholder="tr('contactEmail')"><div class="form-actions"><button class="primary" :disabled="contactBusy || !contactForm.name || !contactForm.email" @click="saveContact">{{ tr('contactSave') }}</button><button v-if="contactForm.id" type="button" class="ghost" @click="resetContact">{{ tr('cancel') }}</button></div></div>
                <input v-model="contactSearch" :placeholder="tr('contactSearch')">
                <div class="rows"><div v-for="contact in filteredContacts" :key="contact.id" class="row"><div class="grow"><strong>{{ contact.name }}</strong><small>{{ contact.email }}</small></div><button type="button" class="ghost" @click="editContact(contact)">{{ tr('edit') }}</button><button type="button" class="ghost danger-text" @click="deleteContact(contact)">{{ tr('deleteContactAction') }}</button></div><p v-if="!filteredContacts.length" class="empty-inline">{{ tr('noContacts') }}</p></div>
            </section>
            <section v-if="providers.length" class="card form-card profile-providers">
                <div><p class="eyebrow">{{t.connectedProviders}}</p><h2>{{ t.connectedProviders }}</h2><p class="setting-description">{{ t.connectedProvidersDesc }}</p></div>
                <div class="rows">
                    <div v-for="provider in providers" :key="provider.name" class="row">
                        <div class="grow"><strong>{{ provider.displayName || provider.name }}</strong></div>
                        <span class="pill">{{ linkedProviders.has(provider.name) ? t.providerConnected : t.providerNotConnected }}</span>
                        <button type="button" class="ghost" :disabled="providerBusy[provider.name]" @click="toggleProvider(provider.name)">{{ linkedProviders.has(provider.name) ? t.disconnectProvider : t.connectProvider }}</button>
                    </div>
                </div>
            </section>
            <section class="card form-card profile-account-actions">
                <div><p class="eyebrow">{{t.accountActions}}</p><h2>{{ t.accountActions }}</h2><p class="setting-description">{{ t.accountActionsDesc }}</p></div>
                <div class="form-actions">
                    <RouterLink v-if="auth.canAdminister" class="ghost" :to="{ name: 'admin' }">{{ tr('systemAdministration') }}</RouterLink>
                    <button type="button" class="ghost danger-text" @click="auth.logout">{{ t.logout }}</button>
                </div>
            </section>
        </div>
    </section>
</template>

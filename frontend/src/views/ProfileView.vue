<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useAuthStore } from '../stores/auth'
import { useI18n } from '../i18n'
import { useTheme } from '../theme'
import TimezoneSelect from '../components/TimezoneSelect.vue'
import CurrencySelect from '../components/CurrencySelect.vue'
import { useWorkspaceStore } from '../stores/workspace'
import CategoryManagement from '../components/CategoryManagement.vue'
const auth = useAuthStore(), workspace=useWorkspaceStore(), saved = ref(false), resetSent = ref(false), resetBusy = ref(false), { t } = useI18n(), { preference, setTheme } = useTheme()
const form = reactive({ name: String(auth.record?.name || ''), timezone: String(auth.record?.timezone || Intl.DateTimeFormat().resolvedOptions().timeZone), default_currency:String(auth.record?.defaultCurrency||'TWD') })
async function submit() { await auth.updateProfile(form); saved.value = true; setTimeout(() => saved.value = false, 1800) }
async function resetPassword() { if (!auth.record?.email) return; resetBusy.value = true; try { await auth.requestPasswordReset(String(auth.record.email)); resetSent.value = true } finally { resetBusy.value = false } }
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
                <button type="button" class="ghost" :disabled="!auth.record?.email || resetBusy" @click="resetPassword">{{ resetBusy ? t.processing : t.resetPassword }}</button>
                <p v-if="resetSent" class="success">{{ t.resetPasswordSent }}</p>
            </section>
            <section class="card form-card profile-categories"><CategoryManagement scope="personal" /></section>
        </div>
    </section>
</template>

<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useAuthStore } from '../stores/auth'
import { useI18n } from '../i18n'
import { useTheme } from '../theme'
import TimezoneSelect from '../components/TimezoneSelect.vue'
const auth = useAuthStore(), saved = ref(false), { t } = useI18n(), { preference, setTheme } = useTheme()
const form = reactive({ name: String(auth.record?.name || ''), timezone: String(auth.record?.timezone || Intl.DateTimeFormat().resolvedOptions().timeZone) })
async function submit() { await auth.updateProfile(form); saved.value = true; setTimeout(() => saved.value = false, 1800) }
</script>
<template>
    <section class="page narrow">
        <div class="page-heading">
            <div>
                <p class="eyebrow">PROFILE</p>
                <h1>{{ t.profile }}</h1>
                <p>{{ t.profileDesc }}</p>
            </div>
        </div>
        <div class="profile-layout">
            <form class="card form-card" @submit.prevent="submit">
                <h2>帳號資料</h2><label>{{ t.email }}<input :value="auth.record?.email"
                        disabled></label><label>{{ t.displayName }}<input v-model="form.name"
                        required></label><label>{{ t.timezone }}
                    <TimezoneSelect v-model="form.timezone" />
                </label><button class="primary">{{ t.save }}</button><span v-if="saved" class="success">{{ t.saved }}</span>
            </form>
            <section class="card form-card appearance-card">
                <div>
                    <p class="eyebrow">APPEARANCE</p>
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
        </div>
    </section>
</template>

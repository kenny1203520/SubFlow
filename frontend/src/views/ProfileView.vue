<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { useAuthStore } from '../stores/auth'
import { useI18n } from '../i18n'
import { useTheme } from '../theme'
import TimezoneSelect from '../components/TimezoneSelect.vue'
import { useWorkspaceStore } from '../stores/workspace'
const auth = useAuthStore(), workspace=useWorkspaceStore(), saved = ref(false), categoryName=ref(''), { t } = useI18n(), { preference, setTheme } = useTheme()
const form = reactive({ name: String(auth.record?.name || ''), timezone: String(auth.record?.timezone || Intl.DateTimeFormat().resolvedOptions().timeZone) })
async function submit() { await auth.updateProfile(form); saved.value = true; setTimeout(() => saved.value = false, 1800) }
async function addCategory(){if(!categoryName.value.trim())return;await workspace.createCategory('personal',categoryName.value.trim());categoryName.value=''}
onMounted(()=>void workspace.loadCategories('personal'))
</script>
<template>
    <section class="page narrow">
        <div class="page-heading">
            <div>
                <p class="eyebrow">{{t.profile}}</p>
                <h1>{{ t.profile }}</h1>
                <p>{{ t.profileDesc }}</p>
            </div>
        </div>
        <div class="profile-layout">
            <form class="card form-card" @submit.prevent="submit">
                <h2>{{t.accountInfo}}</h2><label>{{ t.email }}<input :value="auth.record?.email"
                        disabled></label><label>{{ t.displayName }}<input v-model="form.name"
                        required></label><label>{{ t.timezone }}
                    <TimezoneSelect v-model="form.timezone" />
                </label><button class="primary">{{ t.save }}</button><span v-if="saved" class="success">{{ t.saved }}</span>
            </form>
            <section class="card form-card appearance-card">
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
            <section class="card form-card"><h2>{{t.category}}</h2><div class="inline-create"><input v-model="categoryName" :placeholder="t.addCategory"><button type="button" class="primary" @click="addCategory">{{t.add}}</button></div><div class="data-list"><article v-for="item in workspace.categories.filter(v=>v.scope==='personal')" :key="item.id" class="data-row"><span class="grow">{{item.customName}}</span><button class="ghost danger-text" @click="workspace.archiveCategory(item.id)">{{t.remove}}</button></article></div></section>
        </div>
    </section>
</template>

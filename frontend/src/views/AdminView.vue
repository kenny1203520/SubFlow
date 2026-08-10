<script setup lang="ts">
import './admin.css'
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { ApiClient } from '../api/client'
import type { AccessRole, AuditLog, User } from '../api/types'
import { useAuthStore } from '../stores/auth'
import { useWorkspaceStore } from '../stores/workspace'
import { useI18n } from '../i18n'
import { systemPermissionText } from '../locales/admin'
import CurrencySelect from '../components/CurrencySelect.vue'
import TimezoneSelect from '../components/TimezoneSelect.vue'
import BaseInput from '../components/BaseInput.vue'
import PasswordField from '../components/PasswordField.vue'
import RoleSelect from '../components/RoleSelect.vue'
import BaseCombobox from '../components/BaseCombobox.vue'
import { auditPresentation } from '../utils/audit'

const auth = useAuthStore()
const workspace = useWorkspaceStore()
const route = useRoute()
const { tr, formatDate, locale } = useI18n()
const api = new ApiClient(() => auth.token, auth.logout)
const roles = ref<AccessRole[]>([])
const users = ref<User[]>([])
const logs = ref<AuditLog[]>([])
const error = ref('')
const saved = ref(false)
const loading = ref(false)
const query = ref('')
const settings = ref({ initialized: true, siteName: 'SubFlow', defaultTimezone: 'UTC', defaultCurrency: 'TWD', allowRegistration: true, allowPasswordRegistration: true, allowOidcRegistration: true, captchaProvider: '', captchaSiteKey: '', captchaChallengeUrl: '', captchaVerifyUrl: '', captchaSecret: '', captchaConfigured: false })
const editRole = ref<AccessRole | null>(null)
const allPermissions = ['system.roles.manage', 'system.users.assign', 'system.audit.read', 'system.settings.manage']
const section = computed(() => String(route.params.section || 'overview'))
const roleCategories = computed(() => [...new Set(roles.value.map(role => role.category).filter((value): value is string => !!value))].sort())
const rolesByCategory = computed(() => roles.value.reduce<Record<string, AccessRole[]>>((groups, role) => { const category = role.category || tr('ungroupedRoles'); (groups[category] ||= []).push(role); return groups }, {}))
const can = (permission: string) => auth.permissions.includes('*') || auth.permissions.includes(permission)
const pbAdminUrl = computed(() => {
  const base = import.meta.env.VITE_BACKEND_URL || window.location.origin
  return new URL('/_/', base).toString()
})

function permissionInfo(permission: string) {
  return systemPermissionText[locale.value][permission as keyof typeof systemPermissionText['zh-TW']] ?? { title: permission, description: permission }
}
function auditInfo(log: AuditLog) { return auditPresentation(log, locale.value) }

async function loadRoles() { if (can('system.roles.manage')) roles.value = (await api.get<AccessRole[]>('/system/roles')).data }
async function loadUsers() { if (can('system.users.assign')) users.value = (await api.get<User[]>(`/system/users?perPage=50&q=${encodeURIComponent(query.value)}`)).data }
async function loadLogs() { if (can('system.audit.read')) logs.value = (await api.get<AuditLog[]>('/system/audit-logs?perPage=50')).data }
async function loadSettings() { if (can('system.settings.manage')) settings.value = (await api.get<typeof settings.value>('/system/settings')).data }
async function load() {
  loading.value = true
  error.value = ''
  try { await Promise.all([loadRoles(), loadUsers(), loadLogs(), loadSettings()]) } catch { error.value = tr('requestFailed') } finally { loading.value = false }
}
async function saveSettings() {
  try {
    settings.value = (await api.patch<typeof settings.value>('/system/settings', settings.value)).data
    await auth.refreshAccess()
    saved.value = true
    setTimeout(() => { saved.value = false }, 1800)
  } catch { error.value = tr('requestFailed') }
}
function startRole(role?: AccessRole) {
  editRole.value = role ? { ...role, permissions: [...role.permissions] } : { id: '', scope: 'system', name: '', category: '', key: '', permissions: [], protected: false, createdAt: '', updatedAt: '' }
}
async function saveRole() {
  if (!editRole.value) return
  try {
    const value = editRole.value
    if (value.id) await api.patch(`/system/roles/${value.id}`, value)
    else await api.post('/system/roles', { name: value.name, category: value.category, permissions: value.permissions })
    editRole.value = null
    await loadRoles()
  } catch { error.value = tr('requestFailed') }
}
async function deleteRole(role: AccessRole) {
  if (!confirm(tr('deleteGroupConfirm', { name: role.name }))) return
  try { await api.delete(`/system/roles/${role.id}`); await loadRoles() } catch { error.value = tr('requestFailed') }
}
async function assign(user: User, roleId: string) {
  try {
    await api.request(`/system/users/${user.id}/role`, { method: 'PUT', body: JSON.stringify({ roleId }) })
    await loadUsers()
    if (user.id === auth.record?.id) await auth.refreshAccess()
  } catch { error.value = tr('requestFailed') }
}

watch(section, () => void load())
onMounted(() => void load())
</script>

<template>
  <section class="page admin-page">
    <div class="admin-header">
      <div>
        <p class="eyebrow">{{ tr('systemAdministration') }}</p>
        <h1>{{ tr('systemAdministration') }}</h1>
        <p>{{ tr('accountSettings') }}</p>
      </div>
      <a class="admin-console-link" :href="pbAdminUrl" target="_blank" rel="noopener">
        {{ tr('openPocketBase') }} <span aria-hidden="true">↗</span>
      </a>
    </div>
    <nav class="admin-tabs">
      <RouterLink :to="{ name: 'admin' }">{{ tr('adminOverview') }}</RouterLink>
      <RouterLink v-if="can('system.settings.manage')" :to="{ name: 'admin-section', params: { section: 'settings' } }">{{ tr('settings') }}</RouterLink>
      <RouterLink v-if="can('system.users.assign')" :to="{ name: 'admin-section', params: { section: 'users' } }">{{ tr('userManagement') }}</RouterLink>
      <RouterLink v-if="can('system.roles.manage')" :to="{ name: 'admin-section', params: { section: 'roles' } }">{{ tr('roleManagement') }}</RouterLink>
      <RouterLink v-if="can('system.audit.read')" :to="{ name: 'admin-section', params: { section: 'audit' } }">{{ tr('auditLogs') }}</RouterLink>
    </nav>
    <p v-if="error" class="notice danger">{{ error }}</p>
    <p v-if="loading" class="notice">{{ tr('processing') }}</p>

    <template v-if="section === 'overview'">
      <div class="admin-grid">
        <section class="card"><h2>{{ tr('settings') }}</h2><p>{{ tr('siteName') }} · {{ settings.siteName }}</p><RouterLink v-if="can('system.settings.manage')" class="ghost" :to="{ name: 'admin-section', params: { section: 'settings' } }">{{ tr('edit') }}</RouterLink></section>
        <section class="card"><h2>{{ tr('userManagement') }}</h2><p>{{ tr('records', { count: users.length }) }}</p><RouterLink v-if="can('system.users.assign')" class="ghost" :to="{ name: 'admin-section', params: { section: 'users' } }">{{ tr('members') }}</RouterLink></section>
        <section class="card"><h2>{{ tr('roleManagement') }}</h2><p>{{ tr('records', { count: roles.length }) }}</p><RouterLink v-if="can('system.roles.manage')" class="ghost" :to="{ name: 'admin-section', params: { section: 'roles' } }">{{ tr('settings') }}</RouterLink></section>
      </div>
    </template>

    <form v-else-if="section === 'settings' && can('system.settings.manage')" class="card form-card admin-form" @submit.prevent="saveSettings">
      <h2>{{ tr('settings') }}</h2>
      <BaseInput v-model="settings.siteName" :label="tr('siteName')" required :maxlength="120" />
      <label>{{ tr('timezone') }}<TimezoneSelect v-model="settings.defaultTimezone" /></label>
      <label>{{ tr('currency') }}<CurrencySelect v-model="settings.defaultCurrency" :currencies="workspace.currencies" /></label>
      <fieldset class="settings-section"><legend>{{ tr('loginSecurity') }}</legend>
        <label class="check"><input v-model="settings.allowPasswordRegistration" type="checkbox"><span>{{ tr('allowPasswordRegistration') }}</span></label>
        <label class="check"><input v-model="settings.allowOidcRegistration" type="checkbox"><span>{{ tr('allowOidcRegistration') }}</span></label>
      </fieldset>
      <fieldset class="settings-section"><legend>{{ tr('captcha') }}</legend>
        <BaseCombobox v-model="settings.captchaProvider" :label="tr('captchaProvider')" :options="[{value:'',label:tr('disabled')},{value:'recaptcha',label:'Google reCAPTCHA'},{value:'turnstile',label:'Cloudflare Turnstile'},{value:'hcaptcha',label:'hCaptcha'},{value:'altcha_community',label:'ALTCHA Community'},{value:'altcha_sentinel',label:'ALTCHA Sentinel'}]" />
        <BaseInput v-if="settings.captchaProvider&&settings.captchaProvider!=='altcha_community'&&settings.captchaProvider!=='altcha_sentinel'" v-model="settings.captchaSiteKey" :label="tr('captchaSiteKey')" />
        <BaseInput v-if="settings.captchaProvider==='altcha_sentinel'" v-model="settings.captchaChallengeUrl" label="Sentinel challenge URL" help="Public ALTCHA Sentinel challenge endpoint." />
        <BaseInput v-if="settings.captchaProvider==='altcha_sentinel'" v-model="settings.captchaVerifyUrl" label="Sentinel verification URL" help="Server-side /v1/verify/signature endpoint." />
        <PasswordField v-if="settings.captchaProvider&&settings.captchaProvider!=='altcha_community'" v-model="settings.captchaSecret" :label="tr('captchaSecret')" autocomplete="off" :help="settings.captchaConfigured ? tr('captchaConfigured') : tr('captchaNotConfigured')" />
        <p v-else-if="settings.captchaProvider==='altcha_community'" class="field-help">ALTCHA Community signing secret is generated and encrypted by SubFlow.</p>
      </fieldset>
      <div class="form-actions"><button class="primary">{{ tr('saveChanges') }}</button><span v-if="saved" class="success">{{ tr('saved') }}</span></div>
    </form>

    <section v-else-if="section === 'users' && can('system.users.assign')" class="card admin-form">
      <div class="section-heading"><h2>{{ tr('userManagement') }}</h2><BaseInput v-model="query" :label="tr('search')" @update:model-value="loadUsers" /></div>
      <div class="data-list">
        <article v-for="user in users" :key="user.id" class="data-row">
          <div class="grow"><strong>{{ user.name || tr('unnamedMember') }}</strong><small>{{ user.email }}</small></div>
          <RoleSelect :model-value="user.systemRoleId || ''" :roles="roles" :label="tr('assignRole')" @update:model-value="assign(user, $event)" />
        </article>
        <p v-if="!users.length" class="empty-inline">{{ tr('noUsers') }}</p>
      </div>
    </section>

    <section v-else-if="section === 'roles' && can('system.roles.manage')" class="admin-stack">
      <section class="card admin-form">
        <div class="section-heading"><h2>{{ tr('roleManagement') }}</h2><button class="primary" @click="startRole()">{{ tr('add') }}</button></div>
        <div class="role-category-list">
          <section v-for="(categoryRoles, category) in rolesByCategory" :key="category" class="role-category">
          <h3>{{ category }}</h3>
          <article v-for="role in categoryRoles" :key="role.id" class="data-row">
            <div class="grow">
              <strong>{{ role.name }}</strong>
              <div v-if="role.permissions.length" class="permission-summary">
                <div v-for="permission in role.permissions" :key="permission" class="permission-display">
                  <strong>{{ permissionInfo(permission).title }}</strong><small>{{ permissionInfo(permission).description }}</small>
                </div>
              </div>
              <p v-else class="permission-empty">{{ tr('noSummary') }}</p>
            </div>
            <span v-if="role.protected" class="pill">{{ tr('protectedRole') }}</span>
            <button v-else class="ghost" @click="startRole(role)">{{ tr('edit') }}</button>
            <button v-if="!role.protected" class="danger-text" @click="deleteRole(role)">{{ tr('delete') }}</button>
          </article>
          </section>
        </div>
      </section>
      <form v-if="editRole" class="card admin-form role-editor" @submit.prevent="saveRole">
        <section class="form-section">
          <h2>{{ editRole.id ? tr('editRole') : tr('createRole') }}</h2>
          <BaseInput v-model="editRole.name" :label="tr('roleName')" required />
          <BaseCombobox v-model="editRole.category" :options="roleCategories" :label="tr('roleGroup')" :placeholder="tr('ungroupedRoles')" :help="tr('roleGroupHelp')" />
        </section>
        <fieldset class="permission-options form-section">
          <legend>{{ tr('permissions') }}</legend>
          <label v-for="permission in allPermissions" :key="permission" class="check permission-option">
            <input v-model="editRole.permissions" type="checkbox" :value="permission">
            <span><strong>{{ permissionInfo(permission).title }}</strong><small>{{ permissionInfo(permission).description }}</small></span>
          </label>
        </fieldset>
        <div class="form-actions"><button class="primary">{{ tr('save') }}</button><button type="button" class="ghost" @click="editRole = null">{{ tr('cancel') }}</button></div>
      </form>
    </section>

    <section v-else-if="section === 'audit' && can('system.audit.read')" class="card admin-form">
      <h2>{{ tr('auditLogs') }}</h2>
      <div class="data-list"><article v-for="log in logs" :key="log.id" class="data-row"><div class="grow"><strong>{{ auditInfo(log).action }}</strong><small>{{ auditInfo(log).actor }} · {{ auditInfo(log).resource }} · {{ formatDate(log.createdAt) }}</small></div><span class="pill">{{ auditInfo(log).outcome }}</span></article><p v-if="!logs.length" class="empty-inline">{{ tr('noSummary') }}</p></div>
    </section>
  </section>
</template>

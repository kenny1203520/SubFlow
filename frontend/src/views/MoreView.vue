<script setup lang="ts">
import { RouterLink } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import { useI18n } from '../i18n'

const auth = useAuthStore()
const { tr } = useI18n()
const links = [
  { to:'/notifications', title:'notificationCenter', detail:'moreNotificationsDesc', icon:'N' },
  { to:'/personal/share', title:'share', detail:'moreShareDesc', icon:'S' },
  { to:'/profile', title:'accountSettings', detail:'moreProfileDesc', icon:'P' },
  { to:'/profile#profile-categories', title:'category', detail:'categoryManagementDesc', icon:'#' },
  { to:'/profile#profile-contacts', title:'contacts', detail:'contactManagementDesc', icon:'@' },
  { to:'/personal', title:'attachments', detail:'moreAttachmentsDesc', icon:'A' },
  { to:'/about', title:'about', detail:'moreAboutDesc', icon:'i' },
]
</script>

<template><section class="page more-page"><header class="page-heading"><div><p class="eyebrow">{{ tr('workspace') }}</p><h1>{{ tr('more') }}</h1><p>{{ tr('moreDesc') }}</p></div></header><nav class="more-grid" :aria-label="tr('more')"><RouterLink v-for="link in links" :key="link.to" class="card more-link" :to="link.to"><span class="more-icon" aria-hidden="true">{{ link.icon }}</span><span class="more-copy"><strong>{{ tr(link.title) }}</strong><small>{{ tr(link.detail) }}</small></span><span class="more-arrow" aria-hidden="true">›</span></RouterLink><RouterLink v-if="auth.canAdminister" class="card more-link" to="/admin"><span class="more-icon" aria-hidden="true">◇</span><span class="more-copy"><strong>{{ tr('systemAdministration') }}</strong><small>{{ tr('moreAdminDesc') }}</small></span><span class="more-arrow" aria-hidden="true">›</span></RouterLink></nav></section></template>

<style scoped>
.more-grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(min(100%,320px),1fr));gap:14px}.more-link{display:grid;grid-template-columns:42px minmax(0,1fr) 22px;align-items:center;gap:14px;color:var(--ink);text-decoration:none;min-height:96px;transition:background-color .18s,border-color .18s,box-shadow .18s}.more-link:hover{border-color:var(--brand);box-shadow:var(--shadow-lg)}.more-icon{width:40px;height:40px;display:grid;place-items:center;border-radius:13px;background:var(--brand-soft);color:var(--brand);font-size:20px;font-weight:800}.more-copy{display:grid;gap:5px;min-width:0}.more-copy strong{font-size:15px}.more-copy small{color:var(--muted);line-height:1.4}.more-arrow{justify-self:end;color:var(--brand);font-size:24px}
</style>

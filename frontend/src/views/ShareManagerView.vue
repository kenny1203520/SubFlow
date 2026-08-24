<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { pb } from '../pocketbase'
import { ApiClient } from '../api/client'
import type { Share, ShareAccessMode, ShareRangeMode } from '../api/types'
import { useI18n } from '../i18n'

const route = useRoute()
const client = new ApiClient(() => pb.authStore.token, () => {})
const { tr } = useI18n()
const groupId = computed(() => String(route.params.groupId || ''))
const isGroup = computed(() => !!groupId.value)
const shares = ref<Share[]>([])
const busy = ref(false)
const error = ref('')
const copied = ref('')
// undefined means the editor is closed; null means an intentionally new page.
const editing = ref<Share | null | undefined>(undefined)
const form = reactive({ name:'', accessMode:'link' as ShareAccessMode, password:'', viewerEmails:'', enabled:true, expiresAt:'', rangeMode:'all' as ShareRangeMode, rollingDays:30, startsOn:'', endsOn:'', showSummary:true, showExpenses:true, showSubscriptions:true, showSettlements:true, showIdentities:false, showNotes:false })
const base = computed(() => isGroup.value ? `/groups/${groupId.value}/shares` : '/personal/shares')

function reset(value?: Share) {
  editing.value = value ?? null
  Object.assign(form, { name:value?.name || '', accessMode:value?.accessMode || 'link', password:'', viewerEmails:value?.viewers?.map(item => item.email).join('\n') || '', enabled:value?.enabled ?? true, expiresAt:value?.expiresAt?.slice(0,10) || '', rangeMode:value?.rangeMode || 'all', rollingDays:value?.rollingDays || 30, startsOn:value?.startsOn?.slice(0,10) || '', endsOn:value?.endsOn?.slice(0,10) || '', showSummary:value?.showSummary ?? true, showExpenses:value?.showExpenses ?? true, showSubscriptions:value?.showSubscriptions ?? true, showSettlements:value?.showSettlements ?? true, showIdentities:value?.showIdentities ?? false, showNotes:value?.showNotes ?? false })
}
function payload() {
  return { ...form, expiresAt:form.expiresAt ? new Date(`${form.expiresAt}T23:59:59`).toISOString() : '', startsOn:form.startsOn ? new Date(`${form.startsOn}T00:00:00`).toISOString() : '', endsOn:form.endsOn ? new Date(`${form.endsOn}T00:00:00`).toISOString() : '', viewerEmails:form.viewerEmails.split(/[\n,]/).map(value => value.trim()).filter(Boolean) }
}
async function load() { busy.value=true; error.value=''; try { shares.value=(await client.get<Share[]>(base.value)).data } catch(reason) { error.value=reason instanceof Error ? reason.message : tr('requestFailed') } finally { busy.value=false } }
async function save() { busy.value=true; error.value=''; try { if (editing.value?.id) await client.patch<Share>(`${base.value}/${editing.value.id}`, payload()); else { const result=await client.post<Share & {url?:string}>(base.value,payload()); if (result.data.url) await copy(result.data.url) }; editing.value=undefined; await load() } catch(reason) { error.value=reason instanceof Error ? reason.message : tr('requestFailed') } finally { busy.value=false } }
async function remove(value: Share) { if (!confirm(tr('shareDeleteConfirm',{name:value.name}))) return; await client.delete(`${base.value}/${value.id}`); await load() }
async function rotate(value: Share) { const result=await client.post<Share & {url:string}>(`${base.value}/${value.id}/rotate`); await copy(result.data.url); await load() }
async function copy(value: string) { const url=new URL(value,window.location.origin).toString(); await navigator.clipboard?.writeText(url); copied.value=tr('shareLinkCopied',{url}); setTimeout(()=>copied.value='',3000) }
function status(value: Share) { return tr(value.enabled ? 'shareStatusEnabled' : 'shareStatusDisabled') }
function range(value: Share) { return tr(value.rangeMode === 'all' ? 'shareRangeAll' : value.rangeMode === 'rolling' ? 'shareRangeRolling' : 'shareRangeFixed') }
onMounted(load)
watch(groupId, load)
</script>

<template>
  <section class="page share-manager">
    <div class="page-heading"><div><p class="eyebrow">{{ tr(isGroup ? 'shareLedgerGroup' : 'shareLedgerPersonal') }}</p><h1>{{ tr('sharePages') }}</h1><p>{{ tr('sharePagesDesc') }}</p></div><button class="primary" @click="reset()">{{ tr('createShare') }}</button></div>
    <p v-if="error" class="notice danger">{{ error }}</p><p v-if="copied" class="notice success">{{ copied }}</p>
    <section v-if="editing !== undefined" class="card form-card"><h2>{{ editing?.id ? tr('editShare') : tr('newShare') }}</h2><form @submit.prevent="save"><label>{{ tr('shareName') }}<input v-model="form.name" required maxlength="120" :placeholder="tr('shareNamePlaceholder')"></label><label>{{ tr('shareAccess') }}<select v-model="form.accessMode"><option value="link">{{ tr('shareAccessLink') }}</option><option value="password">{{ tr('shareAccessPassword') }}</option><option value="accounts">{{ tr('shareAccessAccounts') }}</option></select></label><label v-if="form.accessMode==='password'">{{ editing?.id ? tr('sharePasswordNew') : tr('sharePassword') }}<input v-model="form.password" type="password" :required="!editing?.id" minlength="8"></label><label v-if="form.accessMode==='accounts'">{{ tr('shareViewerEmails') }}<textarea v-model="form.viewerEmails" rows="3" :placeholder="tr('shareViewerPlaceholder')"></textarea></label><label><input v-model="form.enabled" type="checkbox"> {{ tr('shareEnabled') }}</label><label>{{ tr('shareExpires') }}<input v-model="form.expiresAt" type="date"></label><label>{{ tr('shareRange') }}<select v-model="form.rangeMode"><option value="all">{{ tr('shareRangeAll') }}</option><option value="rolling">{{ tr('shareRangeRolling') }}</option><option value="fixed">{{ tr('shareRangeFixed') }}</option></select></label><label v-if="form.rangeMode==='rolling'">{{ tr('shareRecentDays') }}<select v-model.number="form.rollingDays"><option :value="30">30</option><option :value="90">90</option><option :value="365">365</option></select></label><div v-if="form.rangeMode==='fixed'" class="two"><label>{{ tr('shareFrom') }}<input v-model="form.startsOn" type="date"></label><label>{{ tr('shareTo') }}<input v-model="form.endsOn" type="date"></label></div><fieldset><legend>{{ tr('shareContent') }}</legend><label><input v-model="form.showSummary" type="checkbox"> {{ tr('shareSummary') }}</label><label><input v-model="form.showExpenses" type="checkbox"> {{ tr('shareExpenses') }}</label><label><input v-model="form.showSubscriptions" type="checkbox"> {{ tr('shareSubscriptions') }}</label><label v-if="isGroup"><input v-model="form.showSettlements" type="checkbox"> {{ tr('shareSettlements') }}</label><label><input v-model="form.showIdentities" type="checkbox"> {{ tr('shareIdentities') }}</label><label><input v-model="form.showNotes" type="checkbox"> {{ tr('shareNotes') }}</label></fieldset><div class="form-actions"><button class="primary" :disabled="busy">{{ tr('save') }}</button><button class="ghost" type="button" @click="editing=undefined">{{ tr('cancel') }}</button></div></form></section>
    <section class="card"><p v-if="busy">{{ tr('shareLoading') }}</p><div v-for="share in shares" :key="share.id" class="share-row"><div><strong>{{ share.name }}</strong><small>{{ tr(`shareAccess${share.accessMode[0].toUpperCase()}${share.accessMode.slice(1)}`) }} · {{ status(share) }} · {{ range(share) }}</small></div><div class="row-actions"><button class="ghost" @click="reset(share)">{{ tr('shareEdit') }}</button><button class="ghost" @click="rotate(share)">{{ tr('shareRegenerate') }}</button><button class="ghost danger-text" @click="remove(share)">{{ tr('shareDelete') }}</button></div></div><p v-if="!busy&&!shares.length" class="empty-inline">{{ tr('noShares') }}</p></section>
  </section>
</template>

<style scoped>
.share-manager{display:grid;gap:20px}.form-card form{display:grid;gap:14px;max-width:720px}.form-card fieldset{display:flex;flex-wrap:wrap;gap:12px;padding:14px;border:1px solid var(--line);border-radius:12px}.form-card fieldset label{display:flex;gap:6px;align-items:center}.two{display:grid;grid-template-columns:1fr 1fr;gap:12px}.share-row{display:flex;justify-content:space-between;align-items:center;gap:16px;padding:14px 0;border-bottom:1px solid var(--line)}.share-row small{display:block;color:var(--muted);margin-top:4px}.row-actions{display:flex;flex-wrap:wrap;gap:8px}@media(max-width:650px){.share-row{align-items:flex-start;flex-direction:column}.two{grid-template-columns:1fr}}
</style>

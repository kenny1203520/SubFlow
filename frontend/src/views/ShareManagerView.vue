<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { pb } from '../pocketbase'
import { ApiClient } from '../api/client'
import type { Share, ShareAccessMode, ShareRangeMode } from '../api/types'
import type { Contact, Membership } from '../api/types'
import PasswordField from '../components/PasswordField.vue'
import { serializeShareForm } from './shareForm'
import { useI18n } from '../i18n'

const route = useRoute()
const client = new ApiClient(() => pb.authStore.token, () => {})
const { tr, formatDate } = useI18n()
const groupId = computed(() => String(route.params.groupId || ''))
const isGroup = computed(() => !!groupId.value)
const shares = ref<Share[]>([])
const busy = ref(false)
const error = ref('')
const copied = ref('')
const knownUrls = ref<Record<string, string>>({})
const editing = ref<Share | null | undefined>(undefined)
const form = reactive({ name: '', accessMode: 'link' as ShareAccessMode, password: '', viewerEmails: '', enabled: true, expiresAt: '', rangeMode: 'all' as ShareRangeMode, rollingDays: 30, startsOn: '', endsOn: '', showSummary: true, showExpenses: true, showSubscriptions: true, showSettlements: true, showIdentities: false, showNotes: false })
const contacts = ref<Contact[]>([])
const memberships = ref<Membership[]>([])
const readerQuery = ref('')
const readerError = ref('')
const showContactForm = ref(false)
const contactBusy = ref(false)
const newContact = reactive({ name: '', email: '' })
const selectedReaders = computed(() => form.viewerEmails.split(',').map(value => value.trim().toLowerCase()).filter(Boolean))
const readerCandidates = computed(() => {
  const query = readerQuery.value.trim().toLowerCase()
  const values = [...contacts.value.map(contact => ({ name: contact.name, email: contact.email })), ...memberships.value.flatMap(member => member.user?.email ? [{ name: member.user.name, email: member.user.email }] : [])]
  const seen = new Set<string>()
  return values.filter(value => {
    const email = value.email.toLowerCase()
    if (seen.has(email) || selectedReaders.value.includes(email)) return false
    seen.add(email)
    return !query || email.includes(query) || value.name.toLowerCase().includes(query)
  }).slice(0, 8)
})
function addReader(email: string) {
  readerError.value = ''
  const normalized = email.trim().toLowerCase()
  if (!normalized || selectedReaders.value.includes(normalized)) return
  form.viewerEmails = [...selectedReaders.value, normalized].join(',')
  readerQuery.value = ''
}
function removeReader(email: string) {
  readerError.value = ''
  form.viewerEmails = selectedReaders.value.filter(value => value !== email).join(',')
}
function addTypedReader() {
  const values = readerQuery.value.split(',').map(value => value.trim()).filter(Boolean)
  values.forEach(addReader)
}
async function loadContacts() {
  try {
    contacts.value = (await client.get<Contact[]>('/personal/contacts')).data
  } catch { contacts.value = [] }
  if (isGroup.value) {
    try { memberships.value = (await client.get<Membership[]>('/groups/' + groupId.value + '/members')).data } catch { memberships.value = [] }
  }
}
async function createContact() {
  contactBusy.value = true
  try {
    const result = await client.post<Contact>('/personal/contacts', newContact)
    contacts.value = [...contacts.value, result.data]
    addReader(result.data.email)
    newContact.name = ''
    newContact.email = ''
    showContactForm.value = false
  } catch (reason) {
    error.value = reason instanceof Error ? reason.message : tr('contactSaveFailed')
  } finally { contactBusy.value = false }
}

const base = computed(() => isGroup.value ? `/groups/${groupId.value}/shares` : '/personal/shares')
const knownUrlsKey = computed(() => 'subflow.share.urls.' + (isGroup.value ? 'group.' + groupId.value : 'personal'))
function loadKnownUrls() {
  const legacyKey = knownUrlsKey.value
  try {
    const value = localStorage.getItem(legacyKey) || sessionStorage.getItem(legacyKey)
    knownUrls.value = value ? JSON.parse(value) : {}
    if (value && !localStorage.getItem(legacyKey)) localStorage.setItem(legacyKey, value)
  } catch { knownUrls.value = {} }
}
function rememberUrl(id: string, url: string) {
  knownUrls.value = { ...knownUrls.value, [id]: url }
  try { localStorage.setItem(knownUrlsKey.value, JSON.stringify(knownUrls.value)) } catch {}
}
function knownShareUrl(value: Share) { return value.url || knownUrls.value[value.id] || '' }
function extractShareToken(value: string) {
  try {
    const parsed = new URL(value.trim(), window.location.origin)
    if (parsed.origin !== window.location.origin || parsed.search || parsed.hash) return ''
    const parts = parsed.pathname.split('/').filter(Boolean)
    return parts.length === 2 && parts[0] === 'share' ? decodeURIComponent(parts[1]) : ''
  } catch { return '' }
}
function openShare(value: Share) {
  const url = knownShareUrl(value)
  if (url) window.open(new URL(url, window.location.origin).toString(), '_blank', 'noopener')
}

const accessOptions = computed(() => [
  { value: 'link' as const, label: tr('shareAccessLink'), description: tr('shareAccessDescLink'), marker: '↗' },
  { value: 'password' as const, label: tr('shareAccessPassword'), description: tr('shareAccessDescPassword'), marker: '⌁' },
  { value: 'accounts' as const, label: tr('shareAccessAccounts'), description: tr('shareAccessDescAccounts'), marker: '◎' },
])
const contentOptions = computed(() => [
  { key: 'showSummary' as const, label: tr('shareSummary'), description: tr('shareContentSummaryDesc'), marker: '▦' },
  { key: 'showExpenses' as const, label: tr('shareExpenses'), description: tr('shareContentExpensesDesc'), marker: '↙' },
  { key: 'showSubscriptions' as const, label: tr('shareSubscriptions'), description: tr('shareContentSubscriptionsDesc'), marker: '⟳' },
  ...(isGroup.value ? [{ key: 'showSettlements' as const, label: tr('shareSettlements'), description: tr('shareContentSettlementsDesc'), marker: '⇄' }] : []),
])

function meaningfulDate(value?: string) {
  return !!value && !value.startsWith('0000') && !value.startsWith('0001')
}
function reset(value?: Share) {
  editing.value = value ?? null
  readerError.value = ''
  Object.assign(form, { name: value?.name || '', accessMode: value?.accessMode || 'link', password: '', viewerEmails: value?.viewers?.map(item => item.email).join(',') || '', enabled: value?.enabled ?? true, expiresAt: meaningfulDate(value?.expiresAt) ? value!.expiresAt!.slice(0, 10) : '', rangeMode: value?.rangeMode || 'all', rollingDays: value?.rollingDays || 30, startsOn: meaningfulDate(value?.startsOn) ? value!.startsOn!.slice(0, 10) : '', endsOn: meaningfulDate(value?.endsOn) ? value!.endsOn!.slice(0, 10) : '', showSummary: value?.showSummary ?? true, showExpenses: value?.showExpenses ?? true, showSubscriptions: value?.showSubscriptions ?? true, showSettlements: value?.showSettlements ?? true, showIdentities: value?.showIdentities ?? false, showNotes: value?.showNotes ?? false })
}
function payload() { return serializeShareForm(form) }
async function load() {
  busy.value = true
  error.value = ''
  try {
    const result = await client.get<Share[]>(base.value)
    shares.value = result.data
    result.data.forEach(share => { if (share.url) rememberUrl(share.id, share.url) })
  } catch (reason) { error.value = reason instanceof Error ? reason.message : tr('requestFailed') }
  finally { busy.value = false }
}
async function save() {
  error.value = ''
  readerError.value = ''
  if (form.accessMode === 'accounts' && selectedReaders.value.length === 0) {
    readerError.value = tr('shareReaderRequired')
    return
  }
  busy.value = true
  try {
    if (editing.value?.id) await client.patch<Share>(base.value + '/' + editing.value.id, payload())
    else {
      const result = await client.post<Share & { url?: string }>(base.value, payload())
      if (result.data.url) { rememberUrl(result.data.id, result.data.url); await copy(result.data.url) }
    }
    editing.value = undefined
    await load()
  } catch (reason) {
    if (form.accessMode === 'accounts') readerError.value = tr('shareReaderValidation')
    error.value = reason instanceof Error ? reason.message : tr('requestFailed')
  } finally { busy.value = false }
}
async function remove(value: Share) { if (!confirm(tr('shareDeleteConfirm', { name: value.name }))) return; await client.delete(base.value + '/' + value.id); await load() }
async function rotate(value: Share) {
  if (!confirm(tr('shareRegenerateConfirm'))) return
  busy.value = true
  error.value = ''
  try {
    const result = await client.post<Share & { url: string }>(base.value + '/' + value.id + '/rotate')
    rememberUrl(result.data.id, result.data.url)
    await copy(result.data.url)
    await load()
  } catch (reason) { error.value = reason instanceof Error ? reason.message : tr('requestFailed') }
  finally { busy.value = false }
}
async function remember(value: Share) {
  const cached = knownShareUrl(value)
  const raw = cached || window.prompt(tr('shareRememberUrlPrompt'))
  if (!raw) return
  const token = extractShareToken(raw)
  if (!token) { error.value = tr('shareRememberUrlInvalid'); return }
  busy.value = true
  error.value = ''
  try {
    const result = await client.post<Share & { url?: string }>(base.value + '/' + value.id + '/remember', { token })
    if (result.data.url) rememberUrl(result.data.id, result.data.url)
    copied.value = tr('shareRemembered')
    await load()
    setTimeout(() => { copied.value = '' }, 3000)
  } catch (reason) { error.value = reason instanceof Error ? reason.message : tr('requestFailed') }
  finally { busy.value = false }
}
async function copy(value: string) { const url = new URL(value, window.location.origin).toString(); await navigator.clipboard?.writeText(url); copied.value = tr('shareLinkCopied', { url }); setTimeout(() => { copied.value = '' }, 3000) }
function status(value: Share) { return tr(value.enabled ? 'shareStatusEnabled' : 'shareStatusDisabled') }
function range(value: Share) { if (value.rangeMode === 'all') return tr('shareRangeAll'); if (value.rangeMode === 'rolling') return tr('shareRangeRollingValue', { days: value.rollingDays || 30 }); return value.startsOn && value.endsOn ? tr('shareRangeFixedValue', { from: formatDate(value.startsOn), to: formatDate(value.endsOn) }) : tr('shareRangeFixed') }
function expiry(value: Share) { return meaningfulDate(value.expiresAt) ? tr('shareExpiryValue', { date: formatDate(value.expiresAt!) }) : tr('shareNoExpiry') }
function accessLabel(value: ShareAccessMode) { return tr(`shareAccess${value[0].toUpperCase()}${value.slice(1)}`) }
onMounted(async () => { loadKnownUrls(); await load(); await loadContacts() })
watch(groupId, () => { loadKnownUrls(); void load() })
</script>

<template>
  <section class="page share-manager">
    <div class="page-heading share-heading"><div><p class="eyebrow">{{ tr(isGroup ? 'shareLedgerGroup' : 'shareLedgerPersonal') }}</p><h1>{{ tr('sharePages') }}</h1><p>{{ tr('sharePagesDesc') }}</p></div><button class="primary create-button" @click="reset()"><span aria-hidden="true">＋</span>{{ tr('createShare') }}</button></div>
    <p v-if="error" class="notice danger">{{ error }}</p><p v-if="copied" class="notice success">{{ copied }}</p>

    <section v-if="editing !== undefined" class="share-editor card" aria-labelledby="share-editor-title">
      <div class="editor-heading"><div class="editor-mark" aria-hidden="true">↗</div><div><p class="eyebrow">{{ editing?.id ? tr('shareEdit') : tr('newShare') }}</p><h2 id="share-editor-title">{{ editing?.id ? tr('shareEditorEditTitle') : tr('shareEditorNewTitle') }}</h2><p>{{ tr('shareEditorDesc') }}</p></div></div>
      <form @submit.prevent="save">
        <section class="editor-section identity-section"><label class="share-name-field"><span>{{ tr('shareName') }}</span><input v-model="form.name" required maxlength="120" :placeholder="tr('shareNamePlaceholder')"></label></section>
        <section class="editor-section"><div class="section-heading"><div><h3>{{ tr('shareAccess') }}</h3><p>{{ tr('shareAccessDesc') }}</p></div></div><div class="access-options"><label v-for="option in accessOptions" :key="option.value" class="access-option" :class="{ selected: form.accessMode === option.value }"><input v-model="form.accessMode" type="radio" :value="option.value"><span class="option-mark" aria-hidden="true">{{ option.marker }}</span><span><strong>{{ option.label }}</strong><small>{{ option.description }}</small></span><span class="radio-indicator" aria-hidden="true"></span></label></div><div v-if="form.accessMode === 'password'" class="access-detail"><PasswordField v-model="form.password" :label="editing?.id ? tr('sharePasswordNew') : tr('sharePassword')" :required="!editing?.id" :minlength="8" autocomplete="new-password" :help="tr('sharePasswordHelp')" /></div><div v-if="form.accessMode === 'accounts'" class="access-detail reader-picker"><label><span>{{ tr('shareReaders') }}</span><div class="reader-chips"><span v-for="email in selectedReaders" :key="email" class="reader-chip">{{ email }}<button type="button" @click="removeReader(email)" :aria-label="tr('remove')">×</button></span><span v-if="!selectedReaders.length" class="reader-empty">{{ tr('shareNoReaders') }}</span></div></label><p v-if="readerError" class="field-error">{{ readerError }}</p><div class="reader-input-row"><input v-model="readerQuery" :placeholder="tr('shareReaderSearchPlaceholder')" @keydown.enter.prevent="addTypedReader"><button type="button" class="ghost" @click="addTypedReader">{{ tr('shareAddReader') }}</button></div><div v-if="readerCandidates.length" class="reader-candidates"><button v-for="candidate in readerCandidates" :key="candidate.email" type="button" class="reader-candidate" @click="addReader(candidate.email)"><strong>{{ candidate.name || candidate.email }}</strong><small>{{ candidate.email }}</small></button></div><p>{{ tr('shareAccessAccountsHelp') }}</p><div class="quick-contact"><button type="button" class="ghost" @click="showContactForm = !showContactForm">{{ tr('shareAddContact') }}</button><div v-if="showContactForm" class="quick-contact-form"><input v-model="newContact.name" :placeholder="tr('contactName')"><input v-model="newContact.email" :placeholder="tr('contactEmail')"><button type="button" class="primary" :disabled="contactBusy" @click="createContact">{{ tr('contactSave') }}</button></div></div></div></section>
        <div class="editor-columns"><section class="editor-section"><div class="section-heading"><div><h3>{{ tr('shareRange') }}</h3><p>{{ tr('shareRangeDesc') }}</p></div></div><div class="range-options"><label v-for="mode in ['all', 'rolling', 'fixed'] as ShareRangeMode[]" :key="mode" :class="{ selected: form.rangeMode === mode }"><input v-model="form.rangeMode" type="radio" :value="mode"><span>{{ tr(mode === 'all' ? 'shareRangeAll' : mode === 'rolling' ? 'shareRangeRolling' : 'shareRangeFixed') }}</span></label></div><label v-if="form.rangeMode === 'rolling'" class="form-field"><span>{{ tr('shareRecentDays') }}</span><select v-model.number="form.rollingDays"><option :value="30">30</option><option :value="90">90</option><option :value="365">365</option></select></label><div v-if="form.rangeMode === 'fixed'" class="date-fields"><label><span>{{ tr('shareFrom') }}</span><input v-model="form.startsOn" type="date"></label><label><span>{{ tr('shareTo') }}</span><input v-model="form.endsOn" type="date"></label></div></section><section class="editor-section"><div class="section-heading"><div><h3>{{ tr('shareAvailability') }}</h3><p>{{ tr('shareAvailabilityDesc') }}</p></div></div><label class="toggle-line"><input v-model="form.enabled" type="checkbox"><span class="toggle-control" aria-hidden="true"></span><span><strong>{{ tr('shareEnabled') }}</strong><small>{{ form.enabled ? tr('shareStatusEnabled') : tr('shareStatusDisabled') }}</small></span></label><label class="form-field"><span>{{ tr('shareExpires') }}</span><input v-model="form.expiresAt" type="date"></label></section></div>
        <section class="editor-section content-section"><div class="section-heading"><div><h3>{{ tr('shareContent') }}</h3><p>{{ tr('shareContentDesc') }}</p></div></div><div class="content-options"><label v-for="option in contentOptions" :key="option.key" class="content-option" :class="{ selected: form[option.key] }"><input v-model="form[option.key]" type="checkbox"><span class="option-mark" aria-hidden="true">{{ option.marker }}</span><span><strong>{{ option.label }}</strong><small>{{ option.description }}</small></span><span class="check-indicator" aria-hidden="true">✓</span></label></div><div class="privacy-options"><div><h4>{{ tr('sharePrivacy') }}</h4><p>{{ tr('sharePrivacyDesc') }}</p></div><label class="privacy-toggle"><input v-model="form.showIdentities" type="checkbox"><span><strong>{{ tr('shareIdentities') }}</strong><small>{{ tr('shareIdentitiesDesc') }}</small></span></label><label class="privacy-toggle"><input v-model="form.showNotes" type="checkbox"><span><strong>{{ tr('shareNotes') }}</strong><small>{{ tr('shareNotesDesc') }}</small></span></label></div></section>
        <div class="editor-actions"><button class="primary" :disabled="busy">{{ busy ? tr('shareSaving') : tr('save') }}</button><button class="ghost" type="button" @click="editing = undefined">{{ tr('cancel') }}</button></div>
      </form>
    </section>

    <section class="shares-card card" aria-labelledby="shares-list-title"><div class="shares-heading"><div><p class="eyebrow">{{ tr('shareActiveLinks') }}</p><h2 id="shares-list-title">{{ tr('shareListTitle') }}</h2><p>{{ tr('shareListDesc') }}</p></div><span v-if="shares.length" class="shares-count">{{ shares.length }}</span></div><p v-if="busy && !shares.length" class="loading-state">{{ tr('shareLoading') }}</p><div v-else-if="shares.length" class="share-list"><article v-for="share in shares" :key="share.id" class="share-item"><div class="share-status" :class="{ inactive: !share.enabled }" aria-hidden="true"></div><div class="share-item-main"><div class="share-item-title"><h3>{{ share.name }}</h3><span class="status-pill" :class="{ inactive: !share.enabled }">{{ status(share) }}</span></div><div class="share-facts"><span>{{ accessLabel(share.accessMode) }}</span><span>{{ range(share) }}</span><span>{{ expiry(share) }}</span><span v-if="share.accessMode === 'accounts'">{{ tr('shareViewerCount', { count: share.viewers?.length || 0 }) }}</span></div></div><div class="share-actions"><button v-if="knownShareUrl(share)" class="ghost" @click="copy(knownShareUrl(share))">{{ tr('shareCopyLink') }}</button><button v-if="knownShareUrl(share)" class="ghost" @click="openShare(share)">{{ tr('shareOpenLink') }}</button><button v-if="!share.url" class="ghost" @click="remember(share)">{{ tr('shareRememberLink') }}</button><button class="ghost" @click="reset(share)">{{ tr('shareEdit') }}</button><button class="ghost" @click="rotate(share)">{{ tr('shareRegenerate') }}</button><button class="ghost danger-text" @click="remove(share)">{{ tr('shareDelete') }}</button></div></article></div><div v-else class="share-empty"><div class="empty-mark" aria-hidden="true">↗</div><h3>{{ tr('noShares') }}</h3><p>{{ tr('noSharesDesc') }}</p><button class="primary" @click="reset()">{{ tr('createFirstShare') }}</button></div></section>
  </section>
</template>

<style>
.share-manager{display:grid;gap:22px}.share-heading{margin-bottom:0}.create-button span{font-size:17px;line-height:1}.share-editor,.shares-card{padding:0;overflow:hidden}.editor-heading{display:flex;gap:15px;padding:26px 28px;border-bottom:1px solid var(--line);background:var(--surface-soft)}.editor-mark{display:grid;place-items:center;width:42px;height:42px;flex:0 0 auto;border-radius:13px;background:var(--brand);color:#fff;font-size:21px;box-shadow:none}.editor-heading h2,.section-heading h3,.shares-heading h2{margin:3px 0 5px;font-size:19px;letter-spacing:-.025em}.editor-heading p:not(.eyebrow),.section-heading p,.shares-heading p,.access-detail p,.privacy-options p{margin:0;color:var(--muted);font-size:13px;line-height:1.5}.share-editor form{display:grid;gap:0}.editor-section{padding:24px 28px;border-bottom:1px solid var(--line)}.identity-section{padding-bottom:22px}.share-name-field,.form-field,.date-fields label,.access-detail label{display:grid;gap:7px;color:var(--ink);font-size:12px;font-weight:750}.share-name-field{width:min(600px,100%)}.section-heading{margin-bottom:15px}.access-options{display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:10px}.access-option,.content-option{position:relative;display:grid;grid-template-columns:auto 1fr auto;gap:10px;align-items:start;min-height:102px;padding:14px;border:1px solid var(--line);border-radius:14px;background:var(--surface-soft);cursor:pointer}.access-option:hover,.content-option:hover{border-color:color-mix(in srgb,var(--brand) 55%,var(--line));background:var(--surface)}.access-option.selected,.content-option.selected{border-color:var(--brand);background:var(--brand-soft);box-shadow:inset 0 0 0 1px color-mix(in srgb,var(--brand) 35%,transparent)}.access-option input,.content-option input,.range-options input{position:absolute;width:1px;height:1px;opacity:0;pointer-events:none}.option-mark{display:grid;place-items:center;width:30px;height:30px;border-radius:9px;background:var(--surface);color:var(--brand);font-weight:850;font-size:16px}.access-option strong,.content-option strong{display:block;font-size:13px}.access-option small,.content-option small,.toggle-line small,.privacy-toggle small{display:block;margin-top:3px;color:var(--muted);font-size:11px;font-weight:500;line-height:1.4}.radio-indicator{width:17px;height:17px;border:2px solid var(--line-strong);border-radius:50%}.access-option.selected .radio-indicator{border:5px solid var(--brand)}.access-detail{display:grid;gap:9px;margin-top:13px;padding:15px;border-left:3px solid var(--brand);border-radius:0 12px 12px 0;background:var(--surface-soft)}.editor-columns{display:grid;grid-template-columns:1fr 1fr}.editor-columns>.editor-section:first-child{border-right:1px solid var(--line)}.range-options{display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:7px;margin-bottom:14px;padding:4px;border-radius:12px;background:var(--surface-soft)}.range-options label{display:grid;place-items:center;min-height:38px;padding:7px;border-radius:9px;color:var(--muted);font-size:12px;font-weight:750;text-align:center;cursor:pointer}.range-options label.selected{background:var(--surface);color:var(--brand);box-shadow:0 2px 8px rgba(20,30,60,.1)}.date-fields{display:grid;grid-template-columns:1fr 1fr;gap:10px}.toggle-line{display:flex;align-items:center;gap:10px;margin:0 0 15px;padding:11px 12px;border:1px solid var(--line);border-radius:12px;background:var(--surface-soft);cursor:pointer}.toggle-line input{position:absolute;opacity:0}.toggle-control{position:relative;width:35px;height:21px;border-radius:999px;background:var(--line-strong);transition:background .18s ease}.toggle-control::after{content:"";position:absolute;top:3px;left:3px;width:15px;height:15px;border-radius:50%;background:#fff;transition:transform .18s ease}.toggle-line input:checked+.toggle-control{background:var(--brand)}.toggle-line input:checked+.toggle-control::after{transform:translateX(14px)}.toggle-line strong{display:block;font-size:13px}.content-section{border-bottom:0}.content-options{display:grid;grid-template-columns:repeat(4,minmax(0,1fr));gap:10px}.content-option{min-height:90px}.check-indicator{display:grid;place-items:center;width:18px;height:18px;border:2px solid var(--line-strong);border-radius:6px;color:transparent;font-size:12px;font-weight:900}.content-option.selected .check-indicator{border-color:var(--brand);background:var(--brand);color:#fff}.privacy-options{display:grid;grid-template-columns:1.15fr 1fr 1fr;gap:10px;align-items:stretch;margin-top:14px;padding-top:14px;border-top:1px solid var(--line)}.privacy-options h4{margin:0 0 4px;font-size:13px}.privacy-toggle{display:flex;align-items:flex-start;gap:9px;padding:11px;border:1px solid var(--line);border-radius:12px;background:var(--surface-soft);cursor:pointer}.privacy-toggle input{width:17px;height:17px;flex:0 0 auto;margin:1px 0 0;accent-color:var(--brand)}.privacy-toggle strong{font-size:12px}.editor-actions{display:flex;align-items:center;gap:8px;padding:17px 28px;border-top:1px solid var(--line);background:color-mix(in srgb,var(--surface) 92%,transparent)}.shares-heading{display:flex;align-items:flex-start;justify-content:space-between;gap:12px;padding:25px 28px 20px;border-bottom:1px solid var(--line)}.shares-count{display:grid;place-items:center;min-width:30px;height:30px;padding:0 8px;border-radius:999px;background:var(--brand-soft);color:var(--brand);font-size:13px;font-weight:800}.share-list{display:grid}.share-item{display:grid;grid-template-columns:auto minmax(0,1fr) auto;gap:13px;align-items:center;padding:18px 28px;border-bottom:1px solid var(--line)}.share-item:last-child{border-bottom:0}.share-item:hover{background:color-mix(in srgb,var(--brand-soft) 36%,transparent)}.share-status{width:9px;height:9px;border-radius:50%;background:var(--success);box-shadow:0 0 0 5px color-mix(in srgb,var(--success) 14%,transparent)}.share-status.inactive{background:var(--muted);box-shadow:0 0 0 5px color-mix(in srgb,var(--muted) 14%,transparent)}.share-item-title{display:flex;align-items:center;gap:8px}.share-item h3{margin:0;font-size:14px}.status-pill{padding:3px 7px;border-radius:999px;background:var(--success-soft);color:var(--success);font-size:10px;font-weight:800}.status-pill.inactive{background:var(--surface-soft);color:var(--muted)}.share-facts{display:flex;flex-wrap:wrap;gap:5px 13px;margin-top:5px;color:var(--muted);font-size:11px}.share-facts span+span{position:relative}.share-facts span+span::before{content:"";position:absolute;left:-8px;top:50%;width:3px;height:3px;border-radius:50%;background:var(--line-strong)}.share-actions{display:flex;flex-wrap:wrap;justify-content:flex-end;gap:3px}.loading-state{margin:0;padding:25px 28px;color:var(--muted);font-size:13px}.share-empty{display:grid;justify-items:center;padding:52px 20px;text-align:center}.share-empty h3{margin:0 0 7px;font-size:16px}.share-empty p{max-width:390px;margin:0 0 17px;color:var(--muted);font-size:13px;line-height:1.55}.share-empty .empty-mark{margin-bottom:12px}@media(max-width:980px){.access-options{grid-template-columns:1fr}.access-option{min-height:76px}.content-options{grid-template-columns:repeat(2,minmax(0,1fr))}.privacy-options{grid-template-columns:1fr 1fr}.privacy-options>div{grid-column:1/-1}.share-item{align-items:start}.share-actions{justify-content:flex-start}}@media(max-width:680px){.share-editor,.shares-card{border-radius:15px}.editor-heading,.editor-section,.shares-heading,.editor-actions{padding-left:18px;padding-right:18px}.editor-columns{grid-template-columns:1fr}.editor-columns>.editor-section:first-child{border-right:0;border-bottom:1px solid var(--line)}.privacy-options{grid-template-columns:1fr}.content-options{grid-template-columns:1fr}.share-item{grid-template-columns:auto minmax(0,1fr);padding:16px 18px}.share-actions{grid-column:2;justify-content:flex-start}.share-facts{line-height:1.5}.range-options{grid-template-columns:1fr}.range-options label{min-height:34px}.date-fields{grid-template-columns:1fr}.create-button{width:100%}}
.reader-chips{display:flex;flex-wrap:wrap;gap:7px;min-height:44px;padding:8px;border:1px solid var(--line-strong);border-radius:10px;background:var(--surface)}
.reader-chip{display:inline-flex;align-items:center;gap:6px;padding:5px 8px;border-radius:999px;background:var(--brand-soft);color:var(--brand-strong);font-size:13px;font-weight:650}
.reader-chip button{width:20px;height:20px;padding:0;border-radius:50%;background:transparent;color:inherit;font-size:16px;line-height:1}
.reader-chip button:hover{background:color-mix(in srgb,var(--brand) 16%,transparent)}
.reader-empty{padding:5px;color:var(--muted);font-size:13px}
.reader-input-row{display:flex;gap:8px;align-items:center}
.reader-input-row input{flex:1}
.reader-candidates{display:grid;grid-template-columns:repeat(auto-fit,minmax(180px,1fr));gap:7px}
.reader-candidate{display:grid;gap:2px;padding:9px 10px;border:1px solid var(--line);border-radius:10px;background:var(--surface);color:var(--ink);text-align:left}
.reader-candidate:hover{border-color:var(--brand);background:var(--brand-soft)}
.reader-candidate strong{font-size:14px}
.reader-candidate small{font-size:13px;color:var(--muted)}
.quick-contact{display:grid;gap:9px}
.quick-contact-form{display:grid;grid-template-columns:1fr 1fr auto;gap:8px;align-items:center}
@media(max-width:680px){.quick-contact-form{grid-template-columns:1fr}.reader-input-row{align-items:stretch;flex-direction:column}.reader-input-row .ghost{width:100%}}</style>

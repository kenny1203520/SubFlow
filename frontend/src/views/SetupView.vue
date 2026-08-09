<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import CurrencySelect from '../components/CurrencySelect.vue'
import TimezoneSelect from '../components/TimezoneSelect.vue'
import { useWorkspaceStore } from '../stores/workspace'
import { useSetupStore } from '../stores/setup'
import { useAuthStore } from '../stores/auth'
import { useI18n } from '../i18n'

const setup=useSetupStore(),auth=useAuthStore(),workspace=useWorkspaceStore(),router=useRouter(),route=useRoute(),{tr}=useI18n()
const busy=ref(false),error=ref('')
const token=String(route.query.token||'')
const form=reactive({adminName:'',email:'',password:'',siteName:'SubFlow',defaultTimezone:Intl.DateTimeFormat().resolvedOptions().timeZone||'UTC',defaultCurrency:'TWD',allowRegistration:true})
onMounted(async()=>{try{const status=await setup.refresh(token);if(status.initialized){void router.replace(auth.authenticated?'/':'/auth');return}form.siteName=status.siteName||form.siteName;form.defaultTimezone=status.defaultTimezone||form.defaultTimezone;form.defaultCurrency=status.defaultCurrency||form.defaultCurrency;form.allowRegistration=!!status.allowRegistration}catch{error.value=tr('requestFailed')}})
async function submit(){busy.value=true;error.value='';try{await setup.initialize({...form,token});await auth.login(form.email,form.password);await workspace.loadGroups();await router.replace('/')}catch(reason){error.value=reason instanceof Error?reason.message:tr('requestFailed')}finally{busy.value=false}}
</script>
<template><section class="auth-panel"><div class="auth-card setup-card"><p class="eyebrow">{{tr('setup')}}</p><h1>{{tr('setupTitle')}}</h1><p>{{tr('setupDesc')}}</p><template v-if="setup.status.setupAvailable"><form @submit.prevent="submit"><label>{{tr('displayName')}}<input v-model="form.adminName" required autocomplete="name"></label><label>{{tr('email')}}<input v-model="form.email" type="email" required autocomplete="email"></label><label>{{tr('password')}}<input v-model="form.password" type="password" minlength="8" required autocomplete="new-password"></label><hr><label>{{tr('siteName')}}<input v-model="form.siteName" required maxlength="120"></label><label>{{tr('timezone')}}<TimezoneSelect v-model="form.defaultTimezone"/></label><label>{{tr('currency')}}<CurrencySelect v-model="form.defaultCurrency" :currencies="setup.status.currencies?.length?setup.status.currencies:workspace.currencies"/></label><label class="check"><input v-model="form.allowRegistration" type="checkbox"><span>{{tr('allowRegistration')}}</span></label><p v-if="error" class="form-error">{{error}}</p><button class="primary wide" :disabled="busy">{{busy?tr('processing'):tr('completeSetup')}}</button></form></template><div v-else class="setup-link-required"><strong>{{tr('setupLinkRequired')}}</strong><p>{{tr('setupLinkRequiredDesc')}}</p></div></div></section></template>

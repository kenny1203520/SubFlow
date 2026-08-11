<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import CurrencySelect from '../components/CurrencySelect.vue'
import TimezoneSelect from '../components/TimezoneSelect.vue'
import { useWorkspaceStore } from '../stores/workspace'
import { useSetupStore } from '../stores/setup'
import { useAuthStore } from '../stores/auth'
import { useI18n } from '../i18n'
import BaseInput from '../components/BaseInput.vue'
import PasswordField from '../components/PasswordField.vue'

const setup=useSetupStore(),auth=useAuthStore(),workspace=useWorkspaceStore(),router=useRouter(),route=useRoute(),{tr}=useI18n()
const busy=ref(false),completed=ref(false),error=ref('')
const token=String(route.query.token||'')
const form=reactive({adminName:'',email:'',password:'',siteName:'SubFlow',defaultTimezone:Intl.DateTimeFormat().resolvedOptions().timeZone||'UTC',defaultCurrency:'TWD',allowRegistration:true})
onMounted(async()=>{try{const status=await setup.refresh(token);if(status.initialized){void router.replace(auth.authenticated?'/':'/auth');return}form.siteName=status.siteName||form.siteName;form.defaultTimezone=status.defaultTimezone||form.defaultTimezone;form.defaultCurrency=status.defaultCurrency||form.defaultCurrency;form.allowRegistration=!!status.allowRegistration}catch{error.value=tr('requestFailed')}})
async function submit(){busy.value=true;error.value='';try{await setup.initialize({...form,token});await auth.login(form.email,form.password);await workspace.loadGroups();completed.value=true;await new Promise(resolve=>setTimeout(resolve,700));await setup.refresh();await router.replace({name:'dashboard'})}catch(reason){completed.value=false;error.value=reason instanceof Error?reason.message:tr('requestFailed')}finally{busy.value=false}}
</script>
<template><section class="auth-panel"><div class="auth-card setup-card"><template v-if="completed"><div class="setup-success" role="status"><div class="setup-success-mark">✓</div><h1>{{tr('setupCompleted')}}</h1><p>{{tr('setupRedirecting')}}</p></div></template><template v-else><p class="eyebrow">{{tr('setup')}}</p><h1>{{tr('setupTitle')}}</h1><p>{{tr('setupDesc')}}</p><template v-if="setup.status.setupAvailable"><form @submit.prevent="submit"><BaseInput v-model="form.adminName" :label="tr('displayName')" required autocomplete="name"/><BaseInput v-model="form.email" :label="tr('email')" type="email" required autocomplete="email"/><PasswordField v-model="form.password" :label="tr('password')" :minlength="8" required autocomplete="new-password"/><hr><BaseInput v-model="form.siteName" :label="tr('siteName')" required :maxlength="120"/><label>{{tr('timezone')}}<TimezoneSelect v-model="form.defaultTimezone"/></label><label>{{tr('currency')}}<CurrencySelect v-model="form.defaultCurrency" :currencies="setup.status.currencies?.length?setup.status.currencies:workspace.currencies"/></label><label class="check"><input v-model="form.allowRegistration" type="checkbox"><span>{{tr('allowRegistration')}}</span></label><p v-if="error" class="form-error">{{error}}</p><button class="primary wide" :disabled="busy">{{busy?tr('processing'):tr('completeSetup')}}</button></form></template><div v-else class="setup-link-required"><strong>{{tr('setupLinkRequired')}}</strong><p>{{tr('setupLinkRequiredDesc')}}</p></div></template></div></section></template>

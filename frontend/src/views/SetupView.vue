<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import CurrencySelect from '../components/CurrencySelect.vue'
import TimezoneSelect from '../components/TimezoneSelect.vue'
import { useWorkspaceStore } from '../stores/workspace'
import { useSetupStore } from '../stores/setup'
import { useAuthStore } from '../stores/auth'
import { useI18n } from '../i18n'

const setup=useSetupStore(),auth=useAuthStore(),workspace=useWorkspaceStore(),router=useRouter(),{tr}=useI18n()
const busy=ref(false),error=ref('')
const form=reactive({adminName:'',email:'',password:'',siteName:'SubFlow',defaultTimezone:Intl.DateTimeFormat().resolvedOptions().timeZone||'UTC',defaultCurrency:'TWD',allowRegistration:true,secret:''})
onMounted(async()=>{try{const status=await setup.refresh();if(status.initialized){void router.replace(auth.authenticated?'/':'/auth');return}form.siteName=status.siteName||form.siteName;form.defaultTimezone=status.defaultTimezone||form.defaultTimezone;form.defaultCurrency=status.defaultCurrency||form.defaultCurrency;form.allowRegistration=!!status.allowRegistration}catch{error.value=tr('requestFailed')}})
async function submit(){busy.value=true;error.value='';try{await setup.initialize(form);await auth.login(form.email,form.password);await workspace.loadGroups();await router.replace('/')}catch(reason){error.value=reason instanceof Error?reason.message:tr('requestFailed')}finally{busy.value=false}}
</script>
<template><section class="auth-panel"><div class="auth-card setup-card"><p class="eyebrow">{{tr('setup')}}</p><h1>{{tr('setupTitle')}}</h1><p>{{tr('setupDesc')}}</p><form @submit.prevent="submit"><label>{{tr('displayName')}}<input v-model="form.adminName" required autocomplete="name"></label><label>{{tr('email')}}<input v-model="form.email" type="email" required autocomplete="email"></label><label>{{tr('password')}}<input v-model="form.password" type="password" minlength="8" required autocomplete="new-password"></label><hr><label>{{tr('siteName')}}<input v-model="form.siteName" required maxlength="120"></label><label>{{tr('timezone')}}<TimezoneSelect v-model="form.defaultTimezone"/></label><label>{{tr('currency')}}<CurrencySelect v-model="form.defaultCurrency" :currencies="workspace.currencies"/></label><label class="check"><input v-model="form.allowRegistration" type="checkbox">{{tr('allowRegistration')}}</label><label>{{tr('setupSecret')}}<input v-model="form.secret" type="password" required autocomplete="off"></label><p v-if="error" class="form-error">{{error}}</p><button class="primary wide" :disabled="busy">{{busy?tr('processing'):tr('completeSetup')}}</button></form></div></section></template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useSetupStore } from '../stores/setup'
import { useI18n } from '../i18n'

const props=withDefaults(defineProps<{ flow?: string }>(),{flow:'authentication'})
const emit=defineEmits<{ 'update:modelValue':[value:string] }>()
const setup=useSetupStore(), { tr }=useI18n(), host=ref<HTMLElement|null>(null), error=ref(''), widget=ref<string|number>('')
const provider=computed(()=>setup.status.captchaProvider||'')
function script(url:string, module=false) { return new Promise<void>((resolve,reject)=>{ const old=document.querySelector(`script[src="${url}"]`); if(old){resolve();return};const node=document.createElement('script');node.src=url;node.async=true;if(module)node.type='module';node.onload=()=>resolve();node.onerror=()=>reject(new Error('captcha'));document.head.appendChild(node) }) }
function altchaChallenge(){ return provider.value==='altcha_sentinel' ? setup.status.captchaChallengeUrl||'' : `/api/subflow/v1/auth/captcha/challenge?flow=${encodeURIComponent(props.flow)}` }
async function render(){
  emit('update:modelValue',''); error.value=''; if(!provider.value||!host.value)return
  try {
    host.value.innerHTML=''; const key=setup.status.captchaSiteKey
    if(provider.value==='turnstile'){if(!key)return;await script('https://challenges.cloudflare.com/turnstile/v0/api.js?render=explicit');widget.value=(window as any).turnstile.render(host.value,{sitekey:key,callback:(token:string)=>emit('update:modelValue',token)})}
    else if(provider.value==='hcaptcha'){if(!key)return;await script('https://js.hcaptcha.com/1/api.js?render=explicit');widget.value=(window as any).hcaptcha.render(host.value,{sitekey:key,callback:(token:string)=>emit('update:modelValue',token)})}
    else if(provider.value==='recaptcha'){if(!key)return;await script('https://www.google.com/recaptcha/api.js?render=explicit');widget.value=(window as any).grecaptcha.render(host.value,{sitekey:key,callback:(token:string)=>emit('update:modelValue',token)})}
    else if(provider.value==='altcha_community'||provider.value==='altcha_sentinel'){
      await script('https://cdn.jsdelivr.net/npm/altcha/dist/altcha.js',true)
      const widgetElement=document.createElement('altcha-widget') as HTMLElement
      const challengeURL=altchaChallenge(); if(!challengeURL) throw new Error('challenge')
      widgetElement.setAttribute('challengeurl',challengeURL); widgetElement.setAttribute('name','altcha')
      widgetElement.addEventListener('verified',(event:any)=>emit('update:modelValue',event.detail?.payload||widgetElement.querySelector('input')?.getAttribute('value')||''))
      widgetElement.addEventListener('error',()=>error.value=tr('captchaVerificationFailed')); host.value.appendChild(widgetElement)
    }
  } catch { error.value=tr('captchaVerificationFailed') }
}
onMounted(()=>void render());watch(()=>[setup.status.captchaProvider,setup.status.captchaSiteKey,setup.status.captchaChallengeUrl],()=>void render());onBeforeUnmount(()=>{const api=(window as any)[provider.value==='turnstile'?'turnstile':provider.value==='hcaptcha'?'hcaptcha':'grecaptcha'];if(api&&widget.value!==''&&api.reset)api.reset(widget.value)})
</script>
<template><div v-if="provider" class="captcha-challenge"><div ref="host"/><p v-if="error" class="form-error">{{error}}</p></div></template>

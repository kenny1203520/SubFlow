<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useSetupStore } from '../stores/setup'
import { useI18n } from '../i18n'

const props=withDefaults(defineProps<{ flow?: string; trigger?: 'load'|'submit'; mode?: 'invisible'|'interactive' }>(),{flow:'authentication',trigger:'load',mode:'interactive'})
const emit=defineEmits<{ 'update:modelValue':[value:string] }>()
const setup=useSetupStore(), { tr }=useI18n(), host=ref<HTMLElement|null>(null), error=ref(''), widget=ref<string|number>('')
const provider=computed(()=>setup.status.captchaProvider||'')
// solve() (trigger==='submit') awaits the next resolved/rejected token instead
// of rendering eagerly on mount -- pending holds callers currently waiting.
const pending=ref<Array<{resolve:(token:string)=>void;reject:(err:Error)=>void}>>([])
function settlePending(token:string){ const waiters=pending.value.splice(0); waiters.forEach(w=>token?w.resolve(token):w.reject(new Error('captcha'))) }
function script(url:string, module=false) { return new Promise<void>((resolve,reject)=>{ const old=document.querySelector(`script[src="${url}"]`); if(old){resolve();return};const node=document.createElement('script');node.src=url;node.async=true;if(module)node.type='module';node.onload=()=>resolve();node.onerror=()=>reject(new Error('captcha'));document.head.appendChild(node) }) }
function altchaChallenge(){ return provider.value==='altcha_sentinel' ? setup.status.captchaChallengeUrl||'' : `/api/subflow/v1/auth/captcha/challenge?flow=${encodeURIComponent(props.flow)}` }
async function render(){
  emit('update:modelValue',''); error.value=''; if(!provider.value||!host.value){settlePending('');return}
  try {
    host.value.innerHTML=''; const key=setup.status.captchaSiteKey; const invisible=props.mode==='invisible'
    // Provider APIs verified directly against each vendor's docs -- none of
    // them expose a uniform "invisible" concept:
    //  - Turnstile has no size:'invisible'; its equivalent is the
    //    'appearance' option ('interaction-only' keeps the widget hidden
    //    unless a challenge needs visible interaction). Its default
    //    execution:'render' already auto-runs once rendered, so no manual
    //    .execute() call is needed here.
    //  - hCaptcha/reCAPTCHA v2 DO support size:'invisible', but an
    //    invisible-size widget never fires its callback on its own -- it
    //    requires an explicit .execute(widgetId) call after rendering.
    //  - ALTCHA's auto="onload" only controls WHEN it auto-solves, not
    //    whether it's visible; true invisible mode is CSS display:none plus
    //    revealing the widget only if its state becomes 'code' (a manual
    //    challenge the automatic solve couldn't clear).
    if(provider.value==='turnstile'){if(!key)return;await script('https://challenges.cloudflare.com/turnstile/v0/api.js?render=explicit');widget.value=(window as any).turnstile.render(host.value,{sitekey:key,appearance:invisible?'interaction-only':'always',callback:(token:string)=>{emit('update:modelValue',token);settlePending(token)}})}
    else if(provider.value==='hcaptcha'){if(!key)return;await script('https://js.hcaptcha.com/1/api.js?render=explicit');const hcaptcha=(window as any).hcaptcha;widget.value=hcaptcha.render(host.value,{sitekey:key,size:invisible?'invisible':'normal',callback:(token:string)=>{emit('update:modelValue',token);settlePending(token)}});if(invisible)hcaptcha.execute(widget.value)}
    else if(provider.value==='recaptcha'){if(!key)return;await script('https://www.google.com/recaptcha/api.js?render=explicit');const grecaptcha=(window as any).grecaptcha;widget.value=grecaptcha.render(host.value,{sitekey:key,size:invisible?'invisible':'normal',callback:(token:string)=>{emit('update:modelValue',token);settlePending(token)}});if(invisible)grecaptcha.execute(widget.value)}
    else if(provider.value==='altcha_community'||provider.value==='altcha_sentinel'){
      await script('https://cdn.jsdelivr.net/npm/altcha/dist/altcha.js',true)
      const widgetElement=document.createElement('altcha-widget') as HTMLElement
      const challengeURL=altchaChallenge(); if(!challengeURL) throw new Error('challenge')
      widgetElement.setAttribute('challengeurl',challengeURL); widgetElement.setAttribute('name','altcha')
      // auto='onload' is what makes it solve without a click at all -- only
      // set it for invisible mode. Interactive mode must leave it unset so
      // the checkbox genuinely waits for the visitor to click it.
      if(invisible){ widgetElement.setAttribute('auto','onload'); widgetElement.style.display='none'; widgetElement.addEventListener('statechange',(event:any)=>{ if(event.detail?.state==='code') widgetElement.style.display='' }) }
      widgetElement.addEventListener('verified',(event:any)=>{const token=event.detail?.payload||widgetElement.querySelector('input')?.getAttribute('value')||'';emit('update:modelValue',token);settlePending(token)})
      widgetElement.addEventListener('error',()=>{error.value=tr('captchaVerificationFailed');settlePending('')}); host.value.appendChild(widgetElement)
    }
  } catch { error.value=tr('captchaVerificationFailed'); settlePending('') }
}
function solve(){ return new Promise<string>((resolve,reject)=>{ pending.value.push({resolve,reject}); void render() }) }
defineExpose({ solve })
// A caller can switch which flow this same mounted instance represents (e.g.
// AuthView toggling between login/register) without the component
// unmounting, since v-if stays true across the switch -- so trigger/mode
// changes must be able to re-sync, not just the initial mount.
function sync(){ if(props.trigger==='load') void render(); else { if(host.value)host.value.innerHTML=''; emit('update:modelValue',''); error.value='' } }
onMounted(sync)
watch(()=>[props.flow,props.trigger,props.mode],sync)
watch(()=>[setup.status.captchaProvider,setup.status.captchaSiteKey,setup.status.captchaChallengeUrl],sync)
onBeforeUnmount(()=>{const api=(window as any)[provider.value==='turnstile'?'turnstile':provider.value==='hcaptcha'?'hcaptcha':'grecaptcha'];if(api&&widget.value!==''&&api.reset)api.reset(widget.value)})
</script>
<template><div v-if="provider" class="captcha-challenge"><div ref="host"/><p v-if="error" class="form-error">{{error}}</p></div></template>

<script setup lang="ts">
import { useI18n } from '../i18n'
import { nextTick, onBeforeUnmount, ref, watch } from 'vue'
const props=defineProps<{ open:boolean; title:string }>()
const emit=defineEmits<{ close:[] }>()
const { tr }=useI18n()
const dialog=ref<HTMLDialogElement>()
let previousFocus:HTMLElement|undefined
watch(()=>props.open,async open=>{
 if(open){previousFocus=document.activeElement instanceof HTMLElement?document.activeElement:undefined;await nextTick();dialog.value?.showModal()}
 else{dialog.value?.close();previousFocus?.focus();previousFocus=undefined}
},{immediate:true})
onBeforeUnmount(()=>{dialog.value?.close();previousFocus?.focus()})
</script>
<template><Teleport to="body"><dialog v-if="open" ref="dialog" class="drawer-backdrop" :aria-label="title" @cancel.prevent="emit('close')" @click.self="emit('close')"><section class="app-drawer"><header><h2>{{title}}</h2><button type="button" class="icon-button" :aria-label="tr('close')" @click="emit('close')">×</button></header><div class="drawer-body"><slot /></div></section></dialog></Teleport></template>
<style scoped>
dialog.drawer-backdrop{margin:0;padding:0;border:0;width:100%;max-width:100%;height:100%;max-height:100%;color:inherit}
dialog.drawer-backdrop::backdrop{background:transparent}
</style>

<script setup lang="ts">
import { useI18n } from '../i18n'
defineProps<{ open:boolean; title:string }>()
const emit=defineEmits<{ close:[] }>()
const { tr }=useI18n()
</script>
<template><Teleport to="body"><div v-if="open" class="drawer-backdrop" @click.self="emit('close')"><aside class="app-drawer" role="dialog" aria-modal="true" :aria-label="title"><header><h2>{{title}}</h2><button class="icon-button" :aria-label="tr('close')" @click="emit('close')">×</button></header><div class="drawer-body"><slot /></div></aside></div></Teleport></template>

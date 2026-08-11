<script setup lang="ts">
import { useI18n } from '../i18n'
defineProps<{ open:boolean; title:string; description?:string; danger?:boolean }>()
const emit=defineEmits<{ confirm:[]; cancel:[] }>()
const { tr }=useI18n()
</script>
<template><Teleport to="body"><div v-if="open" class="modal-backdrop" @click.self="emit('cancel')"><section class="confirm-dialog" role="alertdialog" aria-modal="true" :aria-label="title"><h2>{{title}}</h2><p v-if="description">{{description}}</p><div class="form-actions"><button class="ghost" @click="emit('cancel')">{{tr('cancel')}}</button><button class="primary" :class="{danger}" @click="emit('confirm')">{{danger?tr('delete'):tr('confirm')}}</button></div></section></div></Teleport></template>

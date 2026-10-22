<script setup lang="ts">
import { useI18n } from '../i18n'
import { nextTick, onBeforeUnmount, ref, watch } from 'vue'
const props = defineProps<{ open:boolean; title:string; description?:string; danger?:boolean }>()
const emit=defineEmits<{ confirm:[]; cancel:[] }>()
const { tr }=useI18n()
const dialog = ref<HTMLDialogElement>()
let previousFocus: HTMLElement | undefined
watch(() => props.open, async open => {
  if (open) {
    previousFocus = document.activeElement instanceof HTMLElement ? document.activeElement : undefined
    await nextTick()
    dialog.value?.showModal()
    dialog.value?.querySelector<HTMLElement>('button')?.focus()
  } else {
    dialog.value?.close()
    previousFocus?.focus()
    previousFocus = undefined
  }
}, { immediate: true })
onBeforeUnmount(() => { dialog.value?.close(); previousFocus?.focus() })
</script>
<template><Teleport to="body"><dialog v-if="open" ref="dialog" class="modal-backdrop confirm-dialog-backdrop" :aria-label="title" @cancel.prevent="emit('cancel')" @click.self="emit('cancel')"><section class="confirm-dialog" role="alertdialog" aria-modal="true" :aria-label="title"><h2>{{title}}</h2><p v-if="description">{{description}}</p><div class="form-actions"><button type="button" class="ghost" @click="emit('cancel')">{{tr('cancel')}}</button><button type="button" class="primary" :class="{danger}" @click="emit('confirm')">{{danger?tr('delete'):tr('confirm')}}</button></div></section></dialog></Teleport></template>

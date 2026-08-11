<script setup lang="ts">
import { useI18n } from '../i18n'
import { timezoneLabel } from '../timezone'
defineProps<{ open:boolean; savedTimezone:string; currentTimezone:string }>()
const emit=defineEmits<{ update:[]; later:[] }>()
const { tr }=useI18n()
</script>
<template><Teleport to="body"><div v-if="open" class="modal-backdrop" @click.self="emit('later')"><section class="confirm-dialog timezone-mismatch-dialog" role="alertdialog" aria-modal="true" :aria-label="tr('timezoneMismatchTitle')">
  <h2>{{tr('timezoneMismatchTitle')}}</h2>
  <p>{{tr('timezoneMismatchSaved',{timezone:timezoneLabel(savedTimezone)})}}</p>
  <p>{{tr('timezoneMismatchCurrent',{timezone:timezoneLabel(currentTimezone)})}}</p>
  <div class="form-actions"><button class="ghost" @click="emit('later')">{{tr('later')}}</button><button class="primary" @click="emit('update')">{{tr('updateTimezone')}}</button></div>
</section></div></Teleport></template>

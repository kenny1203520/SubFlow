<script setup lang="ts">
import { useToastStore } from '../stores/toast'
import { useI18n } from '../i18n'
const toast = useToastStore()
const { tr } = useI18n()
</script>
<template><Teleport to="body"><div class="toast-container"><div v-for="item in toast.toasts" :key="item.id" class="toast" :class="item.type" role="status"><span>{{item.message}}</span><button type="button" :aria-label="tr('close')" @click="toast.dismiss(item.id)">×</button></div></div></Teleport></template>

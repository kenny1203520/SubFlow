<script setup lang="ts">
import { useRegisterSW } from 'virtual:pwa-register/vue'
import { useI18n } from '../i18n'

const { tr } = useI18n()
const { needRefresh, offlineReady, updateServiceWorker } = useRegisterSW()

function dismiss() { needRefresh.value = false; offlineReady.value = false }
</script>

<template>
  <div v-if="needRefresh || offlineReady" class="pwa-update-prompt">
    <span>{{ needRefresh ? tr('pwaUpdateAvailable') : tr('pwaOfflineReady') }}</span>
    <button v-if="needRefresh" class="primary" @click="updateServiceWorker()">{{ tr('pwaReload') }}</button>
    <button class="ghost" @click="dismiss">{{ tr('close') }}</button>
  </div>
</template>

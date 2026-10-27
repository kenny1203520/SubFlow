<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, ref, watch } from 'vue'
import { useI18n } from '../i18n'
import { useUpgradeStore } from '../stores/upgrade'

const upgrade = useUpgradeStore()
const { tr } = useI18n()
const dialog = ref<HTMLDialogElement>()
const heading = ref<HTMLElement>()
const phaseText = computed(() => {
  if (upgrade.clientUpdateRequired) return tr('upgradeClientDescription')
  if (upgrade.status.state === 'failed') return tr('upgradeFailed')
  if (upgrade.status.state === 'backing_up') return tr('upgradeBackingUp')
  if (upgrade.status.state === 'migrating') return tr('upgradeMigrating')
  if (upgrade.status.state === 'verifying') return tr('upgradeVerifying')
  if (upgrade.status.state === 'ready') return tr('upgradeReconnect')
  return upgrade.checked ? tr('upgradeReconnect') : tr('upgradeChecking')
})

async function syncDialog(blocked: boolean) {
  await nextTick()
  if (!dialog.value) return
  if (blocked) {
    if (!dialog.value.open) dialog.value.showModal()
    await nextTick()
    heading.value?.focus()
  } else if (dialog.value.open) {
    dialog.value.close()
  }
}

watch(() => upgrade.blocked, blocked => { void syncDialog(blocked) }, { immediate: true })
onBeforeUnmount(() => { if (dialog.value?.open) dialog.value.close() })
</script>

<template>
  <dialog ref="dialog" class="upgrade-maintenance-backdrop" aria-labelledby="upgrade-title" aria-describedby="upgrade-description" @cancel.prevent>
    <section class="upgrade-maintenance">
      <div class="upgrade-maintenance-mark" aria-hidden="true"><span></span></div>
      <p class="eyebrow">SubFlow</p>
      <h1 id="upgrade-title" ref="heading" tabindex="-1">{{ upgrade.clientUpdateRequired ? tr('upgradeClientTitle') : tr('upgradeTitle') }}</h1>
      <p id="upgrade-description" class="upgrade-maintenance-description" role="status" aria-live="polite" aria-atomic="true">{{ phaseText }}</p>
      <p v-if="upgrade.status.state === 'failed' && upgrade.status.problemCode" class="upgrade-problem">{{ tr('upgradeProblem', { code: upgrade.status.problemCode }) }}</p>
      <button v-if="upgrade.clientUpdateRequired" class="primary" type="button" @click="upgrade.reloadAfterSaving">{{ tr('upgradeReload') }}</button>
      <p v-else class="upgrade-maintenance-footnote">{{ tr('upgradeDescription') }}</p>
    </section>
  </dialog>
</template>

<script setup lang="ts">
import { useI18n } from '../i18n'
import { currencyLabel } from '../currency'
import { timezoneLabel } from '../timezone'
const props=defineProps<{open:boolean;kind:'expenses'|'subscriptions';group?:{id:string;name:string;currency:string;timezone:string};currency:string}>()
const emit=defineEmits<{close:[]}>();const {tr,locale}=useI18n()
</script>
<template><Teleport to="body"><div v-if="open" class="modal-backdrop" @click.self="emit('close')"><section class="source-dialog" role="dialog" aria-modal="true"><header><h2>{{tr('source')}}</h2><button class="icon-button" :aria-label="tr('close')" @click="emit('close')">×</button></header><div class="source-dialog-body"><template v-if="group"><strong>{{group.name}}</strong><span>{{currencyLabel(currency,locale)}} · {{timezoneLabel(group.timezone)}}</span><RouterLink class="primary" :to="`/groups/${group.id}/${kind}`" @click="emit('close')">{{tr('openGroup')}}</RouterLink></template><template v-else><strong>{{tr('privateRecord')}}</strong><span>{{currencyLabel(currency,locale)}}</span></template></div></section></div></Teleport></template>
